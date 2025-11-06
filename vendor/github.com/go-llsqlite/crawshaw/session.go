// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package sqlite

// #include <stdint.h>
// #include <stdlib.h>
// #include <sqlite3.h>
// #include "wrappers.h"
//
// static int go_sqlite3session_changeset_strm(
//   sqlite3_session *pSession,
//   int (*xOutput)(void *pOut, const void *pData, int nData),
//   uintptr_t pOut
// ) {
//   return sqlite3session_changeset_strm(pSession, xOutput, (void *)pOut);
// }
//
// static int go_sqlite3session_patchset_strm(
//   sqlite3_session *pSession,
//   int (*xOutput)(void *pOut, const void *pData, int nData),
//   uintptr_t pOut
// ) {
//   return sqlite3session_patchset_strm(pSession, xOutput, (void *)pOut);
// }
//
// static int go_sqlite3changeset_invert_strm(
//   int (*xInput)(void *pIn, void *pData, int *pnData),
//   uintptr_t pIn,
//   int (*xOutput)(void *pOut, const void *pData, int nData),
//   uintptr_t pOut
// ) {
//   return sqlite3changeset_invert_strm(xInput, (void *)pIn, xOutput, (void *)pOut);
// }
//
// static int go_sqlite3changeset_apply_v2_strm(
//   sqlite3 *db,                    /* Apply change to "main" db of this handle */
//   int (*xInput)(void *pIn, void *pData, int *pnData), /* Input function */
//   uintptr_t pIn,                                          /* First arg for xInput */
//   int(*xFilter)(
//     void *pCtx,                   /* Copy of sixth arg to _apply() */
//     const char *zTab              /* Table name */
//   ),
//   int(*xConflict)(
//     void *pCtx,                   /* Copy of sixth arg to _apply() */
//     int eConflict,                /* DATA, MISSING, CONFLICT, CONSTRAINT */
//     sqlite3_changeset_iter *p     /* Handle describing change and conflict */
//   ),
//   uintptr_t pCtx,                     /* First argument passed to xConflict */
//   void **ppRebase, int *pnRebase,
//   int flags
// ) {
//   return sqlite3changeset_apply_v2_strm(db,
//     xInput,
//     (void *)pIn,
//     xFilter,
//     xConflict,
//     (void *)pCtx,
//     ppRebase,
//     pnRebase,
//     flags
//   );
// }
//
// static int go_sqlite3changeset_concat_strm(
//   int (*xInputA)(void *pIn, void *pData, int *pnData),
//   uintptr_t pInA,
//   int (*xInputB)(void *pIn, void *pData, int *pnData),
//   uintptr_t pInB,
//   int (*xOutput)(void *pOut, const void *pData, int nData),
//   uintptr_t pOut
// ) {
//   return sqlite3changeset_concat_strm(
//     xInputA, (void *)pInA,
//     xInputB, (void *)pInB,
//     xOutput, (void *)pOut
//   );
// }
//
// static int go_sqlite3changeset_start_strm(
//   sqlite3_changeset_iter **pp,
//   int (*xInput)(void *pIn, void *pData, int *pnData),
//   uintptr_t pIn
// ) {
//   return sqlite3changeset_start_strm(pp, xInput, (void *)pIn);
// }
//
// static int go_sqlite3changegroup_add_strm(sqlite3_changegroup* g,
//     int (*xInput)(void *pIn, void *pData, int *pnData),
//     uintptr_t pIn
// ) {
//   return sqlite3changegroup_add_strm(g, xInput, (void *)pIn);
// }
//
// static int go_sqlite3changegroup_output_strm(sqlite3_changegroup* g,
//     int (*xOutput)(void *pOut, const void *pData, int nData),
//     uintptr_t pOut
// ) {
//   return sqlite3changegroup_output_strm(g, xOutput, (void *)pOut);
// }
import "C"
import (
	"io"
	"runtime"
	"sync"
	"unsafe"
)

