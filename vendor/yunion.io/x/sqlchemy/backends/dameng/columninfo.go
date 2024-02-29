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

package dameng

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/sqlchemy"
)

// COLUMN_NAME, DATA_TYPE, NULLABLE, DATA_LENGTH, CHARACTER_SET_NAME, DATA_DEFAULT
type sSqlColumnInfo struct {
	TableName        string `json:"TABLE_NAME"`
	ColumnName       string `json:"COLUMN_NAME"`
	DataType         string `json:"DATA_TYPE"`
	Nullable         string `json:"NULLABLE"`
	DataLength       int    `json:"DATA_LENGTH"`
	DataPrecision    int    `json:"DATA_PRECISION"`
	DataScale        int    `json:"DATA_SCALE"`
	CharacterSetName string `json:"CHARACTER_SET_NAME"`
	DataDefault      string `json:"DATA_DEFAULT"`
	IsPrimary        bool   `json:"is_primary"`

	IsAutoIncrement     bool   `json:"is_auto_increment"`
	AutoIncrementOffset uint64 `json:"auto_increment_offset"`
	AutoIncrementStep   uint64 `json:"auto_increment_step"`
}

func fetchTableColInfo(ts sqlchemy.ITableSpec) (map[string]*sSqlColumnInfo, error) {
	sqlStr := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, NULLABLE, DATA_LENGTH, DATA_PRECISION, DATA_SCALE, CHARACTER_SET_NAME, DATA_DEFAULT FROM USER_TAB_COLUMNS WHERE Table_Name='%s'", ts.Name())
	query := ts.Database().NewRawQuery(sqlStr, "column_name", "data_type", "nullable", "data_length", "data_precision", "data_scale", "character_set_name", "data_default")
	infos := make([]sSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	ret := make(map[string]*sSqlColumnInfo)
	for i := range infos {
		infos[i].TableName = ts.Name()
		infos[i].ColumnName = strings.ToLower(infos[i].ColumnName)
		ret[infos[i].ColumnName] = &infos[i]
	}

	indexes, err := fetchTableIndexes(ts)
	if err != nil {
		return nil, errors.Wrap(err, "fetchTableIndexes")
	}

	for _, idx := range indexes {
		if idx.isPrimary {
			for _, col := range idx.colnames {
				ret[col].IsPrimary = true
			}
			break
		}
	}

	autoIncInfo, err := fetchTableAutoIncrementCol(ts)
	if err != nil {
		return nil, errors.Wrap(err, "fetchTableAutoIncrementCol")
	} else if autoIncInfo != nil {
		ret[autoIncInfo.Name].IsAutoIncrement = true
		ret[autoIncInfo.Name].AutoIncrementOffset = autoIncInfo.Offset
		ret[autoIncInfo.Name].AutoIncrementStep = autoIncInfo.Step
	}

	return ret, nil
}

type sDamengTableIndex struct {
	isPrimary bool
	indexName string
	colnames  []string
}

func fetchTableIndexes(ts sqlchemy.ITableSpec) (map[string]sDamengTableIndex, error) {
	type sIndexInfo struct {
		ColumnName     string `json:"COLUMN_NAME"`
		IndexName      string `json:"INDEX_NAME"`
		ConstraintType string `json:"CONSTRAINT_TYPE"`
	}
	sqlStr := fmt.Sprintf("SELECT a.COLUMN_NAME, a.INDEX_NAME, b.CONSTRAINT_TYPE FROM USER_IND_COLUMNS a LEFT JOIN USER_CONSTRAINTS b ON a.INDEX_NAME=b.INDEX_NAME WHERE a.TABLE_NAME='%s'", ts.Name())
	query := ts.Database().NewRawQuery(sqlStr, "column_name", "index_name", "constraint_type")
	infos := make([]sIndexInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]sDamengTableIndex)
	for _, info := range infos {
		info.IndexName = strings.ToLower(info.IndexName)
		info.ColumnName = strings.ToLower(info.ColumnName)
		if idx, ok := ret[info.IndexName]; ok {
			idx.colnames = append(idx.colnames, info.ColumnName)
			ret[info.IndexName] = idx
		} else {
			ret[info.IndexName] = sDamengTableIndex{
				isPrimary: info.ConstraintType == "P",
				indexName: info.IndexName,
				colnames:  []string{info.ColumnName},
			}
		}
	}
	return ret, nil
}

func fetchTableAutoIncrementCol(ts sqlchemy.ITableSpec) (*sDamengAutoIncrementInfo, error) {
	sqlStr := fmt.Sprintf("SELECT a.NAME, c.INFO6 from SYSCOLUMNS a, SYSOBJECTS c WHERE a.INFO2 & 0x01 = 0x01 AND a.ID=c.ID and c.NAME='%s' AND c.SCHID=CURRENT_SCHID", ts.Name())
	query := ts.Database().NewRawQuery(sqlStr, "name", "info6")
	row := query.Row()
	name := ""
	info6 := make([]byte, 0)
	err := row.Scan(&name, &info6)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, errors.Wrap(err, "Query")
		}
	}
	return decodeInfo6(name, info6)
}

