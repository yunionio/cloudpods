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

// iterStack represents a stack of (node, pos) tuples, which captures
// iteration state as an Iterator descends a AugBTree.
type iterStack[K, V, A any] struct {
	a    iterStackArr[K, V, A]
	aLen int16 // -1 when using s
	s    []iterFrame[K, V, A]
}

const iterStackDepth = 10

// Used to avoid allocations for stacks below a certain size.
type iterStackArr[K, V, A any] [iterStackDepth]iterFrame[K, V, A]

type iterFrame[K, V, A any] struct {
	node *Node[K, V, A]
	pos  int16
}

func (is *iterStack[K, V, A]) push(f iterFrame[K, V, A]) {
	if is.aLen == -1 {
		is.s = append(is.s, f)
	} else if int(is.aLen) == len(is.a) {
		is.s = make([](iterFrame[K, V, A]), int(is.aLen)+1, 2*int(is.aLen))
		copy(is.s, is.a[:])
		is.s[int(is.aLen)] = f
		is.aLen = -1
	} else {
		is.a[is.aLen] = f
		is.aLen++
	}
}

func (is *iterStack[K, V, A]) pop() iterFrame[K, V, A] {
	if is.aLen == -1 {
		f := is.s[len(is.s)-1]
		is.s = is.s[:len(is.s)-1]
		return f
	}
	is.aLen--
	return is.a[is.aLen]
}

func (is *iterStack[K, V, A]) len() int {
	if is.aLen == -1 {
		return len(is.s)
	}
	return int(is.aLen)
}

func (is *iterStack[K, V, A]) reset() {
	if is.aLen == -1 {
		is.s = is.s[:0]
	} else {
		is.aLen = 0
	}
}