// A Session tracks database changes made by a Conn.
//
// It is used to build changesets.
//
// Equivalent to the sqlite3_session* C object.
type Session struct {
	ptr *C.sqlite3_session
}

// CreateSession creates a new session object.
// If db is "", then a default of "main" is used.
//
// https://www.sqlite.org/session/sqlite3session_create.html
func (conn *Conn) CreateSession(db string) (*Session, error) {
	var cdb *C.char
	if db == "" || db == "main" {
		cdb = cmain
	} else {
		cdb = C.CString(db)
		defer C.free(unsafe.Pointer(cdb))
	}
	s := &Session{}
	res := C.sqlite3session_create(conn.conn, cdb, &s.ptr)
	if err := conn.reserr("Conn.CreateSession", db, res); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(s, func(s *Session) {
		if s.ptr != nil {
			panic("open *sqlite.Session garbage collected, call Delete method")
		}
	})

	return s, nil
}

// Delete deletes a Session object.
//
// https://www.sqlite.org/session/sqlite3session_delete.html
func (s *Session) Delete() {
	C.sqlite3session_delete(s.ptr)
	s.ptr = nil
}

// Enable enables recording of changes by a Session.
// New Sessions start enabled.
//
// https://www.sqlite.org/session/sqlite3session_enable.html
func (s *Session) Enable() { C.sqlite3session_enable(s.ptr, 1) }

// Disable disables recording of changes by a Session.
//
// https://www.sqlite.org/session/sqlite3session_enable.html
func (s *Session) Disable() { C.sqlite3session_enable(s.ptr, 0) }

// Attach attaches a table to the session object.
// Changes made to the table will be tracked by the session.
//
// An empty tableName attaches all the tables in the database.
func (s *Session) Attach(tableName string) error {
	var ctable *C.char
	if tableName != "" {
		ctable = C.CString(tableName)
		defer C.free(unsafe.Pointer(ctable))
	}
	res := C.sqlite3session_attach(s.ptr, ctable)
	return reserr("Session.Attach", tableName, "", res)
}

// Diff appends the difference between two tables (srcDB and the session DB) to the session.
// The two tables must have the same name and schema.
func (s *Session) Diff(srcDB, tableName string) error {
	var errmsg *C.char
	csrcDB := C.CString(srcDB)
	ctable := C.CString(tableName)
	defer func() {
		C.free(unsafe.Pointer(csrcDB))
		C.free(unsafe.Pointer(ctable))
		if errmsg != nil {
			C.free(unsafe.Pointer(errmsg))
		}
	}()

	res := C.sqlite3session_diff(s.ptr, csrcDB, ctable, &errmsg)
	if res != 0 {
		return reserr("Session.Diff", srcDB+"."+tableName, C.GoString(errmsg), res)
	}
	return nil
}

// Changeset generates a changeset from a session.
//
// https://www.sqlite.org/session/sqlite3session_changeset.html
func (s *Session) Changeset(w io.Writer) error {
	x := newStrm(w, nil)
	defer x.free()
	res := C.go_sqlite3session_changeset_strm(s.ptr, (*[0]byte)(C.c_strm_w_tramp), x.cptr())

	return reserr("Session.Changeset", "", "", res)
}

// Patchset generates a patchset from a session.
//
// https://www.sqlite.org/session/sqlite3session_patchset.html
func (s *Session) Patchset(w io.Writer) error {
	x := newStrm(w, nil)
	defer x.free()
	res := C.go_sqlite3session_patchset_strm(s.ptr, (*[0]byte)(C.c_strm_w_tramp), x.cptr())
	return reserr("Session.Patchset", "", "", res)
}

// ChangesetApply applies a changeset to the database.
//
// If a changeset will not apply cleanly then conflictFn can be used to resolve
// the conflict. See the SQLite documentation for full details.
//
// https://www.sqlite.org/session/sqlite3changeset_apply.html
func (conn *Conn) ChangesetApply(r io.Reader,
	filterFn func(tableName string) bool,
	conflictFn func(ConflictType, ChangesetIter) ConflictAction) error {
	return conn.changesetApply(r, filterFn, conflictFn, false)
}

