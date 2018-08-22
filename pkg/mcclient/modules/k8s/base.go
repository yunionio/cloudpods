package k8s

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ResourceManager struct {
	*modules.ResourceManager
}

func NewResourceManager(keyword, keywordPlural string, columns, adminColumns *Columns) *ResourceManager {
	man := &modules.ResourceManager{
		BaseManager:   *modules.NewBaseManager("k8s", "", "", columns.Array(), adminColumns.Array()),
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

type NamespaceResourceManager struct {
	*ClusterResourceManager
}

func NewNamespaceResourceManager(kw, kwp string, columns, adminColumns *Columns) *NamespaceResourceManager {
	newCols := NewNamespaceCols(columns.Array()...)
	man := NewClusterResourceManager(kw, kwp, newCols, adminColumns)
	return &NamespaceResourceManager{man}
}

func (m NamespaceResourceManager) GetName(obj jsonutils.JSONObject) interface{} {
	name, _ := obj.GetString("name")
	return name
}

func (m NamespaceResourceManager) GetNamespace(obj jsonutils.JSONObject) interface{} {
	ns, _ := obj.GetString("namespace")
	return ns
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

func NewNamespaceCols(col ...string) *Columns {
	return NewColumns("Name", "Namespace").Add(col...)
}

func NewClusterCols(col ...string) *Columns {
	return NewColumns("Cluster").Add(col...)
}

func NewResourceCols(col ...string) *Columns {
	return NewColumns("Name", "Id").Add(col...)
}

type ListPrinter interface {
	GetColumns(*mcclient.ClientSession) []string
}
