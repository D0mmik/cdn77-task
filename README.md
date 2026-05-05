# 1 NGINX
Nginx jsem znal jen z konfigurace, ne z pohledu interních struktur.  Jelikož mě zajímá, jak věci detailně fungují, tak mě tenhle task zaujal.

### 1.1 Začátek

Abych pochopil big picture nginx, začal jsem studovat architekturu kódu od počáteční funkce `main`, s tím že se zaměřím na direktivu `proxy_cache*` a metodu `create_cache_key`

Funkce `ngx_init_cycle()`, která parsuje config, parsuje sockety a vytváří sdílenou paměť, vytvoří nový cycle nebo vrátí NULL, díky tomu reload nikdy neshodí server kvůli chybnému configu. V `ngx_conf_parse()` se přímo rekurzivně rozparsuje `nginx.conf` do konfiguračních structů jednotlivých modulů (podle tokenu ze souboru najde příkaz v commands tabulkách a zavolá handler).

Pokud je v konfiguraci podobná konfigurace:

    proxy_cache_path /tmp/nginx_cache levels=1:2 keys_zone=my_cache:10m;
    location / {
            proxy_pass  http://backend;
            proxy_cache my_cache;
        }
        
`proxy_cache_path` zaregistruje sdílenou paměť (`my_cache`, 10MB) a cestu na disku. `ngx_init_cycle` ji po doparsování namapuje přes mmap - sdílenou mezi všemi worker procesy.

### 1.2 Každý request - vytvoření cache klíče
`ngx_http_proxy_handler()` přiřadí `u->create_key = ngx_http_proxy_create_key`, následně `u->create_key(r)` sestaví string klíče, default je (scheme+host+uri).

Funkce  `ngx_http_file_cache_create_key(r)` vezme všechny stringy v `r->cache->keys` a z nich spočítá dva hashe paralelně:

**CRC32**  (`c->crc32`): rychlý hash, používá se pro rychlé porovnání v sdílené paměti. Než nginx sáhne na disk, porovná CRC32  pokud nesedí, ani nezkoušet.

**MD5** (`c->key`) - 16 bajtů, pomalější ale kolizně odolnější. Slouží jako:
-   jméno souboru na disku (např. `/tmp/nginx_cache/4/2b/60d2b3d297...`)
-   klíč v hash tabulce v sdílené paměti

Z pohledu OS jsou cache klíče a metadata uloženy v anonymním mmap regionu (`MAP_SHARED | MAP_ANON`) - všechny worker procesy sdílejí stejné stránky virtuální paměti. Samotná těla cache souborů jsou na disku, typicky držená v page cache OS.

### 1.3 X-Cache-Key
Klíč máme uložený v `r->cache->main`, takže bylo potřeba zjistit, kde se do response přidávají hlavičky. Obě cesty Cache HIT i MISS jdou přes funkci `ngx_http_send_header(r)`, takže bylo potřeba hlavičku přidat před tuto funkci, proto jsem ji přidal hned za vytvoření klíče.

