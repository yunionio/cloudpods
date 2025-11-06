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

import "sync"

type nodePool[K, V, A any] struct {
	interiorNodePool, leafNodePool sync.Pool
}

var syncPoolMap sync.Map

func getNodePool[K, V, A any]() *nodePool[K, V, A] {
	var nilNode *Node[K, V, A]
	v, ok := syncPoolMap.Load(nilNode)
	if !ok {
		v, _ = syncPoolMap.LoadOrStore(nilNode, newNodePool[K, V, A]())
	}
	return v.(*nodePool[K, V, A])

}

func newNodePool[K, V, A any]() *nodePool[K, V, A] {
	np := nodePool[K, V, A]{}
	np.leafNodePool = sync.Pool{
		New: func() interface{} {
			return new(Node[K, V, A])
		},
	}
	np.interiorNodePool = sync.Pool{
		New: func() interface{} {
			n := new(interiorNode[K, V, A])
			n.Node.children = &n.children
			return &n.Node
		},
	}
	return &np
}

func (np *nodePool[K, V, A]) getInteriorNode() *Node[K, V, A] {
	n := np.interiorNodePool.Get().(*Node[K, V, A])
	n.ref = 1
	return n
}

func (np *nodePool[K, V, A]) getLeafNode() *Node[K, V, A] {
	n := np.leafNodePool.Get().(*Node[K, V, A])
	n.ref = 1
	return n
}

func (np *nodePool[K, V, A]) putInteriorNode(n *Node[K, V, A]) {
	children := n.children
	*children = [MaxEntries + 1]*Node[K, V, A]{}
	*n = Node[K, V, A]{}
	n.children = children
	np.interiorNodePool.Put(n)
}

func (np *nodePool[K, V, A]) putLeafNode(n *Node[K, V, A]) {
	*n = Node[K, V, A]{}
	np.leafNodePool.Put(n)
}
