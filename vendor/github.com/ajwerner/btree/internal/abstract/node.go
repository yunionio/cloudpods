// Copyright 2018 The Cockroach Authors.
// Copyright 2021 Andrew Werner.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package abstract

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Node represents a node in the tree.
type Node[K, V, A any] struct {
	ref      int32
	count    int16
	aug      A
	keys     [MaxEntries]K
	values   [MaxEntries]V
	children *[MaxEntries + 1]*Node[K, V, A]
}

type interiorNode[K, V, A any] struct {
	Node[K, V, A]
	children [MaxEntries + 1]*Node[K, V, A]
}

func (n *Node[K, V, A]) GetA() *A {
	return &n.aug
}

func (n *Node[K, V, A]) IsLeaf() bool {
	return n.children == nil
}

func (n *Node[K, V, A]) Count() int16 {
	return n.count
}

func (n *Node[K, V, A]) GetKey(i int16) K {
	return n.keys[i]
}

func (n *Node[K, V, A]) GetChild(i int16) *A {
	if !n.IsLeaf() && n.children[i] != nil {
		return &n.children[i].aug
	}
	return nil
}

// mut creates and returns a mutable node reference. If the node is not shared
// with any other trees then it can be modified in place. Otherwise, it must be
// cloned to ensure unique ownership. In this way, we enforce a copy-on-write
// policy which transparently incorporates the idea of local mutations, like
// Clojure's transients or Haskell's ST monad, where nodes are only copied
// during the first time that they are modified between Clone operations.
//
// When a node is cloned, the provided pointer will be redirected to the new
// mutable node.
func mut[K, V, A any](
	np *nodePool[K, V, A],
	n **Node[K, V, A],
) *Node[K, V, A] {
	if atomic.LoadInt32(&(*n).ref) == 1 {
		// Exclusive ownership. Can mutate in place.
		return *n
	}
	// If we do not have unique ownership over the node then we
	// clone it to gain unique ownership. After doing so, we can
	// release our reference to the old node. We pass recursive
	// as true because even though we just observed the node's
	// reference count to be greater than 1, we might be racing
	// with another call to decRef on this node.
	c := (*n).clone(np)
	(*n).decRef(np, true /* recursive */)
	*n = c
	return *n
}

// incRef acquires a reference to the node.
func (n *Node[K, V, A]) incRef() {
	atomic.AddInt32(&n.ref, 1)
}

// decRef releases a reference to the node. If requested, the method
// will recurse into child nodes and decrease their refcounts as well.
func (n *Node[K, V, A]) decRef(
	np *nodePool[K, V, A], recursive bool,
) {
	if atomic.AddInt32(&n.ref, -1) > 0 {
		// Other references remain. Can't free.
		return
	}
	// Clear and release node into memory pool.
	if n.IsLeaf() {
		np.putLeafNode(n)
	} else {
		// Release child references first, if requested.
		if recursive {
			for i := int16(0); i <= n.count; i++ {
				n.children[i].decRef(np, true /* recursive */)
			}
		}
		np.putInteriorNode(n)
	}
}

// clone creates a clone of the receiver with a single reference count.
func (n *Node[K, V, A]) clone(
	np *nodePool[K, V, A],
) *Node[K, V, A] {
	var c *Node[K, V, A]
	if n.IsLeaf() {
		c = np.getLeafNode()
	} else {
		c = np.getInteriorNode()
	}
	// NB: copy field-by-field without touching N.N.ref to avoid
	// triggering the race detector and looking like a data race.
	c.count = n.count
	n.aug = c.aug
	c.keys = n.keys
	if !c.IsLeaf() {
		// Copy children and increase each refcount.
		*c.children = *n.children
		for i := int16(0); i <= c.count; i++ {
			c.children[i].incRef()
		}
	}
	return c
}

func (n *Node[K, V, A]) insertAt(index int, item K, value V, nd *Node[K, V, A]) {
	if index < int(n.count) {
		copy(n.keys[index+1:n.count+1], n.keys[index:n.count])
		copy(n.values[index+1:n.count+1], n.values[index:n.count])
		if !n.IsLeaf() {
			copy(n.children[index+2:n.count+2], n.children[index+1:n.count+1])
		}
	}
	n.keys[index] = item
	n.values[index] = value
	if !n.IsLeaf() {
		n.children[index+1] = nd
	}
	n.count++
}

func (n *Node[K, V, A]) pushBack(item K, value V, nd *Node[K, V, A]) {
	n.keys[n.count] = item
	n.values[n.count] = value
	if !n.IsLeaf() {
		n.children[n.count+1] = nd
	}
	n.count++
}

