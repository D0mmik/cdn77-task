package trie

import (
	"math/bits"
	"net"
)

type Node struct {
	valid  bool
	pop    uint16
	scope  int
	skip   int
	prefix net.IP
	child  [2]*Node
}

type Trie struct {
	root *Node
}

func NewTrie() *Trie {
	return &Trie{&Node{}}
}

func (t *Trie) Insert(subnet *net.IPNet, pop uint16) {
	prefixLength, _ := subnet.Mask.Size()
	addr := subnet.IP.To16()

	current := t.root
	offset := 0
	var parent *Node
	var lastBit int

	for {
		sharedBits := commonPrefixLen(addr, current.prefix, current.skip)

		if sharedBits >= current.skip {
			offset += current.skip
			b := bit(addr, offset)
			parent = current
			lastBit = b
			current = current.child[b]

			if current == nil {
				parent.child[lastBit] = &Node{
					prefix: addr,
					skip:   prefixLength - offset,
					pop:    pop,
					valid:  true,
					scope:  prefixLength,
				}
				return
			}
		} else {
			splitNode(parent, current, lastBit, offset, sharedBits, prefixLength, addr, pop)
			return
		}
	}
}

func (t *Trie) Route(ecs *net.IPNet) (pop uint16, scope int) {
	prefixLength, _ := ecs.Mask.Size()
	addr := ecs.IP.To16()
	current := t.root
	offset := 0

	for {
		b := bit(addr, offset)
		current = current.child[b]

		if current == nil {
			return
		}

		sharedBits := commonPrefixLen(addr, current.prefix, current.skip)
		if sharedBits < current.skip {
			return
		}

		if current.valid {
			pop = current.pop
			scope = current.scope
		}

		offset += current.skip + 1

		if offset > prefixLength {
			return
		}
	}
}

func splitNode(parent, current *Node, lastBit, offset, sharedBits, prefixLength int, addr net.IP, pop uint16) {
	bitNew := bit(addr, offset+sharedBits)
	bitOld := bit(current.prefix, offset+sharedBits)

	mid := &Node{
		prefix: addr,
		skip:   sharedBits,
	}

	parent.child[lastBit] = mid

	current.skip = current.skip - sharedBits - 1
	mid.child[bitOld] = current

	mid.child[bitNew] = &Node{
		prefix: addr,
		skip:   prefixLength - offset - sharedBits - 1,
		pop:    pop,
		valid:  true,
		scope:  prefixLength,
	}
}

func bit(ip net.IP, pos int) int {
	return int(ip[pos/8] >> (7 - pos%8) & 1)
}

func commonPrefixLen(a, b net.IP, maxLen int) int {
	for i := 0; i < maxLen/8; i++ {
		xor := a[i] ^ b[i]

		if xor != 0 {
			return i*8 + bits.LeadingZeros8(xor)
		}
	}
	return maxLen
}
