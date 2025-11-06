package sqlitex

import (
	"errors"

	"github.com/go-llsqlite/crawshaw"
)

var ErrNoResults = errors.New("sqlite: statement has no results")
var ErrMultipleResults = errors.New("sqlite: statement has multiple result rows")

func resultSetup(stmt *sqlite.Stmt) error {
	hasRow, err := stmt.Step()
	if err != nil {
		stmt.Reset()
		return err
	}
	if !hasRow {
		stmt.Reset()
		return ErrNoResults
	}
	return nil
}

func resultTeardown(stmt *sqlite.Stmt) error {
	hasRow, err := stmt.Step()
	if err != nil {
		stmt.Reset()
		return err
	}
	if hasRow {
		stmt.Reset()
		return ErrMultipleResults
	}
	return stmt.Reset()
}

// ResultInt steps the Stmt once and returns the first column as an int.
//
// If there are no rows in the result set, ErrNoResults is returned.
//
// If there are multiple rows, ErrMultipleResults is returned with the first
// result.
//
// The Stmt is always Reset, so repeated calls will always return the first
// result.
func ResultInt(stmt *sqlite.Stmt) (int, error) {
	res, err := ResultInt64(stmt)
	return int(res), err
}

// ResultInt64 steps the Stmt once and returns the first column as an int64.
//
// If there are no rows in the result set, ErrNoResults is returned.
//
// If there are multiple rows, ErrMultipleResults is returned with the first
// result.
//
// The Stmt is always Reset, so repeated calls will always return the first
// result.
func ResultInt64(stmt *sqlite.Stmt) (int64, error) {
	if err := resultSetup(stmt); err != nil {
		return 0, err
	}
	return stmt.ColumnInt64(0), resultTeardown(stmt)
}

// ResultText steps the Stmt once and returns the first column as a string.
//
// If there are no rows in the result set, ErrNoResults is returned.
//
// If there are multiple rows, ErrMultipleResults is returned with the first
// result.
//
// The Stmt is always Reset, so repeated calls will always return the first
// result.
func ResultText(stmt *sqlite.Stmt) (string, error) {
	if err := resultSetup(stmt); err != nil {
		return "", err
	}
	return stmt.ColumnText(0), resultTeardown(stmt)
}

// ResultFloat steps the Stmt once and returns the first column as a float64.
//
// If there are no rows in the result set, ErrNoResults is returned.
//
// If there are multiple rows, ErrMultipleResults is returned with the first
// result.
//
// The Stmt is always Reset, so repeated calls will always return the first
// result.
func ResultFloat(stmt *sqlite.Stmt) (float64, error) {
	if err := resultSetup(stmt); err != nil {
		return 0, err
	}
	return stmt.ColumnFloat(0), resultTeardown(stmt)
}
