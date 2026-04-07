package numtree

import (
	"fmt"
	"math/bits"
)

// Key64BitSize is an alias for bitsize of 64-bit radix tree's key.
const Key64BitSize = 64

var (
	masks64 = []uint64{
		0x0000000000000000, 0x8000000000000000, 0xc000000000000000, 0xe000000000000000,
		0xf000000000000000, 0xf800000000000000, 0xfc00000000000000, 0xfe00000000000000,
		0xff00000000000000, 0xff80000000000000, 0xffc0000000000000, 0xffe0000000000000,
		0xfff0000000000000, 0xfff8000000000000, 0xfffc000000000000, 0xfffe000000000000,
		0xffff000000000000, 0xffff800000000000, 0xffffc00000000000, 0xffffe00000000000,
		0xfffff00000000000, 0xfffff80000000000, 0xfffffc0000000000, 0xfffffe0000000000,
		0xffffff0000000000, 0xffffff8000000000, 0xffffffc000000000, 0xffffffe000000000,
		0xfffffff000000000, 0xfffffff800000000, 0xfffffffc00000000, 0xfffffffe00000000,
		0xffffffff00000000, 0xffffffff80000000, 0xffffffffc0000000, 0xffffffffe0000000,
		0xfffffffff0000000, 0xfffffffff8000000, 0xfffffffffc000000, 0xfffffffffe000000,
		0xffffffffff000000, 0xffffffffff800000, 0xffffffffffc00000, 0xffffffffffe00000,
		0xfffffffffff00000, 0xfffffffffff80000, 0xfffffffffffc0000, 0xfffffffffffe0000,
		0xffffffffffff0000, 0xffffffffffff8000, 0xffffffffffffc000, 0xffffffffffffe000,
		0xfffffffffffff000, 0xfffffffffffff800, 0xfffffffffffffc00, 0xfffffffffffffe00,
		0xffffffffffffff00, 0xffffffffffffff80, 0xffffffffffffffc0, 0xffffffffffffffe0,
		0xfffffffffffffff0, 0xfffffffffffffff8, 0xfffffffffffffffc, 0xfffffffffffffffe,
		0xffffffffffffffff}
)

// Node64 is an element of radix tree with 64-bit unsigned integer as a key.
type Node64 struct {
	// Key stores key for current node.
	Key uint64
	// Bits is a number of significant bits in Key.
	Bits uint8
	// Leaf indicates if the node is leaf node and contains any data in Value.
	Leaf bool
	// Value contains data associated with key.
	Value interface{}

	chld [2]*Node64
}

// Dot dumps tree to Graphviz .dot format
func (n *Node64) Dot() string {
	body := ""

	i := 0
	queue := []*Node64{n}
	for len(queue) > 0 {
		c := queue[0]
		body += fmt.Sprintf("N%d %s\n", i, c.dotString())

		if c != nil && (c.chld[0] != nil || c.chld[1] != nil) {
			body += fmt.Sprintf("N%d -> { N%d N%d }\n", i, i+len(queue), i+len(queue)+1)
			queue = append(append(queue, c.chld[0]), c.chld[1])
		}

		queue = queue[1:]
		i++
	}

	return "digraph d {\n" + body + "}\n"
}

// Insert puts new leaf to radix tree and returns pointer to new root. The method uses copy on write strategy so old root doesn't see the change.
func (n *Node64) Insert(key uint64, bits int, value interface{}) *Node64 {
	if bits < 0 {
		bits = 0
	} else if bits > Key64BitSize {
		bits = Key64BitSize
	}

	return n.insert(newNode64(key, uint8(bits), true, value))
}

// InplaceInsert puts new leaf to radix tree (or replaces value in existing one). The method inserts data directly to current tree so make sure you have exclusive access to it.
func (n *Node64) InplaceInsert(key uint64, bits int, value interface{}) *Node64 {
	// Adjust bits.
	if bits < 0 {
		bits = 0
	} else if bits > Key64BitSize {
		bits = Key64BitSize
	}

	return n.inplaceInsert(key, uint8(bits), value)
}

// Enumerate returns channel which is populated by nodes in order of their keys.
func (n *Node64) Enumerate() chan *Node64 {
	ch := make(chan *Node64)

	go func() {
		defer close(ch)

		if n == nil {
			return
		}

		n.enumerate(ch)
	}()

	return ch
}

// Match locates node which key is equal to or "contains" the key passed as argument.
func (n *Node64) Match(key uint64, bits int) (interface{}, bool) {
	if n == nil {
		return n, false
	}

	if bits < 0 {
		bits = 0
	} else if bits > Key64BitSize {
		bits = Key64BitSize
	}

	r := n.match(key, uint8(bits))
	if r == nil {
		return nil, false
	}

	return r.Value, true
}

// ExactMatch locates node which exactly matches given key.
func (n *Node64) ExactMatch(key uint64, bits int) (interface{}, bool) {
	if n == nil {
		return n, false
	}

	if bits < 0 {
		bits = 0
	} else if bits > Key64BitSize {
		bits = Key64BitSize
	}

	r := n.exactMatch(key, uint8(bits))
	if r == nil {
		return nil, false
	}

	return r.Value, true
}

