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
	"fmt"

	"yunion.io/x/log"
)

// DBName is a type of string for name of database
type DBName string

// SDatabase represents a SQL database
type SDatabase struct {
	db      *sql.DB
	name    DBName
	backend IBackend
}

// DefaultDB is the name for the default database instance
const DefaultDB = DBName("__default__")

// the global DB connection table
var _db_tbl = make(map[DBName]*SDatabase)

// Deprecated
// SetDB sets global DB instance
func SetDB(db *sql.DB) {
	SetDefaultDB(db)
}

// SetDefaultDB save default global DB instance
func SetDefaultDB(db *sql.DB) {
	SetDBWithNameBackend(db, DefaultDB, MySQLBackend)
}

// SetDBWithName sets a DB instance with given name
// param: name DBName
func SetDBWithNameBackend(db *sql.DB, name DBName, backend DBBackendName) {
	_db_tbl[name] = &SDatabase{
		name:    name,
		db:      db,
		backend: getBackend(backend),
	}
}

// Deprecated
// GetDB get DB instance
func GetDB() *sql.DB {
	return GetDefaultDB().db
}

// GetDefaultDB get the DB instance set by default
func GetDefaultDB() *SDatabase {
	return GetDBWithName(DefaultDB)
}

// GetDBWithName returns the db instance with given name
func GetDBWithName(name DBName) *SDatabase {
	if name == DefaultDB && len(_db_tbl) == 1 {
		for _, db := range _db_tbl {
			return db
		}
	}
	if db, ok := _db_tbl[name]; ok {
		return db
	}
	panic(fmt.Sprintf("no such database %s", name))
}

type sDBReferer struct {
	dbName    DBName
	_db_cache *SDatabase
}

func (r *sDBReferer) Database() *SDatabase {
	if r._db_cache == nil {
		r._db_cache = GetDBWithName(r.dbName)
	}
	return r._db_cache
}

// CloseDB close DB connection
func CloseDB() {
	names := make([]DBName, 0)
	for n, db := range _db_tbl {
		names = append(names, n)
		db.db.Close()
	}
	for _, n := range names {
		delete(_db_tbl, n)
	}
}

type tableName struct {
	Name string
}

// GetTables get all tables' name in default database
func GetTables() []string {
	return GetDefaultDB().GetTables()
}

// GetTables get all tables' name in database
func (db *SDatabase) GetTables() []string {
	tables := make([]tableName, 0)
	q := db.NewRawQuery(db.backend.GetTableSQL(), "name")
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

// Exec execute a raw SQL query for the default db instance
func Exec(sql string, args ...interface{}) (sql.Result, error) {
	return GetDefaultDB().Exec(sql, args...)
}

// Exec execute a raw SQL query for a db instance
func (db *SDatabase) Exec(sql string, args ...interface{}) (sql.Result, error) {
	return db.db.Exec(sql, args...)
}
