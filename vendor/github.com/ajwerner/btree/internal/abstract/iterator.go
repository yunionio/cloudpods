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

// Iterator is responsible for search and traversal within a AugBTree.
type Iterator[K, V, A any] struct {
	r *Map[K, V, A]
	iterFrame[K, V, A]
	s iterStack[K, V, A]
}

func (i *Iterator[K, V, A]) lowLevel() *LowLevelIterator[K, V, A] {
	return (*LowLevelIterator[K, V, A])(i)
}

// Compare compares two keys using the same comparison function as the map.
func (i *Iterator[K, V, A]) Compare(a, b K) int {
	return i.r.cfg.cmp(a, b)
}

// Reset marks the iterator as invalid and clears any state.
func (i *Iterator[K, V, A]) Reset() {
	i.node = i.r.root
	i.pos = -1
	i.s.reset()
}

// SeekGE seeks to the first key greater-than or equal to the provided
// key.
func (i *Iterator[K, V, A]) SeekGE(key K) {
	i.Reset()
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	for {
		pos, found := i.node.find(i.r.cfg.cmp, key)
		i.pos = int16(pos)
		if found {
			return
		}
		if i.node.IsLeaf() {
			if i.pos == i.node.count {
				i.Next()
			}
			return
		}
		ll.Descend()
	}
}

// SeekLT seeks to the first key less-than the provided key.
func (i *Iterator[K, V, A]) SeekLT(key K) {
	i.Reset()
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	for {
		pos, found := i.node.find(i.r.cfg.cmp, key)
		i.pos = int16(pos)
		if found || i.node.IsLeaf() {
			i.Prev()
			return
		}
		ll.Descend()
	}
}

// First seeks to the first key in the AugBTree.
func (i *Iterator[K, V, A]) First() {
	i.Reset()
	i.pos = 0
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	for !i.node.IsLeaf() {
		ll.Descend()
	}
	i.pos = 0
}

// Last seeks to the last key in the AugBTree.
func (i *Iterator[K, V, A]) Last() {
	i.Reset()
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	for !i.node.IsLeaf() {
		i.pos = i.node.count
		ll.Descend()
	}
	i.pos = i.node.count - 1
}

// Next positions the Iterator to the key immediately following
// its current position.
func (i *Iterator[K, V, A]) Next() {
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	if i.node.IsLeaf() {
		i.pos++
		if i.pos < i.node.count {
			return
		}
		for i.s.len() > 0 && i.pos >= i.node.count {
			ll.Ascend()
		}
		return
	}
	i.pos++
	ll.Descend()
	for !i.node.IsLeaf() {
		i.pos = 0
		ll.Descend()
	}
	i.pos = 0
}

// Prev positions the Iterator to the key immediately preceding
// its current position.
func (i *Iterator[K, V, A]) Prev() {
	if i.node == nil {
		return
	}
	ll := i.lowLevel()
	if i.node.IsLeaf() {
		i.pos--
		if i.pos >= 0 {
			return
		}
		for i.s.len() > 0 && i.pos < 0 {
			ll.Ascend()
			i.pos--
		}
		return
	}

	ll.Descend()
	for !i.node.IsLeaf() {
		i.pos = i.node.count
		ll.Descend()
	}
	i.pos = i.node.count - 1
}

// Valid returns whether the Iterator is positioned at a valid position.
func (i *Iterator[K, V, A]) Valid() bool {
	return i.node != nil && i.pos >= 0 && i.pos < i.node.count
}

// Cur returns the key at the Iterator's current position. It is illegal
// to call Key if the Iterator is not valid.
func (i *Iterator[K, V, A]) Cur() K {
	return i.node.keys[i.pos]
}

// Value returns the value at the Iterator's current position. It is illegal
// to call Value if the Iterator is not valid.
func (i *Iterator[K, V, A]) Value() V {
	return i.node.values[i.pos]
}
