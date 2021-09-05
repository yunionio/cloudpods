package sqlite

import (
	_ "github.com/mattn/go-sqlite3"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SSqliteBackend{})
}

type SSqliteBackend struct {
	sqlchemy.SBaseBackend
}

func (sqlite *SSqliteBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.SQLiteBackend
}

func (sqlite *SSqliteBackend) GetTableSQL() string {
	return "SELECT name FROM sqlite_master WHERE type='table'"
}

func (sqlite *SSqliteBackend) IsSupportIndexAndContraints() bool {
	return false
}

func (sqlite *SSqliteBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	// XXX: TO DO
	return ts.Columns(), nil
}
