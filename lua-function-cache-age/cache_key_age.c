#include <sys/stat.h>
#include <time.h>

long cache_key_age(const char *path) {
    struct stat st;
    if (stat(path, &st) != 0) {
        return -1;
    }
    return (long) (time(NULL) - st.st_mtime);
}