func (n *Node[K, V, A]) pushFront(item K, value V, nd *Node[K, V, A]) {
	if !n.IsLeaf() {
		copy(n.children[1:n.count+2], n.children[:n.count+1])
		n.children[0] = nd
	}
	copy(n.keys[1:n.count+1], n.keys[:n.count])
	copy(n.values[1:n.count+1], n.values[:n.count])
	n.keys[0] = item
	n.values[0] = value
	n.count++
}

// removeAt removes a value at a given index, pulling all subsequent values
// back.
func (n *Node[K, V, A]) removeAt(index int) (K, V, *Node[K, V, A]) {
	var child *Node[K, V, A]
	if !n.IsLeaf() {
		child = n.children[index+1]
		copy(n.children[index+1:n.count], n.children[index+2:n.count+1])
		n.children[n.count] = nil
	}
	n.count--
	outK := n.keys[index]
	outV := n.values[index]
	copy(n.keys[index:n.count], n.keys[index+1:n.count+1])
	copy(n.values[index:n.count], n.values[index+1:n.count+1])
	var rk K
	var rv V
	n.keys[n.count] = rk
	n.values[n.count] = rv
	return outK, outV, child
}

// popBack removes and returns the last element in the list.
func (n *Node[K, V, A]) popBack() (K, V, *Node[K, V, A]) {
	n.count--
	outK := n.keys[n.count]
	outV := n.values[n.count]
	var rK K
	var rV V
	n.keys[n.count] = rK
	n.values[n.count] = rV
	if n.IsLeaf() {
		return outK, outV, nil
	}
	child := n.children[n.count+1]
	n.children[n.count+1] = nil
	return outK, outV, child
}

// popFront removes and returns the first element in the list.
func (n *Node[K, V, A]) popFront() (K, V, *Node[K, V, A]) {
	n.count--
	var child *Node[K, V, A]
	if !n.IsLeaf() {
		child = n.children[0]
		copy(n.children[:n.count+1], n.children[1:n.count+2])
		n.children[n.count+1] = nil
	}
	outK := n.keys[0]
	outV := n.values[0]
	copy(n.keys[:n.count], n.keys[1:n.count+1])
	copy(n.values[:n.count], n.values[1:n.count+1])
	var rK K
	var rV V
	n.keys[n.count] = rK
	n.values[n.count] = rV
	return outK, outV, child
}

// find returns the index where the given item should be inserted into this
// list. 'found' is true if the item already exists in the list at the given
// index.
func (n *Node[K, V, A]) find(cmp func(K, K) int, item K) (index int, found bool) {
	// Logic copied from sort.Search. Inlining this gave
	// an 11% speedup on BenchmarkBTreeDeleteInsert.
	i, j := 0, int(n.count)
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		c := cmp(item, n.keys[h])
		if c < 0 {
			j = h
		} else if c > 0 {
			i = h + 1
		} else {
			return h, true
		}
	}
	return i, false
}

// split splits the given node at the given index. The current node shrinks,
// and this function returns the item that existed at that index and a new
// node containing all keys/children after it.
//
// Before:
//
//          +-----------+
//          |   x y z   |
//          +--/-/-\-\--+
//
// After:
//
//          +-----------+
//          |     y     |
//          +----/-\----+
//              /   \
//             v     v
// +-----------+     +-----------+
// |         x |     | z         |
// +-----------+     +-----------+
//
func (n *Node[K, V, A]) split(cfg *config[K, V, A], i int) (K, V, *Node[K, V, A]) {
	outK := n.keys[i]
	outV := n.values[i]
	var next *Node[K, V, A]
	if n.IsLeaf() {
		next = cfg.np.getLeafNode()
	} else {
		next = cfg.np.getInteriorNode()
	}
	next.count = n.count - int16(i+1)
	copy(next.keys[:], n.keys[i+1:n.count])
	copy(next.values[:], n.values[i+1:n.count])
	var rK K
	var rV V
	for j := int16(i); j < n.count; j++ {
		n.keys[j] = rK
		n.values[j] = rV
	}
	if !n.IsLeaf() {
		copy(next.children[:], n.children[i+1:n.count+1])
		for j := int16(i + 1); j <= n.count; j++ {
			n.children[j] = nil
		}
	}
	n.count = int16(i)
	next.update(&cfg.Config)
	n.updateOn(&cfg.Config, Split, outK, next)
	return outK, outV, next
}

func (n *Node[K, V, A]) update(cfg *Config[K, V, A]) bool {
	return n.updateWithMeta(cfg, UpdateInfo[K, A]{})
}