// ChangesetApplyInverse applies the inverse of a changeset to the database.
//
// If a changeset will not apply cleanly then conflictFn can be used to resolve
// the conflict. See the SQLite documentation for full details.
//
// This is equivalent to inverting a changeset using ChangesetInvert before
// applying it. It is an error to use a patchset.
//
// https://www.sqlite.org/session/sqlite3changeset_apply.html
//
// https://www.sqlite.org/session/c_changesetapply_invert.html
//
// https://www.sqlite.org/session/sqlite3changeset_invert.html
func (conn *Conn) ChangesetApplyInverse(r io.Reader,
	filterFn func(tableName string) bool,
	conflictFn func(ConflictType, ChangesetIter) ConflictAction) error {
	return conn.changesetApply(r, filterFn, conflictFn, true)
}

func (conn *Conn) changesetApply(r io.Reader,
	filterFn func(tableName string) bool,
	conflictFn func(ConflictType, ChangesetIter) ConflictAction,
	invert bool) error {
	xIn := newStrm(nil, r)
	x := &xapply{
		conn:       conn,
		filterFn:   filterFn,
		conflictFn: conflictFn,
	}

	xapplys.mu.Lock()
	xapplys.next++
	x.id = xapplys.next
	xapplys.m[x.id] = x
	xapplys.mu.Unlock()

	var filterTramp, conflictTramp *[0]byte
	if x.filterFn != nil {
		filterTramp = (*[0]byte)(C.c_xapply_filter_tramp)
	}
	if x.conflictFn != nil {
		conflictTramp = (*[0]byte)(C.c_xapply_conflict_tramp)
	}

	var flags C.int
	if invert {
		flags = C.SQLITE_CHANGESETAPPLY_INVERT
	}

	pCtx := (C.uintptr_t)(x.id)
	res := C.go_sqlite3changeset_apply_v2_strm(conn.conn,
		(*[0]byte)(C.c_strm_r_tramp),
		xIn.cptr(),
		filterTramp, conflictTramp,
		pCtx,
		nil, nil,
		flags)

	xapplys.mu.Lock()
	delete(xapplys.m, x.id)
	xapplys.mu.Unlock()

	xIn.free()

	return conn.reserr("Conn.ChangesetApply", "", res)
}

// ChangesetInvert inverts a changeset.
//
// https://www.sqlite.org/session/sqlite3changeset_invert.html
func ChangesetInvert(w io.Writer, r io.Reader) error {
	xIn := newStrm(nil, r)
	xOut := newStrm(w, nil)
	res := C.go_sqlite3changeset_invert_strm(
		(*[0]byte)(C.c_strm_r_tramp), xIn.cptr(),
		(*[0]byte)(C.c_strm_w_tramp), xOut.cptr(),
	)
	xIn.free()
	xOut.free()
	return reserr("ChangesetInvert", "", "", res)
}

// ChangesetConcat concatenates two changesets.
//
// https://www.sqlite.org/session/sqlite3changeset_concat.html
func ChangesetConcat(w io.Writer, r1, r2 io.Reader) error {
	xInA := newStrm(nil, r1)
	xInB := newStrm(nil, r2)
	xOut := newStrm(w, nil)
	res := C.go_sqlite3changeset_concat_strm(
		(*[0]byte)(C.c_strm_r_tramp), xInA.cptr(),
		(*[0]byte)(C.c_strm_r_tramp), xInB.cptr(),
		(*[0]byte)(C.c_strm_w_tramp), xOut.cptr(),
	)
	xInA.free()
	xInB.free()
	xOut.free()
	return reserr("ChangesetConcat", "", "", res)
}

