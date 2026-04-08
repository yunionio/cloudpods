// Package numtree implements radix tree data structure for 32 and 64-bit unsigned integets. Copy-on-write is used for any tree modification so old root doesn't see any change happens with tree.
package numtree

import (
	"fmt"
	"math/bits"
)

// Key32BitSize is an alias for bitsize of 32-bit radix tree's key.
const Key32BitSize = 32

var (
	masks32 = []uint32{
		0x00000000, 0x80000000, 0xc0000000, 0xe0000000,
		0xf0000000, 0xf8000000, 0xfc000000, 0xfe000000,
		0xff000000, 0xff800000, 0xffc00000, 0xffe00000,
		0xfff00000, 0xfff80000, 0xfffc0000, 0xfffe0000,
		0xffff0000, 0xffff8000, 0xffffc000, 0xffffe000,
		0xfffff000, 0xfffff800, 0xfffffc00, 0xfffffe00,
		0xffffff00, 0xffffff80, 0xffffffc0, 0xffffffe0,
		0xfffffff0, 0xfffffff8, 0xfffffffc, 0xfffffffe,
		0xffffffff}
)

// Node32 is an element of radix tree with 32-bit unsigned integer as a key.
type Node32 struct {
	// Key stores key for current node.
	Key uint32
	// Bits is a number of significant bits in Key.
	Bits uint8
	// Leaf indicates if the node is leaf node and contains any data in Value.
	Leaf bool
	// Value contains data associated with key.
	Value interface{}

	chld [2]*Node32
}

// Dot dumps tree to Graphviz .dot format
func (n *Node32) Dot() string {
	body := ""

	// Iterate all nodes using breadth-first search algorithm.
	i := 0
	queue := []*Node32{n}
	for len(queue) > 0 {
		c := queue[0]
		body += fmt.Sprintf("N%d %s\n", i, c.dotString())
		if c != nil && (c.chld[0] != nil || c.chld[1] != nil) {
			// Children for current node if any always go to the end of the queue
			// so we can know their indices using current queue length.
			body += fmt.Sprintf("N%d -> { N%d N%d }\n", i, i+len(queue), i+len(queue)+1)
			queue = append(append(queue, c.chld[0]), c.chld[1])
		}

		queue = queue[1:]
		i++
	}

	return "digraph d {\n" + body + "}\n"
}

// Insert puts new leaf to radix tree and returns pointer to new root. The method uses copy on write strategy so old root doesn't see the change.
func (n *Node32) Insert(key uint32, bits int, value interface{}) *Node32 {
	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key32BitSize {
		bits = Key32BitSize
	}

	return n.insert(newNode32(key, uint8(bits), true, value))
}

// InplaceInsert puts new leaf to radix tree (or replaces value in existing one). The method inserts data directly to current tree so make sure you have exclusive access to it.
func (n *Node32) InplaceInsert(key uint32, bits int, value interface{}) *Node32 {
	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key32BitSize {
		bits = Key32BitSize
	}

	return n.inplaceInsert(key, uint8(bits), value)
}

// Enumerate returns channel which is populated by nodes with data in order of their keys.
func (n *Node32) Enumerate() chan *Node32 {
	ch := make(chan *Node32)

	go func() {
		defer close(ch)

		// If tree is empty -
		if n == nil {
			// return nothing.
			return
		}

		n.enumerate(ch)
	}()

	return ch
}

// Match locates node which key is equal to or "contains" the key passed as argument.
func (n *Node32) Match(key uint32, bits int) (interface{}, bool) {
	// If tree is empty -
	if n == nil {
		// report nothing.
		return n, false
	}

	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key32BitSize {
		bits = Key32BitSize
	}

	r := n.match(key, uint8(bits))
	if r == nil {
		return nil, false
	}

	return r.Value, true
}

// ExactMatch locates node which exactly matches given key.
func (n *Node32) ExactMatch(key uint32, bits int) (interface{}, bool) {
	// If tree is empty -
	if n == nil {
		// report nothing.
		return n, false
	}

	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key32BitSize {
		bits = Key32BitSize
	}

	r := n.exactMatch(key, uint8(bits))
	if r == nil {
		return nil, false
	}

	return r.Value, true
}