func (n *Node[K, V, A]) updateWithMeta(cfg *Config[K, V, A], md UpdateInfo[K, A]) bool {
	if cfg.Updater == nil {
		return false
	}
	return cfg.Updater.Update(n, md)
}

func (n *Node[K, V, A]) updateOn(cfg *Config[K, V, A], action Action, k K, affected *Node[K, V, A]) bool {
	if cfg.Updater == nil {
		return false
	}
	var a *A
	if affected != nil {
		a = &affected.aug
	}
	return n.updateWithMeta(cfg, UpdateInfo[K, A]{
		Action:        action,
		RelevantKey:   k,
		ModifiedOther: a,
	})
}

// insert inserts an item into the suAugBTree rooted at this node, making sure no
// nodes in the suAugBTree exceed MaxEntries keys. Returns true if an existing item
// was replaced and false if an item was inserted. Also returns whether the
// node's upper bound changes.
func (n *Node[K, V, A]) insert(cfg *config[K, V, A], item K, value V) (replacedK K, replacedV V, replaced, newBound bool) {
	i, found := n.find(cfg.cmp, item)
	if found {
		replacedV = n.values[i]
		replacedK = n.keys[i]
		n.keys[i] = item
		n.values[i] = value
		return replacedK, replacedV, true, false
	}
	if n.IsLeaf() {
		n.insertAt(i, item, value, nil)
		return replacedK, replacedV, false, n.updateOn(&cfg.Config, Insertion, item, nil)
	}
	if n.children[i].count >= MaxEntries {
		splitLK, splitLV, splitNode := mut(cfg.np, &n.children[i]).
			split(cfg, MaxEntries/2)
		n.insertAt(i, splitLK, splitLV, splitNode)
		if c := cfg.cmp(item, n.keys[i]); c < 0 {
			// no change, we want first split node
		} else if c > 0 {
			i++ // we want second split node
		} else {
			// TODO(ajwerner): add something to the augmentation api to
			// deal with replacement.
			replacedV = n.values[i]
			replacedK = n.keys[i]
			n.keys[i] = item
			n.values[i] = value
			return replacedK, replacedV, true, false
		}
	}
	replacedK, replacedV, replaced, newBound =
		mut(cfg.np, &n.children[i]).insert(cfg, item, value)
	if newBound {
		newBound = n.updateOn(&cfg.Config, Insertion, item, nil)
	}
	return replacedK, replacedV, replaced, newBound
}

// removeMax removes and returns the maximum item from the suAugBTree rooted at
// this node.
func (n *Node[K, V, A]) removeMax(cfg *config[K, V, A]) (K, V) {
	if n.IsLeaf() {
		n.count--
		outK := n.keys[n.count]
		outV := n.values[n.count]
		var rK K
		var rV V
		n.keys[n.count] = rK
		n.values[n.count] = rV
		n.updateOn(&cfg.Config, Removal, outK, nil)
		return outK, outV
	}
	// Recurse into max child.
	i := int(n.count)
	if n.children[i].count <= MinEntries {
		// Child not large enough to remove from.
		n.rebalanceOrMerge(cfg, i)
		return n.removeMax(cfg) // redo
	}
	child := mut(cfg.np, &n.children[i])
	outK, outV := child.removeMax(cfg)
	n.updateOn(&cfg.Config, Removal, outK, nil)
	return outK, outV
}

