//go:build !zombiezen_sqlite

package sqlite

import (
	"errors"
	"fmt"

	sqlite "github.com/go-llsqlite/crawshaw"
)

type (
	// This is only used as a pointer type because the existing APIs expect *Conn.
	Conn struct {
		// This can't be embedded because the underlying libraries probably won't support it.
		*sqlite.Conn
	}
	Stmt       = sqlite.Stmt
	Context    = sqlite.Context
	Value      = sqlite.Value
	OpenFlags  = sqlite.OpenFlags
	ResultCode sqlite.ErrorCode
	Blob       = sqlite.Blob
)

const (
	TypeNull = sqlite.SQLITE_NULL

	OpenNoMutex     = sqlite.SQLITE_OPEN_NOMUTEX
	OpenReadOnly    = sqlite.SQLITE_OPEN_READONLY
	OpenURI         = sqlite.SQLITE_OPEN_URI
	OpenWAL         = sqlite.SQLITE_OPEN_WAL
	OpenCreate      = sqlite.SQLITE_OPEN_CREATE
	OpenReadWrite   = sqlite.SQLITE_OPEN_READWRITE
	OpenSharedCache = sqlite.SQLITE_OPEN_SHAREDCACHE

	ResultCodeInterrupt        = ResultCode(sqlite.SQLITE_INTERRUPT)
	ResultCodeBusy             = ResultCode(sqlite.SQLITE_BUSY)
	ResultCodeAbort            = ResultCode(sqlite.SQLITE_ABORT)
	ResultCodeConstraintUnique = ResultCode(sqlite.SQLITE_CONSTRAINT_UNIQUE)
	ResultCodeGenericError     = ResultCode(sqlite.SQLITE_ERROR)

	LimitLength = sqlite.SQLITE_LIMIT_LENGTH
)

var (
	ErrBlobClosed = sqlite.ErrBlobClosed
)

// GoValue is a result value for application-defined functions. crawshaw provides the context result
// API, but zombiezen expects a hybrid value-type to be returned. GoValue calls out the Go part of
// this hybrid type explicitly.
type GoValue any

func BlobValue(b []byte) GoValue {
	return b
}

type FunctionImpl struct {
	NArgs         int
	Scalar        func(ctx Context, args []Value) (GoValue, error)
	Deterministic bool
	// This is exposed in zombiezen, but I don't think I need it for my use case. If I do I'll have
	// to add it to crawshaw.
	//AllowIndirect bool
}

func OpenConn(path string, flags ...OpenFlags) (*Conn, error) {
	crawshawConn, err := sqlite.OpenConn(path, flags...)
	return &Conn{crawshawConn}, err
}

func (c *Conn) CreateFunction(name string, impl *FunctionImpl) error {
	return c.Conn.CreateFunction(name, impl.Deterministic, impl.NArgs, func(context sqlite.Context, value ...sqlite.Value) {
		goResValue, err := impl.Scalar(context, value)
		if err != nil {
			context.ResultError(err)
			return
		}
		switch v := goResValue.(type) {
		case []byte:
			context.ResultBlob(v)
		default:
			context.ResultError(fmt.Errorf("unhandled function result type: %T", v))
		}
	}, nil, nil)
}

func GetResultCode(err error) (_ ResultCode, ok bool) {
	var crawshawError sqlite.Error
	if !errors.As(err, &crawshawError) {
		// Should we attempt to extract old pkg/error style errors like crawshaw's ErrCode does
		// here?
		return
	}
	return ResultCode(crawshawError.Code), true
}

// Deprecated. This may differ from GetResultCode in that it returns SQLITE_OK if there's no error,
// and SQLITE_ERROR if there's an error, but it can't extract a sqlite error code.
var ErrCode = sqlite.ErrCode