// Delete removes subtree which is contained by given key. The method uses copy on write strategy.
func (n *Node32) Delete(key uint32, bits int) (*Node32, bool) {
	// If tree is empty -
	if n == nil {
		// report nothing.
		return n, false
	}

	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key32BitSize {
		bits = Key32BitSize
	}

	return n.del(key, uint8(bits))
}

func (n *Node32) dotString() string {
	if n == nil {
		return "[label=\"nil\"]"
	}

	if n.Leaf {
		v := fmt.Sprintf("%q", fmt.Sprintf("%#v", n.Value))
		return fmt.Sprintf("[label=\"k: %08x, b: %d, v: \\\"%s\\\"\"]", n.Key, n.Bits, v[1:len(v)-1])
	}

	return fmt.Sprintf("[label=\"k: %08x, b: %d\"]", n.Key, n.Bits)
}

func (n *Node32) insert(c *Node32) *Node32 {
	if n == nil {
		return c
	}

	// Find number of common most significant bits (NCSB):
	// 1. xor operation puts zeroes at common bits;
	// 2. or masks put ones so that zeroes can't go after smaller number of significant bits (NSB)
	// 3. count of leading zeroes gives number of common bits
	bits := uint8(bits.LeadingZeros32((n.Key ^ c.Key) | ^masks32[n.Bits] | ^masks32[c.Bits]))

	// There are three cases possible:
	// - NCSB less than number of significant bits (NSB) of current tree node:
	if bits < n.Bits {
		// (branch for current tree node is determined by a bit right after the last common bit)
		branch := (n.Key >> (Key32BitSize - 1 - bits)) & 1

		// - NCSB equals to NSB of candidate node:
		if bits == c.Bits {
			// make new root from the candidate and put current node to one of its branch;
			c.chld[branch] = n
			return c
		}

		// - NCSB less than NSB of candidate node (it can't be greater because bits after NSB don't count):
		// make new root (non-leaf node)
		m := newNode32(c.Key&masks32[bits], bits, false, nil)
		// with current tree node at one of branches
		m.chld[branch] = n
		// and the candidate at the other.
		m.chld[1-branch] = c

		return m
	}

	// - keys are equal (NCSB not less than NSB of current tree node and both numbers are equal):
	if c.Bits == n.Bits {
		// replace current node with the candidate.
		c.chld = n.chld
		return c
	}

	// - current tree node contains candidate node:
	// make new root as a copy of current tree node;
	m := newNode32(n.Key, n.Bits, n.Leaf, n.Value)
	m.chld = n.chld

	// (branch for the candidate is determined by a bit right after the last common bit)
	branch := (c.Key >> (Key32BitSize - 1 - bits)) & 1
	// insert it to correct branch.
	m.chld[branch] = m.chld[branch].insert(c)

	return m
}

func (n *Node32) inplaceInsert(key uint32, sbits uint8, value interface{}) *Node32 {
	var (
		p      *Node32
		branch uint32
	)

	r := n

	for n != nil {
		cbits := uint8(bits.LeadingZeros32((n.Key ^ key) | ^masks32[n.Bits] | ^masks32[sbits]))
		if cbits < n.Bits {
			pBranch := branch
			branch = (n.Key >> (Key32BitSize - 1 - cbits)) & 1

			var m *Node32

			if cbits == sbits {
				m = newNode32(key, sbits, true, value)
				m.chld[branch] = n
			} else {
				m = newNode32(key&masks32[cbits], cbits, false, nil)
				m.chld[1-branch] = newNode32(key, sbits, true, value)
			}

			m.chld[branch] = n
			if p == nil {
				r = m
			} else {
				p.chld[pBranch] = m
			}

			return r
		}

		if sbits == n.Bits {
			n.Key = key
			n.Leaf = true
			n.Value = value
			return r
		}

		p = n
		branch = (key >> (Key32BitSize - 1 - cbits)) & 1
		n = n.chld[branch]
	}

	n = newNode32(key, sbits, true, value)
	if p == nil {
		return n
	}

	p.chld[branch] = n
	return r
}

