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

package sqlite

import (
	"yunion.io/x/log"

	"yunion.io/x/sqlchemy"
)

type sSqlColumnInfo struct {
	Cid       int    `json:"cid"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Notnull   bool   `json:"notnull"`
	DfltValue string `json:"dflt_value"`
	Pk        bool   `json:"pk"`
}

func (info *sSqlColumnInfo) getTagmap() map[string]string {
	tagmap := make(map[string]string)
	if info.Notnull {
		tagmap[sqlchemy.TAG_NULLABLE] = "false"
	} else {
		tagmap[sqlchemy.TAG_NULLABLE] = "true"
	}
	if len(info.DfltValue) > 0 {
		tagmap[sqlchemy.TAG_DEFAULT] = info.DfltValue
		if info.Type == "TEXT" {
			tagmap[sqlchemy.TAG_DEFAULT] = info.DfltValue[1 : len(info.DfltValue)-1]
		}
	}
	if info.Pk {
		tagmap[sqlchemy.TAG_PRIMARY] = "true"
	}
	return tagmap
}

func (info *sSqlColumnInfo) toColumnSpec() sqlchemy.IColumnSpec {
	switch info.Type {
	case "TEXT", "BLOB":
		c := NewTextColumn(info.Name, info.getTagmap(), false)
		return &c
	case "INTEGER":
		c := NewIntegerColumn(info.Name, info.getTagmap(), false)
		return &c
	case "INTEGER AUTO_INCREMENT":
		tagmap := info.getTagmap()
		tagmap[sqlchemy.TAG_AUTOINCREMENT] = "true"
		c := NewIntegerColumn(info.Name, tagmap, false)
		return &c
	case "REAL":
		c := NewFloatColumn(info.Name, info.getTagmap(), false)
		return &c
	default:
		log.Errorf("unsupported type %s", info.Type)
	}
	return nil
}
