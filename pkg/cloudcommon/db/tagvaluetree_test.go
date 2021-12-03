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
	"reflect"
	"sort"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func TestTagValueTreeNodeSort(t *testing.T) {
	cases := []struct {
		nodes []*sTagValueTreeNode
		want  []*sTagValueTreeNode
	}{
		{
			nodes: []*sTagValueTreeNode{
				&sTagValueTreeNode{
					Value: tagutils.NoValue,
					Count: 100,
				},
				&sTagValueTreeNode{
					Value: "丽水市",
					Count: 10,
				},
				&sTagValueTreeNode{
					Value: "义务市",
					Count: 10,
				},
				&sTagValueTreeNode{
					Value: "绍兴市",
					Count: 100,
				},
			},
			want: []*sTagValueTreeNode{
				&sTagValueTreeNode{
					Value: "绍兴市",
					Count: 100,
				},
				&sTagValueTreeNode{
					Value: "丽水市",
					Count: 10,
				},
				&sTagValueTreeNode{
					Value: "义务市",
					Count: 10,
				},
				&sTagValueTreeNode{
					Value: tagutils.NoValue,
					Count: 100,
				},
			},
		},
	}
	for i, c := range cases {
		sort.Sort(sTagValueTreeNodeChildren(c.nodes))
		if !reflect.DeepEqual(c.nodes, c.want) {
			t.Errorf("order not match %d got %s", i, jsonutils.Marshal(c.nodes))
		}
	}
}