func (n *Node32) enumerate(ch chan *Node32) {
	// Implemented by depth-first search.
	if n.Leaf {
		ch <- n
	}

	if n.chld[0] != nil {
		n.chld[0].enumerate(ch)
	}

	if n.chld[1] != nil {
		n.chld[1].enumerate(ch)
	}
}

func (n *Node32) match(key uint32, bits uint8) *Node32 {
	// If can't be contained in current root node -
	if n.Bits > bits {
		// report nothing.
		return nil
	}

	// If NSB of current tree node is the same as key has -
	if n.Bits == bits {
		// return current node only if it contains data (leaf node) and masked keys are equal.
		if n.Leaf && (n.Key^key)&masks32[n.Bits] == 0 {
			return n
		}

		return nil
	}

	// If key can be contained by current tree node -
	if (n.Key^key)&masks32[n.Bits] != 0 {
		// but it isn't report nothing.
		return nil
	}

	// Otherwise jump to branch by key bit right after NSB of current tree node
	c := n.chld[(key>>(Key32BitSize-1-n.Bits))&1]
	if c != nil {
		// and check if child on the branch has anything.
		r := c.match(key, bits)
		if r != nil {
			return r
		}
	}

	// If nothing matches check if current node contains any data.
	if n.Leaf {
		return n
	}

	return nil
}

func (n *Node32) exactMatch(key uint32, bits uint8) *Node32 {
	// If can't be contained in current root node -
	if n.Bits > bits {
		// report nothing.
		return nil
	}

	// If NSB of current tree node is the same as key has -
	if n.Bits == bits {
		// return current node only if it contains data (leaf node) and masked keys are equal.
		if n.Leaf && (n.Key^key)&masks32[n.Bits] == 0 {
			return n
		}

		return nil
	}

	// If key can be contained by current tree node -
	if (n.Key^key)&masks32[n.Bits] != 0 {
		// but it isn't report nothing.
		return nil
	}

	// Otherwise jump to branch by key bit right after NSB of current tree node
	c := n.chld[(key>>(Key32BitSize-1-n.Bits))&1]
	if c != nil {
		// and check if child on the branch has anything.
		r := c.exactMatch(key, bits)
		if r != nil {
			return r
		}
	}

	return nil
}

func (n *Node32) del(key uint32, bits uint8) (*Node32, bool) {
	// If key can contain current tree node -
	if bits <= n.Bits {
		// report empty new tree and put deletion mark if it contains indeed.
		if (n.Key^key)&masks32[bits] == 0 {
			return nil, true
		}

		return n, false
	}

	// If key can be contained by current tree node -
	if (n.Key^key)&masks32[n.Bits] != 0 {
		// but it isn't report nothing.
		return n, false
	}

	// Otherwise jump to branch by key bit right after NSB of current tree node
	branch := (key >> (Key32BitSize - 1 - n.Bits)) & 1
	c := n.chld[branch]
	if c == nil {
		// report nothing if the branch is empty.
		return n, false
	}

	// Try to remove from subtree
	c, ok := c.del(key, bits)
	if !ok {
		// and report nothing if nothing has been deleted.
		return n, false
	}

	// If child of non-leaf node has been completely deleted -
	if c == nil && !n.Leaf {
		// drop the node.
		return n.chld[1-branch], true
	}

	// If deletion happens inside the branch then copy current node.
	m := newNode32(n.Key, n.Bits, n.Leaf, n.Value)
	m.chld = n.chld

	// Replace changed child with new one and return new root with deletion mark set.
	m.chld[branch] = c
	return m, true
}

func newNode32(key uint32, bits uint8, leaf bool, value interface{}) *Node32 {
	return &Node32{
		Key:   key,
		Bits:  bits,
		Leaf:  leaf,
		Value: value}
}