[Odkaz na commit: X-Cache-Key](https://github.com/D0mmik/cdn77-task/commit/3dc660a7ab1db51b367505e8d6d26e8241083a14)

Zjistil jsem, jak se přidávají ostatní hlavičky a podle toho jsem implementoval i moji funkci.

Funkce hex-enkóduje 16 bajtů MD5 hashe z `r->cache->main` do 32 znaků a přidá je jako `X-Cache-Key` do
 `r->headers_out.headers`. 
 `hash = 1` zajistí že ji nginx nevyfiltruje, binární MD5 musí být hex-enkódované protože HTTP hlavičky musí být text.

### 1.4 Lua funkce pro získání jak dlouho je soubor uložen v cache
Jako bonusový úkol jsem implementoval funkci, která navazuje na předchozí tasky. Podle X-Cache-Key víme cestu k souboru v cache na disku, proto jsem v C napsal funkci která vrátí kolik sekund je soubor starý.

`stat(2)` syscall vrátí `mtime` souboru, `time(NULL) - mtime` je stáří v sekundách.

[Odkaz na commit: C funkce](https://github.com/D0mmik/cdn77-task/commit/fd71a14af2a558e90a766a15d57eecf8eb5620f9)

![Praktická ukázka C funkce](https://github.com/user-attachments/assets/95cd91e2-2a9c-40b8-a624-c29be3f4c536)


### 1.5 Vysvětlení algoritmu pro vyhodnocování platnosti DNS wildcard záznamů
DNS wildcard v hostname může být pouze na první nebo poslední pozici - nelze mít záznamy typu `aaaa.*.bbbb.cc`

Struct `ngx_hash_combined_t` obsahuje tři hash tabulky:

-   `hash` - přesná shoda
-   `wc_head` - wildcard na začátku (`*.aaaa.bbbb.cc`)
-   `wc_tail` - wildcard na konci (`aaaa.bbbb.*`)

Při načtení konfigurace se každý vzor uloží do odpovídající tabulky s odříznutým wildcardem jako klíč

Funkce `ngx_hash_find_combined` při vyhodnocování příchozího jména postupně zkouší všechny tři tabulky a vrátí první nalezenou shodu. U wildcard tabulek postupně odřezává labely (zleva pro `wc_head`, zprava pro `wc_tail`) a každý zkrácený tvar hledá v hash tabulce.

Počet lookup operací nezávisí na počtu uložených záznamů `n`, ale pouze na počtu labelů příchozího jména oddělených tečkami - což je v praxi konstanta. Každý jednotlivý lookup v hash tabulce je O(1) v průměrném případě.

**Výsledná časová složitost vyhodnocení je tedy O(1) vzhledem k n.** 
I při miliardách uložených záznamů funkce provede jen několik desítek hash lookupů.

### 1.6 Kde řešení není optimální a co bych změnil pro produkci
`mtime` není spolehlivý - nginx metadata souboru aktualizuje, takže `mtime` nemusí odpovídat času uložení do cache.

`X-Cache-Key` leakuje informaci - MD5 hash je URL struktury backendu. Pro produkci bych přidal možnost pro vypnutí, defaultně off nebo bych to rovnou ani neimplementoval. Zbytečná informace pro klienty.

### 1.7  Na čem jsem se zasekl
Hlavně při pochopení architektury commandů a modulů a následně kde přesně přidat hlavičku tak, aby fungovala u HIT i MISS - pomohlo dohledání volání od `ngx_http_send_header` zpět.

# 2 DNS Go

### 2.1 Začátek
Task mě zaujal, jelikož jsem měl předešlé zkušenosti v Go a téma mi přišlo zajímavé.

### 2.2 Problém a volba datové struktury
Začal jsem přemýšlet nad rychlým způsobem vyhledávání. Došlo mi, že nemůžu využít běžný algoritmus pro vyhledávání, jako například binary search, protože není cíl najít přesnou shodu, ale pouze nejdelší shodný prefix.

Po delším zkoumání různých algoritmů, hlavně v IP adresách a celkově počítačových sítí, jsem zjistil, že musím využít stromovou strukturu, konkrétně speciální typ trie. 

### 2.3 Základní binární trie

Trie je stromová datová struktura, kde každá hrana představuje část klíče - v mém případě jeden bit. Každý uzel má tedy dva potomky (0 a 1) podle konkrétního bitu IPv6 adresy převedené do binární podoby. Při vyhledávání procházím strom od začátku bit po bitu a průběžně si zaznamenávám poslední uzel, který byl označen jako platný prefix, to zajišťuje pole `valid` v každém uzlu a pak vrátím nejdelší nalezenou shodu.

Vyhledávání v triích je konstantní konkrétně O(128), jelikož IPv6 adresa má 16 bajtů.  Časová složitost je optimální, ale prostorová náročnost optimální není. Binární trie může mít hodně uzlů a hodně z nich má jen jednoho potomka a tvoří dlouhé řetězce bez větvení.

### 2.4 Patricia trie - optimalizace

Začal jsem proto hledat způsob optimalizace a našel jsem Patricia trie (radix tree), která tyto řetězce komprimuje do jedné hrany s vícebitovým labelem. V mojí implementaci to odpovídá poli `skip` v každém uzlu, které říká kolik bitů lze přeskočit najednou. Délku sdíleného prefixu počítá funkce `commonPrefixLen` pomocí XOR bajt po bajtu a `bits.LeadingZeros8` pro přesné určení prvního rozdílného bitu. Pokud při vkládání nový prefix sdílí jen část aktuální hrany, funkce `splitNode` hranu rozdělí na dva uzly. Tím se výrazně sníží počet uzlů a paměť, aniž by se zhoršila složitost vyhledávání.

[Odkaz na implementaci DNS v Go](https://github.com/D0mmik/cdn77-task/tree/main/dns-go)

### 2.5 Kde řešení není optimální a co bych změnil pro produkci

#### LC-Trie
Jako možné další rozšíření by bylo možné využít LC-Trie (Level-Compressed Trie), která kromě vertikální komprese Patricia trie přidává i kompresi horizontální. Pro tenhle task jsem zůstal u Patricia trie, která nabízí dobrý kompromis mezi efektivitou a jednoduchostí implementace, ale LC-Trie by byla dalším krokem při škálování řešení.

LC-Trie prostorovou efektivitu nezlepšuje - vytváří vícebitové uzly s 2^k pointry, čímž obětuje paměť za plošší strom a rychlejší lookup na reálném hardwaru.

#### Další vylepšení
`net.IP` alokuje na heapu - slice overhead + heap alokace. V produkci bych použil `[16]byte` přímo v uzlu a tím bych se vyhnul i `To16()` při každém `Route`, zbytečná alokace na lookup.

Souběžné `Insert` + `Route` je race condition. V produkci buď RWMutex, nebo copy-on-write.

### 2.6 Na čem jsem se zasekl 
Nejvíc práce dala `splitNode` a `commonPrefixLen` na hranicích bajtů. Pomohlo napsání unit testů a manuální procházení stromu na papíře.


# 3 Závěr

Naučil jsem se toho opravdu hodně a proto mě to bavilo. Některé věci, jako například komprimované trie nebo jak konkrétně funguje nginx jsem viděl poprvé a díky tomu si o těhle a podobných tématech chci zjišťovat ještě více.

Všechny potřebné repozitáře jsou součástí mého repozitáře přes git subtrees, 
takže by mělo stačit `git clone` bez nutnosti stahovat cokoliv zvlášť.

### 3.1 Časová náročnost

Jednotlivé úkoly mi přišly náročné a zabraly poměrně hodně času.

### 3.1.1 nginx

U prvního úkolu mi určitě nejvíc zabral research, celkové zjišťování toho, jak nginx funguje, jak je rozdělen na moduly, volání commandů při parsování konfigurace. Potom co jsem získal přehled a našel konkrétní funkce, tak už to šlo rychleji.

**Celkově mi task zabral:**
6h research a zjišťování, jak co funguje
2h implementace
1h napojení Lua + bonusová funkce  

### 3.1.2 DNS Go

U druhého úkolu mi taky nejvíc zabral research, hlavně trie a pak ještě víc času komprimované trie.
 
**Celkově mi task zabral:**
5h research a zjišťování, jak co funguje
2h implementace trie
3h implementace komprimované trie