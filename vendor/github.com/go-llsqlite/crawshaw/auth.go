package sqlite

// #include <stdint.h>
// #include <sqlite3.h>
// #include "wrappers.h"
//
// static int sqlite3_go_set_authorizer(sqlite3* conn, uintptr_t id) {
//   return sqlite3_set_authorizer(conn, c_auth_tramp, (void*)id);
// }
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// An Authorizer is called during statement preparation to see whether an action
// is allowed by the application. See https://sqlite.org/c3ref/set_authorizer.html
type Authorizer interface {
	Authorize(info ActionInfo) AuthResult
}

// SetAuthorizer registers an authorizer for the database connection.
// SetAuthorizer(nil) clears any authorizer previously set.
func (conn *Conn) SetAuthorizer(auth Authorizer) error {
	if auth == nil {
		if conn.authorizer == -1 {
			return nil
		}
		conn.releaseAuthorizer()
		res := C.sqlite3_set_authorizer(conn.conn, nil, nil)
		return reserr("SetAuthorizer", "", "", res)
	}

	authFuncs.mu.Lock()
	id := authFuncs.next
	next := authFuncs.next + 1
	if next < 0 {
		authFuncs.mu.Unlock()
		return errors.New("sqlite: authorizer function id overflow")
	}
	authFuncs.next = next
	authFuncs.m[id] = auth
	authFuncs.mu.Unlock()

	res := C.sqlite3_go_set_authorizer(conn.conn, C.uintptr_t(id))
	return reserr("SetAuthorizer", "", "", res)
}

func (conn *Conn) releaseAuthorizer() {
	if conn.authorizer == -1 {
		return
	}
	authFuncs.mu.Lock()
	delete(authFuncs.m, conn.authorizer)
	authFuncs.mu.Unlock()
	conn.authorizer = -1
}

var authFuncs = struct {
	mu   sync.RWMutex
	m    map[int]Authorizer
	next int
}{
	m: make(map[int]Authorizer),
}

//export go_sqlite_auth_tramp
func go_sqlite_auth_tramp(id uintptr, action C.int, cArg1, cArg2 *C.char, cDB *C.char, cTrigger *C.char) C.int {
	authFuncs.mu.RLock()
	auth := authFuncs.m[int(id)]
	authFuncs.mu.RUnlock()
	var arg1, arg2, database, trigger string
	if cArg1 != nil {
		arg1 = C.GoString(cArg1)
	}
	if cArg2 != nil {
		arg2 = C.GoString(cArg2)
	}
	if cDB != nil {
		database = C.GoString(cDB)
	}
	if cTrigger != nil {
		trigger = C.GoString(cTrigger)
	}
	return C.int(auth.Authorize(newActionInfo(OpType(action), arg1, arg2, database, trigger)))
}

// AuthorizeFunc is a function that implements Authorizer.
type AuthorizeFunc func(info ActionInfo) AuthResult

// Authorize calls f.
func (f AuthorizeFunc) Authorize(info ActionInfo) AuthResult {
	return f(info)
}

// AuthResult is the result of a call to an Authorizer. The zero value is
// SQLITE_OK.
type AuthResult int

// Possible return values of an Authorizer.
const (
	// Cause the entire SQL statement to be rejected with an error.
	SQLITE_DENY = AuthResult(C.SQLITE_DENY)
	// Disallow the specific action but allow the SQL statement to continue to
	// be compiled.
	SQLITE_IGNORE = AuthResult(C.SQLITE_IGNORE)
)

// String returns the C constant name of the result.
func (result AuthResult) String() string {
	switch result {
	default:
		var buf [20]byte
		return "SQLITE_UNKNOWN_AUTH_RESULT(" + string(itoa(buf[:], int64(result))) + ")"
	case AuthResult(C.SQLITE_OK):
		return "SQLITE_OK"
	case SQLITE_DENY:
		return "SQLITE_DENY"
	case SQLITE_IGNORE:
		return "SQLITE_IGNORE"
	}
}

