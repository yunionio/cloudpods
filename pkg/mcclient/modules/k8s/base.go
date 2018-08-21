package k8s

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func NewManager(keyword, keywordPlural string, columns, adminColumns *Columns) *modules.ResourceManager {
	return &modules.ResourceManager{
		BaseManager:   *modules.NewBaseManager("k8s", "", "", columns.Array(), adminColumns.Array()),
		Keyword:       keyword,
		KeywordPlural: keywordPlural,
	}
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
	return NewColumns("name", "namespace").Add(col...)
}

func NewClusterCols(col ...string) *Columns {
	return NewColumns("cluster").Add(col...)
}

func NewResourceCols(col ...string) *Columns {
	return NewColumns("name", "id").Add(col...)
}
