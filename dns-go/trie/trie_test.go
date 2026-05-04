package trie

import (
	"net"
	"testing"
)

func TestBit(t *testing.T) {
	ip := net.ParseIP("2001::").To16()

	if bit(ip, 0) != 0 {
		t.Errorf("bit 0: expected 0, got %d", bit(ip, 0))
	}
	if bit(ip, 2) != 1 {
		t.Errorf("bit 2: expected 1, got %d", bit(ip, 2))
	}
	if bit(ip, 15) != 1 {
		t.Errorf("bit 15: expected 1, got %d", bit(ip, 15))
	}
}

func TestTrieMatch(t *testing.T) {
	trie := NewTrie()
	_, subnet, _ := net.ParseCIDR("2001:49f0:d0b8::/48")

	trie.Insert(subnet, 174)

	pop, scope := trie.Route(subnet)
	if pop != 174 {
		t.Errorf("pop expected 174, got %d", pop)
	}
	if scope != 48 {
		t.Errorf("scope expected 48, got %d", scope)
	}
}

func TestTrieLongestPrefixMatch(t *testing.T) {
	trie := NewTrie()
	_, subnet, _ := net.ParseCIDR("2001:49f0::/32")
	_, subnet2, _ := net.ParseCIDR("2001:49f0:d0b8:8a00::/48")
	_, subnet3, _ := net.ParseCIDR("2001:49f0:d0b8:8a00::/56")

	trie.Insert(subnet, 99)
	trie.Insert(subnet2, 174)

	pop, scope := trie.Route(subnet3)
	if pop != 174 {
		t.Errorf("pop expected 174, got %d", pop)
	}
	if scope != 48 {
		t.Errorf("scope expected 48, got %d", scope)
	}
}

func TestTrieNoMatch(t *testing.T) {
	trie := NewTrie()
	_, subnet, _ := net.ParseCIDR("2001:49f0:d0b8::/48")
	_, subnet2, _ := net.ParseCIDR("2002:49f0:dfb8::/22")

	trie.Insert(subnet, 174)

	pop, scope := trie.Route(subnet2)
	if pop != 0 {
		t.Errorf("pop expected 0, got %d", pop)
	}
	if scope != 0 {
		t.Errorf("scope expected 0, got %d", scope)
	}
}
