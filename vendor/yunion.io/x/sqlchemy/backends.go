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
	"reflect"
)

type DBBackendName string

const (
	// MySQL is the backend name for MySQL/MariaDB
	MySQLBackend = DBBackendName("MySQL")
	// Clickhouse is the backend name of Clickhouse
	ClickhouseBackend = DBBackendName("Clickhouse")
	// SQLiteBackend is the backend name of Sqlite3
	SQLiteBackend = DBBackendName("SQLite")
	// PostgreSQLBackend = DBBackendName("PostgreSQL")
)

// IBackend is the interface for all kinds of sql backends, e.g. MySQL, ClickHouse, Sqlite, PostgreSQL, etc.
type IBackend interface {
	// Name returns the name of the driver
	Name() DBBackendName
	// GetTableSQL returns the SQL for query tablenames
	GetTableSQL() string
	// GetCreateSQL returns the SQL for create a table
	GetCreateSQLs(ts ITableSpec) []string
	//
	IsSupportIndexAndContraints() bool
	//
	FetchTableColumnSpecs(ts ITableSpec) ([]IColumnSpec, error)
	//
	FetchIndexesAndConstraints(ts ITableSpec) ([]STableIndex, []STableConstraint, error)
	//
	GetColumnSpecByFieldType(table *STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec
	//
	CurrentUTCTimeStampString() string

	// Capability

	// CanUpdate returns wether the backend supports update
	CanUpdate() bool
	// CanInsert returns wether the backend supports Insert
	CanInsert() bool
	// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
	CanInsertOrUpdate() bool

	DropIndexSQLTemplate() string

	CanSupportRowAffected() bool
}

var _driver_tbl = make(map[DBBackendName]IBackend)

// RegisterBackend registers a backend
func RegisterBackend(drv IBackend) {
	_driver_tbl[drv.Name()] = drv
}

func getBackend(name DBBackendName) IBackend {
	return _driver_tbl[name]
}
