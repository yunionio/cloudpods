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
	// IsSupportIndexAndContraints returns whether the backend supports index and contraints such as foreigh keys
	//     MySQL: true
	//     Sqlite: true
	//     Clickhouse: false
	IsSupportIndexAndContraints() bool
	// FetchTableColumnSpecs parse the table definition in database to extract columns' specification of a table
	FetchTableColumnSpecs(ts ITableSpec) ([]IColumnSpec, error)
	// FetchIndexesAndConstraints parse the table defintion in database to extract index and constraints information of a table
	FetchIndexesAndConstraints(ts ITableSpec) ([]STableIndex, []STableConstraint, error)
	// GetColumnSpecByFieldType parse the field of model struct to extract column specifiction of a field
	GetColumnSpecByFieldType(table *STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec
	// CurrentUTCTimeStampString returns the string represents current UTC time
	CurrentUTCTimeStampString() string
	// CurrentTimeStampString returns the string represents current local time
	CurrentTimeStampString() string
	//
	CaseInsensitiveLikeString() string
	//
	UnionAllString() string
	//
	UnionDistinctString() string
	// support mixed insert vars
	SupportMixedInsertVariables() bool
	// Drop table
	DropTableSQL(table string) string

	// Capability

	// CanUpdate returns wether the backend supports update
	CanUpdate() bool
	// CanInsert returns wether the backend supports Insert
	CanInsert() bool
	// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
	CanInsertOrUpdate() bool

	// InsertSQLTemplate returns the template of insert SQL
	InsertSQLTemplate() string
	// UpdateSQLTemplate returns the template of update SQL
	UpdateSQLTemplate() string
	// InsertOrUpdateSQLTemplate returns the template of insert or update SQL
	InsertOrUpdateSQLTemplate() string

	// CanSupportRowAffected returns wether the backend support RowAffected method after update
	//     MySQL: true
	//     Sqlite: false
	//     Clickhouse: false
	CanSupportRowAffected() bool

	// CommitTableChangeSQL outputs the SQLs to alter a table
	CommitTableChangeSQL(ts ITableSpec, changes STableChanges) []string

	///////////////////////////////////////////////////////////////////////
	////////////////// FUNCTIONS //////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////

	// cast
	CAST(field IQueryField, typeStr string, fieldname string) IQueryField
	// TIMESTAMPADD
	TIMESTAMPADD(name string, field IQueryField, offsetSeconds int) IQueryField
	// INET_ATON
	INET_ATON(field IQueryField) IQueryField
	// AND_Val
	AND_Val(name string, field IQueryField, v interface{}) IQueryField
	// OR_Val
	OR_Val(name string, field IQueryField, v interface{}) IQueryField
	// SUBSTR
	SUBSTR(name string, field IQueryField, pos, length int) IQueryField
	// CONCAT
	CONCAT(name string, fields ...IQueryField) IQueryField
	// REPLACE
	REPLACE(name string, field IQueryField, old string, new string) IQueryField
	// GROUP_CONCAT2
	GROUP_CONCAT2(name string, sep string, field IQueryField) IQueryField
	// DISTINCT
	DISTINCT(name string, field IQueryField) IQueryField
	// COUNT
	COUNT(name string, field ...IQueryField) IQueryField
	// MAX
	MAX(name string, field IQueryField) IQueryField
	// MIN
	MIN(name string, field IQueryField) IQueryField
	// SUM
	SUM(name string, field IQueryField) IQueryField
	// LENGTH
	LENGTH(name string, field IQueryField) IQueryField
	// LOWER
	LOWER(name string, field IQueryField) IQueryField
	// UPPER
	UPPER(name string, field IQueryField) IQueryField
	// DATEDIFF
	DATEDIFF(unit string, field1, field2 IQueryField) IQueryField
}

var _driver_tbl = make(map[DBBackendName]IBackend)

// RegisterBackend registers a backend
func RegisterBackend(drv IBackend) {
	_driver_tbl[drv.Name()] = drv
}

func getBackend(name DBBackendName) IBackend {
	return _driver_tbl[name]
}
