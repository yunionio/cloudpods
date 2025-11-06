//go:build zombiezen_sqlite

package sqlitex

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	llsqlite "github.com/go-llsqlite/adapter"
)

var (
	Exec          = sqlitex.Exec
	Execute       = sqlitex.Execute
	ExecScript    = sqlitex.ExecScript
	ExecuteScript = sqlitex.ExecuteScript
	Transaction   = sqlitex.Transaction
	Open          = sqlitex.Open
	ExecTransient = sqlitex.ExecTransient
)

type (
	Pool        = sqlitex.Pool
	ExecOptions = sqlitex.ExecOptions
)

func ExecChecked(conn *sqlite.Conn, query string, res func(*llsqlite.Stmt) error, args ...any) error {
	return sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args:       args,
		ResultFunc: res,
	})
}
