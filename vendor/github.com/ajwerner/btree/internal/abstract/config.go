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

// Config is used to configure the tree. It consists of a comparison function
// for keys and any auxiliary data provided by the instantiator. It is provided
// on the iterator and passed to the augmentation's Update method.
type Config[K, V, A any] struct {

	// Updater is used to update the augmentations to the tree.
	Updater Updater[K, V, A]

	cmp func(K, K) int
}

// Updater is used to update the augmentation of the node when the subtree
// changes.
type Updater[K, V, A any] interface {

	// Update should update the augmentation of the passed node, optionally
	// using the data in the UpdataMeta to optimize the update. If the
	// augmentation changed, and thus, changes should occur in the ancestors
	// of the subtree rooted at this node, return true.
	Update(*Node[K, V, A], UpdateInfo[K, A]) (changed bool)
}

// UpdateInfo is used to describe the update operation.
type UpdateInfo[K, A any] struct {

	// Action indicates the semantics of the below fields. If Default, no
	// fields will be populated.
	Action Action

	// ModifiedOther is the augmentation of a node which was either a previous
	// child (Removal), new child (Insertion), or represents the new
	// right-hand-side after a split.
	ModifiedOther *A

	// RelevantKey will be populated in all non-Default events.
	RelevantKey K
}

// Action is used to classify the type of Update in order to permit various
// optimizations when updated the augmented state.
type Action int

const (

	// Default implies that no assumptions may be made with regards to the
	// change in state of the node and thus the augmented state should be
	// recalculated in full.
	Default Action = iota

	// Split indicates that this node is the left-hand side of a split.
	// The ModifiedOther will correspond to the updated state of the
	// augmentation for the right-hand side and the RelevantKey is the split
	// key to be moved into the parent.
	Split

	// Removal indicates that this is a removal event. If the RelevantNode is
	// populated, it indicates a rebalance which caused the node rooted
	// at that subtree to also be removed.
	Removal

	// Insertion indicates that this is a insertion event. If the RelevantNode is
	// populated, it indicates a rebalance which caused the node rooted
	// at that subtree to also be added.
	Insertion
)

// Compare compares two values using the same comparison function as the Map.
func (c *Config[K, V, A]) Compare(a, b K) int { return c.cmp(a, b) }

type config[K, V, A any] struct {
	Config[K, V, A]
	np *nodePool[K, V, A]
}

func makeConfig[K, V, A any](
	cmp func(K, K) int, up Updater[K, V, A],
) (c config[K, V, A]) {
	c.Updater = up
	c.cmp = cmp
	c.np = getNodePool[K, V, A]()
	return c
}