// ChangesetIter is an iterator over a changeset.
//
// An iterator is used much like a Stmt over result rows.
// It is also used in the conflictFn provided to ChangesetApply.
// To process the changes in a changeset:
//
//	iter, err := ChangesetIterStart(r)
//	if err != nil {
//		// ... handle err
//	}
//	for {
//		hasRow, err := iter.Next()
//		if err != nil {
//			// ... handle err
//		}
//		if !hasRow {
//			break
//		}
//		// Use the Op, New, Old method to inspect the change.
//	}
//	if err := iter.Finalize(); err != nil {
//		// ... handle err
//	}
type ChangesetIter struct {
	ptr *C.sqlite3_changeset_iter
	xIn *strm
}

// ChangesetIterStart creates an iterator over a changeset.
//
// https://www.sqlite.org/session/sqlite3changeset_start.html
func ChangesetIterStart(r io.Reader) (ChangesetIter, error) {
	iter := ChangesetIter{}
	iter.xIn = newStrm(nil, r)
	res := C.go_sqlite3changeset_start_strm(
		&iter.ptr,
		(*[0]byte)(C.c_strm_r_tramp),
		iter.xIn.cptr())
	if err := reserr("ChangesetIterStart", "", "", res); err != nil {
		return ChangesetIter{}, err
	}
	return iter, nil
}

// Finalize deletes a changeset iterator.
// Do not use in iterators passed to a ChangesetApply conflictFn.
//
// https://www.sqlite.org/session/sqlite3changeset_finalize.html
func (iter ChangesetIter) Finalize() error {
	res := C.sqlite3changeset_finalize(iter.ptr)
	iter.ptr = nil
	if iter.xIn != nil {
		iter.xIn.free()
		iter.xIn = nil
	}
	return reserr("ChangesetIter.Finalize", "", "", res)
}

// Old obtains old row values from an iterator.
//
// https://www.sqlite.org/session/sqlite3changeset_old.html
func (iter ChangesetIter) Old(col int) (v Value, err error) {
	res := C.sqlite3changeset_old(iter.ptr, C.int(col), &v.ptr)
	if err := reserr("ChangesetIter.Old", "", "", res); err != nil {
		return Value{}, err
	}
	return v, nil
}

// New obtains new row values from an iterator.
//
// https://www.sqlite.org/session/sqlite3changeset_new.html
func (iter ChangesetIter) New(col int) (v Value, err error) {
	res := C.sqlite3changeset_new(iter.ptr, C.int(col), &v.ptr)
	if err := reserr("ChangesetIter.New", "", "", res); err != nil {
		return Value{}, err
	}
	return v, nil
}

// Conflict obtains conflicting row values from an iterator.
// Only use this in an iterator passed to a ChangesetApply conflictFn.
//
// https://www.sqlite.org/session/sqlite3changeset_conflict.html
func (iter ChangesetIter) Conflict(col int) (v Value, err error) {
	res := C.sqlite3changeset_conflict(iter.ptr, C.int(col), &v.ptr)
	if err := reserr("ChangesetIter.Conflict", "", "", res); err != nil {
		return Value{}, err
	}
	return v, nil
}

// Next moves a changeset iterator forward.
// Do not use in iterators passed to a ChangesetApply conflictFn.
//
// https://www.sqlite.org/session/sqlite3changeset_next.html
func (iter ChangesetIter) Next() (rowReturned bool, err error) {
	switch res := C.sqlite3changeset_next(iter.ptr); res {
	case C.SQLITE_ROW:
		return true, nil
	case C.SQLITE_DONE:
		return false, nil
	default:
		return false, reserr("ChangesetIter.Next", "", "", res)
	}
}

// Op reports details about the current operation in the iterator.
//
// https://www.sqlite.org/session/sqlite3changeset_op.html
func (iter ChangesetIter) Op() (table string, numCols int, opType OpType, indirect bool, err error) {
	var pzTab *C.char
	var pnCol, pOp, pbIndirect C.int
	res := C.sqlite3changeset_op(iter.ptr, &pzTab, &pnCol, &pOp, &pbIndirect)
	if err := reserr("ChangesetIter.Op", "", "", res); err != nil {
		return "", 0, 0, false, err
	}
	table = C.GoString(pzTab)
	numCols = int(pnCol)
	opType = OpType(pOp)
	if pbIndirect != 0 {
		indirect = true
	}
	return table, numCols, opType, indirect, nil
}