type sDamengAutoIncrementInfo struct {
	Name   string
	Offset uint64
	Step   uint64
	Dummy  uint64
}

func decodeInfo6(name string, binHex []byte) (*sDamengAutoIncrementInfo, error) {
	buf := bytes.NewReader(binHex)
	var pi [3]uint64
	for i := range pi {
		err := binary.Read(buf, binary.LittleEndian, &pi[i])
		if err != nil {
			return nil, errors.Wrap(err, "binary.LittleEndian.Read")
		}
	}
	return &sDamengAutoIncrementInfo{
		Name:   strings.ToLower(name),
		Offset: pi[0],
		Step:   pi[1],
		Dummy:  pi[2],
	}, nil
}

func (info *sSqlColumnInfo) toColumnSpec() sqlchemy.IColumnSpec {
	tagmap := make(map[string]string)

	typeStr := strings.ToUpper(info.DataType)
	if info.Nullable == "Y" {
		tagmap[sqlchemy.TAG_NULLABLE] = "true"
	} else {
		tagmap[sqlchemy.TAG_NULLABLE] = "false"
	}
	if info.IsPrimary {
		tagmap[sqlchemy.TAG_PRIMARY] = "true"
	} else {
		tagmap[sqlchemy.TAG_PRIMARY] = "false"
	}
	if info.DataDefault != "NULL" && len(info.DataDefault) > 0 {
		tagmap[sqlchemy.TAG_DEFAULT] = info.DataDefault
	}
	if typeStr == "VARCHAR" || typeStr == "CHAR" || typeStr == "CHARACTER" {
		tagmap[sqlchemy.TAG_WIDTH] = fmt.Sprintf("%d", info.DataLength)
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "'\"")
		}
		c := NewTextColumn(info.ColumnName, typeStr, tagmap, false)
		return &c
	} else if typeStr == "TEXT" || typeStr == "LONGVARCHAR" || typeStr == "CLOB" || typeStr == "BLOB" {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "'\"")
		}
		c := NewTextColumn(info.ColumnName, typeStr, tagmap, false)
		return &c
	} else if strings.HasSuffix(typeStr, "INT") {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "()")
		}
		if typeStr == "TINYINT" {
			if info.Nullable == "Y" {
				c := NewTristateColumn(info.TableName, info.ColumnName, tagmap, false)
				return &c
			} else {
				if info.DataDefault == "1" {
					c := NewBooleanColumn(info.ColumnName, tagmap, true)
					return &c
				} else {
					c := NewBooleanColumn(info.ColumnName, tagmap, false)
					return &c
				}
			}
		} else {
			if info.IsAutoIncrement {
				if info.AutoIncrementOffset > 0 {
					tagmap[sqlchemy.TAG_AUTOINCREMENT] = fmt.Sprintf("%d", info.AutoIncrementOffset)
				} else {
					tagmap[sqlchemy.TAG_AUTOINCREMENT] = "true"
				}
			}
			c := NewIntegerColumn(info.ColumnName, typeStr, tagmap, false)
			return &c
		}
	} else if typeStr == "REAL" || typeStr == "FLOAT" || typeStr == "DOUBLE" || typeStr == "DOUBLE PRECISION" {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "()")
		}
		c := NewFloatColumn(info.ColumnName, typeStr, tagmap, false)
		return &c
	} else if typeStr == "NUMERIC" || typeStr == "NUMBER" || typeStr == "DECIMAL" || typeStr == "DEC" {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "()")
		}
		tagmap[sqlchemy.TAG_WIDTH] = fmt.Sprintf("%d", info.DataPrecision)
		tagmap[sqlchemy.TAG_PRECISION] = fmt.Sprintf("%d", info.DataScale)
		c := NewDecimalColumn(info.ColumnName, tagmap, false)
		return &c
	} else if typeStr == "TIMESTAMP" || typeStr == "DATATIME" {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "'\"")
		}
		c := NewDateTimeColumn(info.ColumnName, tagmap, false)
		return &c
	} else if typeStr == "TIME" || typeStr == "DATE" {
		if val, ok := tagmap[sqlchemy.TAG_DEFAULT]; ok {
			tagmap[sqlchemy.TAG_DEFAULT] = strings.Trim(val, "'\"")
		}
		c := NewTimeTypeColumn(info.ColumnName, typeStr, tagmap, false)
		return &c
	} else {
		log.Errorf("unsupported column data type %s", typeStr)
		return nil
	}
}