// rebalanceOrMerge grows child 'i' to ensure it has sufficient room to remove
// an item from it while keeping it at or above MinItems.
func (n *Node[K, V, A]) rebalanceOrMerge(
	cfg *config[K, V, A], i int,
) {
	switch {
	case i > 0 && n.children[i-1].count > MinEntries:
		// Rebalance from left sibling.
		//
		//          +-----------+
		//          |     y     |
		//          +----/-\----+
		//              /   \
		//             v     v
		// +-----------+     +-----------+
		// |         x |     |           |
		// +----------\+     +-----------+
		//             \
		//              v
		//              a
		//
		// After:
		//
		//          +-----------+
		//          |     x     |
		//          +----/-\----+
		//              /   \
		//             v     v
		// +-----------+     +-----------+
		// |           |     | y         |
		// +-----------+     +/----------+
		//                   /
		//                  v
		//                  a
		//
		left := mut(cfg.np, &n.children[i-1])
		child := mut(cfg.np, &n.children[i])
		xLaK, xLaV, grandChild := left.popBack()
		yLaK, yLaV := n.keys[i-1], n.values[i-1]
		child.pushFront(yLaK, yLaV, grandChild)
		n.keys[i-1], n.values[i-1] = xLaK, xLaV
		left.updateOn(&cfg.Config, Removal, xLaK, grandChild)
		child.updateOn(&cfg.Config, Insertion, yLaK, grandChild)

	case i < int(n.count) && n.children[i+1].count > MinEntries:
		// Rebalance from right sibling.
		//
		//          +-----------+
		//          |     y     |
		//          +----/-\----+
		//              /   \
		//             v     v
		// +-----------+     +-----------+
		// |           |     | x         |
		// +-----------+     +/----------+
		//                   /
		//                  v
		//                  a
		//
		// After:
		//
		//          +-----------+
		//          |     x     |
		//          +----/-\----+
		//              /   \
		//             v     v
		// +-----------+     +-----------+
		// |         y |     |           |
		// +----------\+     +-----------+
		//             \
		//              v
		//              a
		//
		right := mut(cfg.np, &n.children[i+1])
		child := mut(cfg.np, &n.children[i])
		xLaK, xLaV, grandChild := right.popFront()
		yLaK, yLaV := n.keys[i], n.values[i]
		child.pushBack(yLaK, yLaV, grandChild)
		n.keys[i], n.values[i] = xLaK, xLaV
		right.updateOn(&cfg.Config, Removal, xLaK, grandChild)
		child.updateOn(&cfg.Config, Insertion, yLaK, grandChild)

	default:
		// Merge with either the left or right sibling.
		//
		//          +-----------+
		//          |   u y v   |
		//          +----/-\----+
		//              /   \
		//             v     v
		// +-----------+     +-----------+
		// |         x |     | z         |
		// +-----------+     +-----------+
		//
		// After:
		//
		//          +-----------+
		//          |    u v    |
		//          +-----|-----+
		//                |
		//                v
		//          +-----------+
		//          |   x y z   |
		//          +-----------+
		//
		if i >= int(n.count) {
			i = int(n.count - 1)
		}
		child := mut(cfg.np, &n.children[i])
		// Make mergeChild mutable, bumping the refcounts on its children if necessary.
		_ = mut(cfg.np, &n.children[i+1])
		mergeLaK, mergeLaV, mergeChild := n.removeAt(i)
		child.keys[child.count] = mergeLaK
		child.values[child.count] = mergeLaV
		copy(child.keys[child.count+1:], mergeChild.keys[:mergeChild.count])
		copy(child.values[child.count+1:], mergeChild.values[:mergeChild.count])
		if !child.IsLeaf() {
			copy(child.children[child.count+1:], mergeChild.children[:mergeChild.count+1])
		}
		child.count += mergeChild.count + 1

		child.updateOn(&cfg.Config, Insertion, mergeLaK, mergeChild)
		mergeChild.decRef(cfg.np, false /* recursive */)
	}
}

// remove removes an item from the suAugBTree rooted at this node. Returns the item
// that was removed or nil if no matching item was found. Also returns whether
// the node's upper bound changes.
func (n *Node[K, V, A]) remove(
	cfg *config[K, V, A], item K,
) (outK K, outV V, found, newBound bool) {
	i, found := n.find(cfg.cmp, item)
	if n.IsLeaf() {
		if found {
			outK, outV, _ = n.removeAt(i)
			return outK, outV, true, n.updateOn(&cfg.Config, Removal, outK, nil)
		}
		var rK K
		var rV V
		return rK, rV, false, false
	}
	if n.children[i].count <= MinEntries {
		// Child not large enough to remove from.
		n.rebalanceOrMerge(cfg, i)
		return n.remove(cfg, item) // redo
	}
	child := mut(cfg.np, &n.children[i])
	if found {
		// Replace the item being removed with the max item in our left child.
		outK = n.keys[i]
		outV = n.values[i]
		n.keys[i], n.values[i] = child.removeMax(cfg)
		return outK, outV, true, n.updateOn(&cfg.Config, Removal, outK, nil)
	}
	// Latch is not in this node and child is large enough to remove from.
	outK, outV, found, newBound = child.remove(cfg, item)
	if newBound {
		newBound = n.updateOn(&cfg.Config, Removal, outK, nil)
	}
	return outK, outV, found, newBound
}

func (n *Node[K, V, A]) writeString(b *strings.Builder) {
	if n.IsLeaf() {
		for i := int16(0); i < n.count; i++ {
			if i != 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(b, "%v:%v", n.keys[i], n.values[i])
		}
		return
	}
	for i := int16(0); i <= n.count; i++ {
		b.WriteString("(")
		n.children[i].writeString(b)
		b.WriteString(")")
		if i < n.count {
			fmt.Fprintf(b, "%v:%v", n.keys[i], n.values[i])
		}
	}
}