// FKConflicts reports the number of foreign key constraint violations.
//
// https://www.sqlite.org/session/sqlite3changeset_fk_conflicts.html
func (iter ChangesetIter) FKConflicts() (int, error) {
	var pnOut C.int
	res := C.sqlite3changeset_fk_conflicts(iter.ptr, &pnOut)
	if err := reserr("ChangesetIter.FKConflicts", "", "", res); err != nil {
		return 0, err
	}
	return int(pnOut), nil
}

// PK reports the columns that make up the primary key.
//
// https://www.sqlite.org/session/sqlite3changeset_pk.html
func (iter ChangesetIter) PK() ([]bool, error) {
	var pabPK *C.uchar
	var pnCol C.int
	res := C.sqlite3changeset_pk(iter.ptr, &pabPK, &pnCol)
	if err := reserr("ChangesetIter.PK", "", "", res); err != nil {
		return nil, err
	}
	vals := (*[127]byte)(unsafe.Pointer(pabPK))[:pnCol:pnCol]
	cols := make([]bool, pnCol)
	for i, val := range vals {
		if val != 0 {
			cols[i] = true
		}
	}
	return cols, nil
}

// OpType is an enumeration of SQLite statements. Used for authorization and
// changeset details.
type OpType int

// Operation types
const (
	SQLITE_CREATE_INDEX        OpType = C.SQLITE_CREATE_INDEX
	SQLITE_CREATE_TABLE        OpType = C.SQLITE_CREATE_TABLE
	SQLITE_CREATE_TEMP_INDEX   OpType = C.SQLITE_CREATE_TEMP_INDEX
	SQLITE_CREATE_TEMP_TABLE   OpType = C.SQLITE_CREATE_TEMP_TABLE
	SQLITE_CREATE_TEMP_TRIGGER OpType = C.SQLITE_CREATE_TEMP_TRIGGER
	SQLITE_CREATE_TEMP_VIEW    OpType = C.SQLITE_CREATE_TEMP_VIEW
	SQLITE_CREATE_TRIGGER      OpType = C.SQLITE_CREATE_TRIGGER
	SQLITE_CREATE_VIEW         OpType = C.SQLITE_CREATE_VIEW
	SQLITE_DELETE              OpType = C.SQLITE_DELETE
	SQLITE_DROP_INDEX          OpType = C.SQLITE_DROP_INDEX
	SQLITE_DROP_TABLE          OpType = C.SQLITE_DROP_TABLE
	SQLITE_DROP_TEMP_INDEX     OpType = C.SQLITE_DROP_TEMP_INDEX
	SQLITE_DROP_TEMP_TABLE     OpType = C.SQLITE_DROP_TEMP_TABLE
	SQLITE_DROP_TEMP_TRIGGER   OpType = C.SQLITE_DROP_TEMP_TRIGGER
	SQLITE_DROP_TEMP_VIEW      OpType = C.SQLITE_DROP_TEMP_VIEW
	SQLITE_DROP_TRIGGER        OpType = C.SQLITE_DROP_TRIGGER
	SQLITE_DROP_VIEW           OpType = C.SQLITE_DROP_VIEW
	SQLITE_INSERT              OpType = C.SQLITE_INSERT
	SQLITE_PRAGMA              OpType = C.SQLITE_PRAGMA
	SQLITE_READ                OpType = C.SQLITE_READ
	SQLITE_SELECT              OpType = C.SQLITE_SELECT
	SQLITE_TRANSACTION         OpType = C.SQLITE_TRANSACTION
	SQLITE_UPDATE              OpType = C.SQLITE_UPDATE
	SQLITE_ATTACH              OpType = C.SQLITE_ATTACH
	SQLITE_DETACH              OpType = C.SQLITE_DETACH
	SQLITE_ALTER_TABLE         OpType = C.SQLITE_ALTER_TABLE
	SQLITE_REINDEX             OpType = C.SQLITE_REINDEX
	SQLITE_ANALYZE             OpType = C.SQLITE_ANALYZE
	SQLITE_CREATE_VTABLE       OpType = C.SQLITE_CREATE_VTABLE
	SQLITE_DROP_VTABLE         OpType = C.SQLITE_DROP_VTABLE
	SQLITE_FUNCTION            OpType = C.SQLITE_FUNCTION
	SQLITE_SAVEPOINT           OpType = C.SQLITE_SAVEPOINT
	SQLITE_COPY                OpType = C.SQLITE_COPY
	SQLITE_RECURSIVE           OpType = C.SQLITE_RECURSIVE
)

