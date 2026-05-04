package trie

import "net"

type Node struct {
	valid bool
	pop   uint16
	scope int
	child [2]*Node
}

type Trie struct {
	root *Node
}

func NewTrie() *Trie {
	return &Trie{&Node{}}
}

func (t *Trie) Insert(subnet *net.IPNet, pop uint16) {
	prefixLength, _ := subnet.Mask.Size()
	bytes := subnet.IP.To16()
	current := t.root

	for i := 0; i < prefixLength; i++ {
		b := bit(bytes, i)
		if current.child[b] == nil {
			current.child[b] = &Node{}
		}
		current = current.child[b]
	}
	current.pop = pop
	current.valid = true
	current.scope = prefixLength
}

func (t *Trie) Route(esc *net.IPNet) (pop uint16, scope int) {
	prefixLength, _ := esc.Mask.Size()
	bytes := esc.IP.To16()
	current := t.root

	for i := 0; i < prefixLength; i++ {
		b := bit(bytes, i)
		current = current.child[b]

		if current == nil {
			return
		}

		if current.valid {
			pop = current.pop
			scope = current.scope
		}
	}
	return
}

func bit(ip net.IP, pos int) int {
	return int(ip[pos/8] >> (7 - pos%8) & 1)
}