// Delete removes subtree which is contained by given key. The method uses copy on write strategy.
func (n *Node64) Delete(key uint64, bits int) (*Node64, bool) {
	if n == nil {
		return n, false
	}

	if bits < 0 {
		bits = 0
	} else if bits > Key64BitSize {
		bits = Key64BitSize
	}

	return n.del(key, uint8(bits))
}

func (n *Node64) dotString() string {
	if n == nil {
		return "[label=\"nil\"]"
	}

	if n.Leaf {
		v := fmt.Sprintf("%q", fmt.Sprintf("%#v", n.Value))
		return fmt.Sprintf("[label=\"k: %016x, b: %d, v: \\\"%s\\\"\"]", n.Key, n.Bits, v[1:len(v)-1])
	}

	return fmt.Sprintf("[label=\"k: %016x, b: %d\"]", n.Key, n.Bits)
}

func (n *Node64) insert(c *Node64) *Node64 {
	if n == nil {
		return c
	}

	bits := uint8(bits.LeadingZeros64((n.Key ^ c.Key) | ^masks64[n.Bits] | ^masks64[c.Bits]))
	if bits < n.Bits {
		branch := (n.Key >> (Key64BitSize - 1 - bits)) & 1
		if bits == c.Bits {
			c.chld[branch] = n
			return c
		}

		m := newNode64(c.Key&masks64[bits], bits, false, nil)
		m.chld[branch] = n
		m.chld[1-branch] = c

		return m
	}

	if c.Bits == n.Bits {
		c.chld = n.chld
		return c
	}

	m := newNode64(n.Key, n.Bits, n.Leaf, n.Value)
	m.chld = n.chld

	branch := (c.Key >> (Key64BitSize - 1 - bits)) & 1
	m.chld[branch] = m.chld[branch].insert(c)

	return m
}

func (n *Node64) inplaceInsert(key uint64, sbits uint8, value interface{}) *Node64 {
	var (
		p      *Node64
		branch uint64
	)

	r := n

	for n != nil {
		cbits := uint8(bits.LeadingZeros64((n.Key ^ key) | ^masks64[n.Bits] | ^masks64[sbits]))
		if cbits < n.Bits {
			pBranch := branch
			branch = (n.Key >> (Key64BitSize - 1 - cbits)) & 1

			var m *Node64

			if cbits == sbits {
				m = newNode64(key, sbits, true, value)
				m.chld[branch] = n
			} else {
				m = newNode64(key&masks64[cbits], cbits, false, nil)
				m.chld[1-branch] = newNode64(key, sbits, true, value)
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
		branch = (key >> (Key64BitSize - 1 - cbits)) & 1
		n = n.chld[branch]
	}

	n = newNode64(key, sbits, true, value)
	if p == nil {
		return n
	}

	p.chld[branch] = n
	return r
}

func (n *Node64) enumerate(ch chan *Node64) {
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

func (n *Node64) match(key uint64, bits uint8) *Node64 {
	if n.Bits > bits {
		return nil
	}

	if n.Bits == bits {
		if n.Leaf && (n.Key^key)&masks64[n.Bits] == 0 {
			return n
		}

		return nil
	}

	if (n.Key^key)&masks64[n.Bits] != 0 {
		return nil
	}

	c := n.chld[(key>>(Key64BitSize-1-n.Bits))&1]
	if c != nil {
		r := c.match(key, bits)
		if r != nil {
			return r
		}
	}

	if n.Leaf {
		return n
	}

	return nil
}

func (n *Node64) exactMatch(key uint64, bits uint8) *Node64 {
	if n.Bits > bits {
		return nil
	}

	if n.Bits == bits {
		if n.Leaf && (n.Key^key)&masks64[n.Bits] == 0 {
			return n
		}

		return nil
	}

	if (n.Key^key)&masks64[n.Bits] != 0 {
		return nil
	}

	c := n.chld[(key>>(Key64BitSize-1-n.Bits))&1]
	if c != nil {
		r := c.exactMatch(key, bits)
		if r != nil {
			return r
		}
	}

	return nil
}

func (n *Node64) del(key uint64, bits uint8) (*Node64, bool) {
	if bits <= n.Bits {
		if (n.Key^key)&masks64[bits] == 0 {
			return nil, true
		}

		return n, false
	}

	if (n.Key^key)&masks64[n.Bits] != 0 {
		return n, false
	}

	branch := (key >> (Key64BitSize - 1 - n.Bits)) & 1
	c := n.chld[branch]
	if c == nil {
		return n, false
	}

	c, ok := c.del(key, bits)
	if !ok {
		return n, false
	}

	if c == nil && !n.Leaf {
		return n.chld[1-branch], true
	}

	m := newNode64(n.Key, n.Bits, n.Leaf, n.Value)
	m.chld = n.chld

	m.chld[branch] = c
	return m, true
}

func newNode64(key uint64, bits uint8, leaf bool, value interface{}) *Node64 {
	return &Node64{
		Key:   key,
		Bits:  bits,
		Leaf:  leaf,
		Value: value}
}