func (opType OpType) String() string {
	switch opType {
	default:
		var buf [20]byte
		return "SQLITE_UNKNOWN_OP_TYPE(" + string(itoa(buf[:], int64(opType))) + ")"
	case SQLITE_CREATE_INDEX:
		return "SQLITE_CREATE_INDEX"
	case SQLITE_CREATE_TABLE:
		return "SQLITE_CREATE_TABLE"
	case SQLITE_CREATE_TEMP_INDEX:
		return "SQLITE_CREATE_TEMP_INDEX"
	case SQLITE_CREATE_TEMP_TABLE:
		return "SQLITE_CREATE_TEMP_TABLE"
	case SQLITE_CREATE_TEMP_TRIGGER:
		return "SQLITE_CREATE_TEMP_TRIGGER"
	case SQLITE_CREATE_TEMP_VIEW:
		return "SQLITE_CREATE_TEMP_VIEW"
	case SQLITE_CREATE_TRIGGER:
		return "SQLITE_CREATE_TRIGGER"
	case SQLITE_CREATE_VIEW:
		return "SQLITE_CREATE_VIEW"
	case SQLITE_DELETE:
		return "SQLITE_DELETE"
	case SQLITE_DROP_INDEX:
		return "SQLITE_DROP_INDEX"
	case SQLITE_DROP_TABLE:
		return "SQLITE_DROP_TABLE"
	case SQLITE_DROP_TEMP_INDEX:
		return "SQLITE_DROP_TEMP_INDEX"
	case SQLITE_DROP_TEMP_TABLE:
		return "SQLITE_DROP_TEMP_TABLE"
	case SQLITE_DROP_TEMP_TRIGGER:
		return "SQLITE_DROP_TEMP_TRIGGER"
	case SQLITE_DROP_TEMP_VIEW:
		return "SQLITE_DROP_TEMP_VIEW"
	case SQLITE_DROP_TRIGGER:
		return "SQLITE_DROP_TRIGGER"
	case SQLITE_DROP_VIEW:
		return "SQLITE_DROP_VIEW"
	case SQLITE_INSERT:
		return "SQLITE_INSERT"
	case SQLITE_PRAGMA:
		return "SQLITE_PRAGMA"
	case SQLITE_READ:
		return "SQLITE_READ"
	case SQLITE_SELECT:
		return "SQLITE_SELECT"
	case SQLITE_TRANSACTION:
		return "SQLITE_TRANSACTION"
	case SQLITE_UPDATE:
		return "SQLITE_UPDATE"
	case SQLITE_ATTACH:
		return "SQLITE_ATTACH"
	case SQLITE_DETACH:
		return "SQLITE_DETACH"
	case SQLITE_ALTER_TABLE:
		return "SQLITE_ALTER_TABLE"
	case SQLITE_REINDEX:
		return "SQLITE_REINDEX"
	case SQLITE_ANALYZE:
		return "SQLITE_ANALYZE"
	case SQLITE_CREATE_VTABLE:
		return "SQLITE_CREATE_VTABLE"
	case SQLITE_DROP_VTABLE:
		return "SQLITE_DROP_VTABLE"
	case SQLITE_FUNCTION:
		return "SQLITE_FUNCTION"
	case SQLITE_SAVEPOINT:
		return "SQLITE_SAVEPOINT"
	case SQLITE_COPY:
		return "SQLITE_COPY"
	case SQLITE_RECURSIVE:
		return "SQLITE_RECURSIVE"
	}
}