// ActionInfo holds information about an action to be authorized.
//
// Only the fields relevant to the Action are populated when this is passed to
// an Authorizer.
//
// https://sqlite.org/c3ref/c_alter_table.html
type ActionInfo struct {
	Action OpType

	Index     string
	Table     string
	Column    string
	Trigger   string
	View      string
	Function  string
	Pragma    string
	PragmaArg string
	Operation string
	Filename  string
	Module    string
	Database  string
	Savepoint string

	InnerMostTrigger string
}

// newActionInfo returns an ActionInfo with the correct fields relevant to the
// action.
func newActionInfo(action OpType, arg1, arg2, database, trigger string) ActionInfo {

	// We use the blank identifier with unused args below simply for visual
	// consistency between the cases.

	a := ActionInfo{Action: action, Database: database, InnerMostTrigger: trigger}
	switch action {
	case SQLITE_DROP_INDEX,
		SQLITE_DROP_TEMP_INDEX,
		SQLITE_CREATE_INDEX,
		SQLITE_CREATE_TEMP_INDEX:
		/* Index Name      Table Name      */
		a.Index = arg1
		a.Table = arg2

	case SQLITE_DELETE,
		SQLITE_DROP_TABLE,
		SQLITE_DROP_TEMP_TABLE,
		SQLITE_INSERT,
		SQLITE_ANALYZE,
		SQLITE_CREATE_TABLE,
		SQLITE_CREATE_TEMP_TABLE:
		/* Table Name      NULL            */
		a.Table = arg1
		_ = arg2

	case SQLITE_CREATE_TEMP_TRIGGER,
		SQLITE_CREATE_TRIGGER,
		SQLITE_DROP_TEMP_TRIGGER,
		SQLITE_DROP_TRIGGER:
		/* Trigger Name    Table Name      */
		a.Trigger = arg1
		a.Table = arg2

	case SQLITE_CREATE_TEMP_VIEW,
		SQLITE_CREATE_VIEW,
		SQLITE_DROP_TEMP_VIEW,
		SQLITE_DROP_VIEW:
		/* View Name       NULL            */
		a.View = arg1
		_ = arg2

	case SQLITE_PRAGMA:
		/* Pragma Name     1st arg or NULL */
		a.Pragma = arg1
		a.PragmaArg = arg2

	case SQLITE_READ,
		SQLITE_UPDATE:
		/* Table Name      Column Name     */
		a.Table = arg1
		a.Column = arg2

	case SQLITE_TRANSACTION:
		/* Operation       NULL            */
		a.Operation = arg1
		_ = arg2

	case SQLITE_ATTACH:
		/* Filename        NULL            */
		a.Filename = arg1
		_ = arg2

	case SQLITE_DETACH:
		/* Database Name   NULL            */
		a.Database = arg1
		_ = arg2

	case SQLITE_ALTER_TABLE:
		/* Database Name   Table Name      */
		a.Database = arg1
		a.Table = arg2

	case SQLITE_REINDEX:
		/* Index Name      NULL            */
		a.Index = arg1
		_ = arg2

	case SQLITE_CREATE_VTABLE,
		SQLITE_DROP_VTABLE:
		/* Table Name      Module Name     */
		a.Table = arg1
		a.Module = arg2

	case SQLITE_FUNCTION:
		/* NULL            Function Name   */
		_ = arg1
		a.Function = arg2

	case SQLITE_SAVEPOINT:
		/* Operation       Savepoint Name  */
		a.Operation = arg1
		a.Savepoint = arg2

	case SQLITE_RECURSIVE,
		SQLITE_SELECT:
		/* NULL            NULL            */
		_ = arg1
		_ = arg2

	case SQLITE_COPY:
		/* No longer used */
	default:
		panic(fmt.Errorf("unknown action: %v", action))
	}
	return a
}

