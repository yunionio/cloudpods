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

package k8s

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ResourceManager struct {
	*modulebase.ResourceManager
}

func NewResourceManager(keyword, keywordPlural string, columns, adminColumns *Columns) *ResourceManager {
	man := &modulebase.ResourceManager{
		BaseManager:   *modulebase.NewBaseManager("k8s", "", "", columns.Array(), adminColumns.Array()),
		Keyword:       keyword,
		KeywordPlural: keywordPlural,
	}
	return &ResourceManager{man}
}

type ClusterResourceManager struct {
	*ResourceManager
}

func NewClusterResourceManager(keyword, keywordPlural string, columns, adminColumns *Columns) *ClusterResourceManager {
	newAdminCols := NewClusterCols(adminColumns.Array()...)
	man := NewResourceManager(keyword, keywordPlural, columns, newAdminCols)
	return &ClusterResourceManager{man}
}

func (man ClusterResourceManager) GetCluster(obj jsonutils.JSONObject) interface{} {
	cluster, _ := obj.GetString("cluster")
	return cluster
}

type MetaResourceManager struct {
	*ClusterResourceManager
	nameGetter
	ageGetter
	labelGetter
}

func NewMetaResourceManager(kw, kwp string, columns, adminColumns *Columns) *MetaResourceManager {
	newCols := NewMetaCols(columns.Array()...)
	man := NewClusterResourceManager(kw, kwp, newCols, adminColumns)
	return &MetaResourceManager{man, getName, getAge, getLabel}
}

type NamespaceResourceManager struct {
	*MetaResourceManager
	namespaceGetter
}

func NewNamespaceResourceManager(kw, kwp string, columns, adminColumns *Columns) *NamespaceResourceManager {
	newCols := NewNamespaceCols(columns.Array()...)
	man := NewMetaResourceManager(kw, kwp, newCols, adminColumns)
	return &NamespaceResourceManager{man, getNamespace}
}

type Columns struct {
	cols []string
}

func NewColumns(cols ...string) *Columns {
	c := &Columns{cols: make([]string, 0)}
	return c.Add(cols...)
}

func (c *Columns) Add(cols ...string) *Columns {
	for _, col := range cols {
		c.add(col)
	}
	return c
}

func (c *Columns) add(col string) *Columns {
	src := sets.NewString(c.Array()...)
	if src.Has(col) {
		return c
	}
	c.cols = append(c.cols, col)
	return c
}

func (c Columns) Array() []string {
	return c.cols
}

func NewNameCols(col ...string) *Columns {
	return NewColumns("Name").Add(col...)
}

func NewMetaCols(col ...string) *Columns {
	return NewNameCols("Age").Add(col...)
}

func NewNamespaceCols(col ...string) *Columns {
	return NewMetaCols("Namespace", "Labels").Add(col...)
}

func NewClusterCols(col ...string) *Columns {
	return NewColumns("Cluster").Add(col...)
}

func NewResourceCols(col ...string) *Columns {
	return NewNameCols("Id").Add(col...)
}

type ListPrinter interface {
	GetColumns(*mcclient.ClientSession) []string
}