type ConflictType int

const (
	SQLITE_CHANGESET_DATA        = ConflictType(C.SQLITE_CHANGESET_DATA)
	SQLITE_CHANGESET_NOTFOUND    = ConflictType(C.SQLITE_CHANGESET_NOTFOUND)
	SQLITE_CHANGESET_CONFLICT    = ConflictType(C.SQLITE_CHANGESET_CONFLICT)
	SQLITE_CHANGESET_CONSTRAINT  = ConflictType(C.SQLITE_CHANGESET_CONSTRAINT)
	SQLITE_CHANGESET_FOREIGN_KEY = ConflictType(C.SQLITE_CHANGESET_FOREIGN_KEY)
)

func (code ConflictType) String() string {
	switch code {
	default:
		var buf [20]byte
		return "SQLITE_UNKNOWN_CONFLICT_TYPE(" + string(itoa(buf[:], int64(code))) + ")"
	case SQLITE_CHANGESET_DATA:
		return "SQLITE_CHANGESET_DATA"
	case SQLITE_CHANGESET_NOTFOUND:
		return "SQLITE_CHANGESET_NOTFOUND"
	case SQLITE_CHANGESET_CONFLICT:
		return "SQLITE_CHANGESET_CONFLICT"
	case SQLITE_CHANGESET_CONSTRAINT:
		return "SQLITE_CHANGESET_CONSTRAINT"
	case SQLITE_CHANGESET_FOREIGN_KEY:
		return "SQLITE_CHANGESET_FOREIGN_KEY"
	}
}

type ConflictAction int

const (
	SQLITE_CHANGESET_OMIT    = ConflictAction(C.SQLITE_CHANGESET_OMIT)
	SQLITE_CHANGESET_ABORT   = ConflictAction(C.SQLITE_CHANGESET_ABORT)
	SQLITE_CHANGESET_REPLACE = ConflictAction(C.SQLITE_CHANGESET_REPLACE)
)

func (code ConflictAction) String() string {
	switch code {
	default:
		var buf [20]byte
		return "SQLITE_UNKNOWN_CONFLICT_ACTION(" + string(itoa(buf[:], int64(code))) + ")"
	case SQLITE_CHANGESET_OMIT:
		return "SQLITE_CHANGESET_OMIT"
	case SQLITE_CHANGESET_ABORT:
		return "SQLITE_CHANGESET_ABORT"
	case SQLITE_CHANGESET_REPLACE:
		return "SQLITE_CHANGESET_REPLACE"
	}
}

type Changegroup struct {
	ptr *C.sqlite3_changegroup
}

// https://www.sqlite.org/session/sqlite3changegroup_new.html
func NewChangegroup() (*Changegroup, error) {
	c := &Changegroup{}
	res := C.sqlite3changegroup_new(&c.ptr)
	if err := reserr("NewChangegroup", "", "", res); err != nil {
		return nil, err
	}
	return c, nil
}

// https://www.sqlite.org/session/sqlite3changegroup_add.html
func (cg Changegroup) Add(r io.Reader) error {
	xIn := newStrm(nil, r)
	res := C.go_sqlite3changegroup_add_strm(
		cg.ptr,
		(*[0]byte)(C.c_strm_r_tramp),
		xIn.cptr())
	xIn.free()
	return reserr("Changegroup.Add", "", "", res)
}

// Delete a Changegroup.
//
// https://www.sqlite.org/session/sqlite3changegroup_delete.html
func (cg Changegroup) Delete() {
	C.sqlite3changegroup_delete(cg.ptr)
}