// String returns a string describing only the fields relevant to `a.Action`.
func (a ActionInfo) String() string {

	switch a.Action {
	case SQLITE_DROP_INDEX,
		SQLITE_DROP_TEMP_INDEX,
		SQLITE_CREATE_INDEX,
		SQLITE_CREATE_TEMP_INDEX:
		/* Index Name      Table Name      */
		return fmt.Sprintf("%v: Index: %q Table: %q",
			a.Action, a.Index, a.Table)

	case SQLITE_DELETE,
		SQLITE_DROP_TABLE,
		SQLITE_DROP_TEMP_TABLE,
		SQLITE_INSERT,
		SQLITE_ANALYZE,
		SQLITE_CREATE_TABLE,
		SQLITE_CREATE_TEMP_TABLE:
		/* Table Name      NULL            */
		return fmt.Sprintf("%v: Table: %q", a.Action, a.Table)

	case SQLITE_CREATE_TEMP_TRIGGER,
		SQLITE_CREATE_TRIGGER,
		SQLITE_DROP_TEMP_TRIGGER,
		SQLITE_DROP_TRIGGER:
		/* Trigger Name    Table Name      */
		return fmt.Sprintf("%v: Trigger: %q Table: %q",
			a.Action, a.Trigger, a.Table)

	case SQLITE_CREATE_TEMP_VIEW,
		SQLITE_CREATE_VIEW,
		SQLITE_DROP_TEMP_VIEW,
		SQLITE_DROP_VIEW:
		/* View Name       NULL            */
		return fmt.Sprintf("%v: View: %q", a.Action, a.View)

	case SQLITE_PRAGMA:
		/* Pragma Name     1st arg or NULL */
		return fmt.Sprintf("%v: Pragma: %q",
			a.Action, strings.TrimSpace(a.Pragma+" "+a.PragmaArg))

	case SQLITE_READ,
		SQLITE_UPDATE:
		/* Table Name      Column Name     */
		return fmt.Sprintf("%v: Table: %q Column: %q",
			a.Action, a.Table, a.Column)

	case SQLITE_TRANSACTION:
		/* Operation       NULL            */
		return fmt.Sprintf("%v: Operation: %q", a.Action, a.Operation)

	case SQLITE_ATTACH:
		/* Filename        NULL            */
		return fmt.Sprintf("%v: Filename: %q", a.Action, a.Filename)

	case SQLITE_DETACH:
		/* Database Name   NULL            */
		return fmt.Sprintf("%v: Database: %q", a.Action, a.Database)

	case SQLITE_ALTER_TABLE:
		/* Database Name   Table Name      */
		return fmt.Sprintf("%v: Database: %q Table: %q",
			a.Action, a.Database, a.Table)

	case SQLITE_REINDEX:
		/* Index Name      NULL            */
		return fmt.Sprintf("%v: Index: %q", a.Action, a.Index)

	case SQLITE_CREATE_VTABLE,
		SQLITE_DROP_VTABLE:
		/* Table Name      Module Name     */
		return fmt.Sprintf("%v: Table: %q Module: %q",
			a.Action, a.Table, a.Module)

	case SQLITE_FUNCTION:
		/* NULL            Function Name   */
		return fmt.Sprintf("%v: Function: %q", a.Action, a.Function)

	case SQLITE_SAVEPOINT:
		/* Operation       Savepoint Name  */
		return fmt.Sprintf("%v: Operation: %q Savepoint: %q",
			a.Action, a.Operation, a.Savepoint)

	case SQLITE_RECURSIVE,
		SQLITE_SELECT:
		/* NULL            NULL            */
		return fmt.Sprintf("%v:", a.Action)

	case SQLITE_COPY:
		/* No longer used */
		return fmt.Sprintf("%v:", a.Action)
	default:
		return fmt.Sprintf("unknown action: %v", a.Action)
	}
}
