// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import (
	"fmt"
	"sort"
	"strconv"

	"yunion.io/x/onecloud/pkg/util/tagutils"
)

const (
	tagValueCountKey = "__count__"
)

func TagValueKey(idx int) string {
	return fmt.Sprintf("value%d", idx)
}

type sTagValueTreeNode struct {
	Key   string           `json:"key"`
	Value string           `json:"value"`
	Count float64          `json:"count"`
	Tags  tagutils.TTagSet `json:"tags"`

	Children []*sTagValueTreeNode `json:"children"`
}

type sTagValueTreeNodeChildren []*sTagValueTreeNode

func (a sTagValueTreeNodeChildren) Len() int      { return len(a) }
func (a sTagValueTreeNodeChildren) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sTagValueTreeNodeChildren) Less(i, j int) bool {
	if a[i].Value == tagutils.NoValue {
		return false
	}
	if a[j].Value == tagutils.NoValue {
		return true
	}
	if a[i].Count > a[j].Count {
		return true
	} else if a[i].Count < a[j].Count {
		return false
	}
	return a[i].Value < a[j].Value
}

func (node *sTagValueTreeNode) findChild(key, value string) *sTagValueTreeNode {
	for _, child := range node.Children {
		if child.Key == key && child.Value == value {
			return child
		}
	}
	child := &sTagValueTreeNode{
		Key:   key,
		Value: value,
	}
	node.Children = append(node.Children, child)
	return child
}

func (node *sTagValueTreeNode) getTag() tagutils.STag {
	return tagutils.STag{
		Key:   node.Key,
		Value: node.Value,
	}
}

func (node *sTagValueTreeNode) populateTags() {
	sort.Sort(sTagValueTreeNodeChildren(node.Children))
	for _, child := range node.Children {
		if len(node.Tags) > 0 {
			child.Tags = append(child.Tags, node.Tags...)
		}
		child.Tags = append(child.Tags, child.getTag())
		child.populateTags()
	}
}

func constructTree(data []map[string]string, keys []string) *sTagValueTreeNode {
	root := &sTagValueTreeNode{}
	for i := range data {
		processOneRow(root, data[i], keys)
	}
	root.populateTags()
	return root
}

func processOneRow(node *sTagValueTreeNode, row map[string]string, keys []string) {
	rowCount, _ := strconv.ParseFloat(row[tagValueCountKey], 64)
	node.Count += rowCount
	for i := range keys {
		key := keys[i]
		value := row[TagValueKey(i)]
		child := node.findChild(key, value)
		child.Count += rowCount
		node = child
	}
}
