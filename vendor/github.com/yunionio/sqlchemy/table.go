package sqlchemy

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
)

type STableSpec struct {
	structType reflect.Type
	name       string
	columns    []IColumnSpec
}

type STable struct {
	spec  *STableSpec
	alias string
}

type STableField struct {
	table *STable
	spec  IColumnSpec
	alias string
}

func NewTableSpecFromStruct(s interface{}, name string) *STableSpec {
	st := reflect.TypeOf(s)
	if st.Kind() != reflect.Struct {
		log.Fatalf("Invalid table struct, NOT a STRUCT!!!")
		return nil
	}
	table := STableSpec{columns: make([]IColumnSpec, 0), name: name, structType: st}
	struct2TableSpec(st, &table)
	return &table
}

func (ts *STableSpec) Name() string {
	return ts.name
}

func (ts *STableSpec) Columns() []IColumnSpec {
	return ts.columns
}

func (ts *STableSpec) DataType() reflect.Type {
	return ts.structType
}

func (ts *STableSpec) CreateSQL() string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	indexes := make([]string, 0)
	for _, c := range ts.columns {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
		}
		if c.IsIndex() {
			indexes = append(indexes, fmt.Sprintf("KEY `ix_%s_%s` (`%s`)", ts.name, c.Name(), c.Name()))
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	if len(indexes) > 0 {
		cols = append(cols, indexes...)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8", ts.name, strings.Join(cols, ",\n"))
}

func (ts *STableSpec) Instance() *STable {
	table := STable{spec: ts, alias: getTableAliasName()}
	return &table
}

func (ts *STableSpec) ColumnSpec(name string) IColumnSpec {
	for _, c := range ts.Columns() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func (tbl *STable) Field(name string, alias ...string) IQueryField {
	// name = reflectutils.StructFieldName(name)
	name = utils.CamelSplit(name, "_")
	cSpec := tbl.spec.ColumnSpec(name)
	if cSpec == nil {
		log.Fatalf("Column %s not found", name)
		return nil
	}
	col := STableField{table: tbl, spec: cSpec}
	if len(alias) > 0 {
		col.Label(alias[0])
	}
	return &col
}

func (tbl *STable) Fields() []IQueryField {
	ret := make([]IQueryField, 0)
	for _, c := range tbl.spec.Columns() {
		ret = append(ret, tbl.Field(c.Name()))
	}
	return ret
}

func (c *STableField) Expression() string {
	if len(c.alias) > 0 {
		return fmt.Sprintf("%s.%s as `%s`", c.table.Alias(), c.spec.Name(), c.alias)
	} else {
		return fmt.Sprintf("%s.%s", c.table.Alias(), c.spec.Name())
	}
}

func (c *STableField) Name() string {
	if len(c.alias) > 0 {
		return c.alias
	} else {
		return c.spec.Name()
	}
}

func (c *STableField) Reference() string {
	return fmt.Sprintf("`%s`.`%s`", c.table.Alias(), c.Name())
}

func (c *STableField) Label(label string) IQueryField {
	if len(label) > 0 && label != c.spec.Name() {
		c.alias = label
	}
	return c
}
