package sqlchemy

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

type SSqlColumnInfo struct {
	Field      string
	Type       string
	Collation  string
	Null       string
	Key        string
	Default    string
	Extra      string
	Privileges string
	Comment    string
}

func decodeSqlTypeString(typeStr string) []string {
	typeReg := regexp.MustCompile(`(\w+)\((\d+)(,\s*(\d+))?\)`)
	matches := typeReg.FindStringSubmatch(typeStr)
	if len(matches) >= 3 {
		return matches[1:]
	} else {
		return []string{typeStr}
	}
}

func (info *SSqlColumnInfo) toColumnSpec() IColumnSpec {
	tagmap := make(map[string]string)

	matches := decodeSqlTypeString(info.Type)
	typeStr := strings.ToUpper(matches[0])
	width := 0
	if len(matches) > 1 {
		width, _ = strconv.Atoi(matches[1])
	}
	if width > 0 {
		tagmap[TAG_WIDTH] = fmt.Sprintf("%d", width)
	}
	if info.Null == "YES" {
		tagmap[TAG_NULLABLE] = "true"
	} else {
		tagmap[TAG_NULLABLE] = "false"
	}
	if info.Key == "PRI" {
		tagmap[TAG_PRIMARY] = "true"
	} else {
		tagmap[TAG_PRIMARY] = "false"
	}
	charset := ""
	if info.Collation == "ascii_general_ci" {
		charset = "ascii"
	} else if info.Collation == "utf8_general_ci" {
		charset = "utf8"
	}
	if len(charset) > 0 {
		tagmap[TAG_CHARSET] = charset
	}
	if info.Default != "NULL" {
		tagmap[TAG_DEFAULT] = info.Default
	}
	if strings.HasSuffix(typeStr, "CHAR") {
		c := NewTextColumn(info.Field, tagmap)
		return &c
	} else if strings.HasSuffix(typeStr, "TEXT") {
		tagmap[TAG_TEXT_LENGTH] = typeStr[:len(typeStr)-4]
		c := NewTextColumn(info.Field, tagmap)
		return &c
	} else if strings.HasSuffix(typeStr, "INT") {
		if info.Extra == "auto_increment" {
			tagmap[TAG_AUTOINCREMENT] = "true"
		}
		unsigned := false
		if strings.HasSuffix(info.Type, " unsigned") {
			unsigned = true
		}
		c := NewIntegerColumn(info.Field, typeStr, unsigned, tagmap)
		return &c
	} else if typeStr == "FLOAT" || typeStr == "DOUBLE" {
		c := NewFloatColumn(info.Field, typeStr, tagmap)
		return &c
	} else if typeStr == "DECIMAL" {
		if len(matches) > 3 {
			precision, _ := strconv.Atoi(matches[3])
			if precision > 0 {
				tagmap[TAG_PRECISION] = fmt.Sprintf("%d", precision)
			}
		}
		c := NewDecimalColumn(info.Field, tagmap)
		return &c
	} else if typeStr == "DATETIME" {
		c := NewDateTimeColumn(info.Field, tagmap)
		return &c
	} else {
		log.Errorf("unsupported type %s", typeStr)
		return nil
	}
}

func (ts *STableSpec) fetchColumnDefs() ([]IColumnSpec, error) {
	sql := fmt.Sprintf("SHOW FULL COLUMNS IN %s", ts.name)
	query := NewRawQuery(sql, "field", "type", "collation", "null", "key", "default", "extra", "privileges", "comment")
	infos := make([]SSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	specs := make([]IColumnSpec, 0)
	for _, info := range infos {
		specs = append(specs, info.toColumnSpec())
	}
	return specs, nil
}

func compareColumnSpec(c1, c2 IColumnSpec) int {
	return strings.Compare(c1.Name(), c2.Name())
}

func diffCols(cols1 []IColumnSpec, cols2 []IColumnSpec) ([]IColumnSpec, []IColumnSpec, []IColumnSpec) {
	sort.Slice(cols1, func(i, j int) bool {
		return compareColumnSpec(cols1[i], cols1[j]) < 0
	})
	sort.Slice(cols2, func(i, j int) bool {
		return compareColumnSpec(cols2[i], cols2[j]) < 0
	})
	i := 0
	j := 0
	remove := make([]IColumnSpec, 0)
	update := make([]IColumnSpec, 0)
	add := make([]IColumnSpec, 0)
	for i < len(cols1) || j < len(cols2) {
		if i < len(cols1) && j < len(cols2) {
			comp := compareColumnSpec(cols1[i], cols2[j])
			if comp == 0 {
				if cols1[i].DefinitionString() != cols2[j].DefinitionString() {
					log.Infof("UPDATE: %s => %s", cols1[i].DefinitionString(), cols2[j].DefinitionString())
					update = append(update, cols2[j])
				}
				i += 1
				j += 1
			} else if comp > 0 {
				add = append(add, cols2[j])
				j += 1
			} else {
				remove = append(remove, cols1[i])
				i += 1
			}
		} else if i < len(cols1) {
			remove = append(remove, cols1[i])
			i += 1
		} else if j < len(cols2) {
			add = append(add, cols2[j])
			j += 1
		}
	}
	return remove, update, add
}

func (ts *STableSpec) SyncSQL() []string {
	tables := GetTables()
	in, _ := utils.InStringArray(ts.name, tables)
	if !in {
		log.Debugf("table %s not created yet", ts.name)
		sql := ts.CreateSQL()
		return []string{sql}
	}
	ret := make([]string, 0)
	cols, err := ts.fetchColumnDefs()
	if err != nil {
		log.Errorf("fetchColumnDefs fail: %s", err)
		return nil
	}
	remove, update, add := diffCols(cols, ts.columns)
	/* IGNORE DROP STATEMENT */
	for _, col := range remove {
		sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", ts.name, col.Name())
		// ret = append(ret, sql)
		log.Infof(sql)
	}
	for _, col := range update {
		sql := fmt.Sprintf("ALTER TABLE %s MODIFY %s;", ts.name, col.DefinitionString())
		ret = append(ret, sql)
	}
	for _, col := range add {
		sql := fmt.Sprintf("ALTER TABLE %s ADD %s;", ts.name, col.DefinitionString())
		ret = append(ret, sql)
	}
	return ret
}

func (ts *STableSpec) Sync() error {
	sqls := ts.SyncSQL()
	if sqls != nil {
		for _, sql := range sqls {
			_, err := _db.Exec(sql)
			if err != nil {
				log.Errorf("exec sql error %s: %s", sql, err)
				return err
			}
		}
	}
	return nil
}

func (ts *STableSpec) CheckSync() {
	sqls := ts.SyncSQL()
	if len(sqls) > 0 {
		for _, sql := range sqls {
			fmt.Println(sql)
		}
		log.Fatalf("DB table %q not in sync", ts.name)
	}
}
