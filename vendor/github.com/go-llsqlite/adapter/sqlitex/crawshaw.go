//go:build !zombiezen_sqlite

package sqlitex

import (
	"context"

	"github.com/go-llsqlite/crawshaw/sqlitex"

	sqlite "github.com/go-llsqlite/adapter"
)

var (
	// In zombiezen this can be done with ExecOption parameter to Execute. In crawshaw this is a
	// noop for now. Here is how it was done for zombiezen:
	// https://github.com/zombiezen/go-sqlite/commit/754b7de62e83f3bc7fd226d0e9e1ab2bbe0f6916.
	Transaction = Save
)

type ExecOptions = sqlitex.ExecOptions

func Execute(conn *sqlite.Conn, query string, opts *ExecOptions) error {
	return sqlitex.Execute(conn.Conn, query, opts)
}

func ExecuteScript(conn *sqlite.Conn, queries string, opts *ExecOptions) (err error) {
	return sqlitex.ExecuteScript(conn.Conn, queries, opts)
}

// TODO: Actually implement checked for crawshaw.
func ExecChecked(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) error {
	return sqlitex.Execute(conn.Conn, query, &ExecOptions{
		Args:       args,
		ResultFunc: resultFn,
	})
}

func ExecScript(conn *sqlite.Conn, queries string) error {
	return sqlitex.ExecScript(conn.Conn, queries)
}

func Save(conn *sqlite.Conn) (releaseFn func(*error)) {
	return sqlitex.Save(conn.Conn)
}

func Exec(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) error {
	return sqlitex.Exec(conn.Conn, query, resultFn, args...)
}

func ExecTransient(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) (err error) {
	return sqlitex.ExecTransient(conn.Conn, query, resultFn, args...)
}

type Pool struct {
	*sqlitex.Pool
}

func Open(uri string, flags sqlite.OpenFlags, poolSize int) (*Pool, error) {
	crawshawPool, err := sqlitex.Open(uri, flags, poolSize)
	return &Pool{crawshawPool}, err
}

func (me Pool) Get(ctx context.Context) *sqlite.Conn {
	conn := me.Pool.Get(ctx)
	if conn == nil {
		return nil
	}
	return &sqlite.Conn{conn}
}

func (me Pool) Put(conn *sqlite.Conn) {
	me.Pool.Put(conn.Conn)
}
