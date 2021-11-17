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

package sqlchemy

import (
	"database/sql"

	"yunion.io/x/log"
)

// the global DB connection
var _db *sql.DB

// SetDB sets global DB instance
func SetDB(db *sql.DB) {
	_db = db
}

// GetDB get DB instance
func GetDB() *sql.DB {
	return _db
}

// CloseDB close DB connection
func CloseDB() {
	_db.Close()
	_db = nil
}

type tableName struct {
	Name string
}

// GetTables get all tables' name in database
func GetTables() []string {
	tables := make([]tableName, 0)
	q := NewRawQuery("SHOW TABLES", "name")
	err := q.All(&tables)
	if err != nil {
		log.Errorf("show tables fail %s", err)
		return nil
	}
	ret := make([]string, len(tables))
	for i, t := range tables {
		ret[i] = t.Name
	}
	return ret
}

// Exec execute a raw SQL query
func Exec(sql string, args ...interface{}) (sql.Result, error) {
	return _db.Exec(sql, args...)
}
