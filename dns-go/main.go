package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/D0mmik/cdn77-task/dns-go/trie"
)

func main() {
	data, err := loadRouting("data/routing-data.txt")
	if err != nil {
		log.Fatal(err)
	}

	queries := []string{
		"2001:49f0:d0b8:1::/56",
		"2402:8100:2582:1::/56",
		"240e:438:1e30:f::/48",
		"2a02:26f7:ded0:1::/56",
		"2404:0:2000:f::/48",
		"2409:8904:2480:f::/48",
		"2409:8d80:a001::/48",
		"2804:1c1c:3000:1::/56",
		"2600:9000:2096:1::/56",
		"2605:c3c0:e001::/48",
		"ffff::/16",
	}

	for _, query := range queries {
		_, ecs, _ := net.ParseCIDR(query)
		pop, scope := data.Route(ecs)
		fmt.Printf("ECS: %s → PoP %d, scope /%d\n", ecs, pop, scope)
	}

}

func loadRouting(filename string) (*trie.Trie, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	data := trie.NewTrie()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		subnet, pop, err := parseLine(fields)
		if err != nil {
			return nil, err
		}

		data.Insert(subnet, pop)
	}
	return data, nil
}

func parseLine(fields []string) (*net.IPNet, uint16, error) {
	_, subnet, err := net.ParseCIDR(fields[0])
	if err != nil {
		return nil, 0, err
	}
	pop, err := strconv.ParseUint(fields[1], 10, 16)
	if err != nil {
		return nil, 0, err
	}

	return subnet, uint16(pop), nil
}