// https://www.sqlite.org/session/sqlite3changegroup_output.html
func (cg Changegroup) Output(w io.Writer) (n int, err error) {
	xOut := newStrm(w, nil)
	res := C.go_sqlite3changegroup_output_strm(
		cg.ptr,
		(*[0]byte)(C.c_strm_w_tramp),
		xOut.cptr())
	n = xOut.n
	xOut.free()
	return n, reserr("Changegroup.Output", "", "", res)
}

type strm struct {
	id int
	w  io.Writer // one of w or r is set
	r  io.Reader
	n  int // number of bytes read or written
}

var strms = struct {
	mu   sync.RWMutex
	m    map[int]*strm
	next int
}{
	m: make(map[int]*strm),
}

func newStrm(w io.Writer, r io.Reader) *strm {
	x := &strm{w: w, r: r}

	strms.mu.Lock()
	strms.next++
	x.id = strms.next
	strms.m[x.id] = x
	strms.mu.Unlock()

	return x
}

func (x strm) free() {
	strms.mu.Lock()
	delete(strms.m, x.id)
	strms.mu.Unlock()
}

func (x *strm) cptr() C.uintptr_t { return (C.uintptr_t)(x.id) }

func getStrm(cptr uintptr) *strm {
	strms.mu.RLock()
	x := strms.m[int(cptr)]
	strms.mu.RUnlock()

	return x
}

//export go_strm_w_tramp
func go_strm_w_tramp(pOut uintptr, pData *C.char, n C.int) C.int {
	//println("go_strm_w_tramp start")
	x := getStrm(pOut)
	b := (*[1 << 30]byte)(unsafe.Pointer(pData))[:n:n]
	for len(b) > 0 {
		nw, err := x.w.Write(b)
		x.n += nw
		b = b[nw:]

		if err != nil {
			if code := ErrCode(err); code != SQLITE_ERROR {
				return C.int(code)
			}
			return C.SQLITE_IOERR
		}
	}
	//println("go_strm_w_tramp OK, nw=", nw)
	return C.SQLITE_OK
}

//export go_strm_r_tramp
func go_strm_r_tramp(pIn uintptr, pData *C.char, pnData *C.int) C.int {
	x := getStrm(pIn)
	b := (*[1 << 30]byte)(unsafe.Pointer(pData))[:*pnData:*pnData]

	var n int
	var err error
	for n == 0 && err == nil {
		// Technically an io.Reader is allowed to return (0, nil)
		// and it is not treated as the end of the stream.
		//
		// So we spin here until the io.Reader is gracious enough
		// to get off its butt and actually do something.
		n, err = x.r.Read(b)
	}

	x.n += n
	//println("*pnData:", *pnData, "n:", n)
	*pnData = C.int(n)
	if err != nil && err != io.EOF {
		if code := ErrCode(err); code != SQLITE_ERROR {
			return C.int(code)
		}
		return C.SQLITE_IOERR
	}
	return C.SQLITE_OK
}

type xapply struct {
	id         int
	conn       *Conn
	filterFn   func(string) bool
	conflictFn func(ConflictType, ChangesetIter) ConflictAction
}

var xapplys = struct {
	mu   sync.RWMutex
	m    map[int]*xapply
	next int
}{
	m: make(map[int]*xapply),
}

//export go_xapply_filter_tramp
func go_xapply_filter_tramp(pCtx uintptr, zTab *C.char) C.int {
	xapplys.mu.Lock()
	x, ok := xapplys.m[int(pCtx)]
	xapplys.mu.Unlock()

	if !ok {
		panic("not ok")
	}
	if x == nil {
		panic("x == nil")
	}

	tableName := C.GoString(zTab)
	if x.filterFn(tableName) {
		return 1
	}
	return 0
}

//export go_xapply_conflict_tramp
func go_xapply_conflict_tramp(pCtx uintptr, eConflict C.int, p *C.sqlite3_changeset_iter) C.int {
	xapplys.mu.Lock()
	x := xapplys.m[int(pCtx)]
	xapplys.mu.Unlock()

	action := x.conflictFn(ConflictType(eConflict), ChangesetIter{ptr: p})
	return C.int(action)
}
