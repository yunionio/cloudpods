package sqlitex

import (
	"errors"
	"fmt"
	sqlite "github.com/go-llsqlite/adapter"
)

func WithTransactionRollbackOnError(conn *sqlite.Conn, level string, inside func() error) (err error) {
	err = Exec(conn, fmt.Sprintf("begin %v", level), nil)
	if err != nil {
		return
	}
	err = inside()
	query := "rollback"
	if err == nil {
		query = "commit"
	}
	err = errors.Join(err, Exec(conn, query, nil))
	return
}
