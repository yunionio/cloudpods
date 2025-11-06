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

// #include <sqlite3.h>
// #include <stdlib.h>
import "C"
import (
	"runtime"
	"unsafe"
)

// A Snapshot records the state of a WAL mode database for some specific point
// in history.
//
// Equivalent to the sqlite3_snapshot* C object.
//
// https://www.sqlite.org/c3ref/snapshot.html
type Snapshot struct {
	ptr    *C.sqlite3_snapshot
	schema *C.char
}

// GetSnapshot attempts to make a new Snapshot that records the current state
// of the given schema in conn. If successful, a *Snapshot and a func() is
// returned, and the conn will have an open READ transaction which will
// continue to reflect the state of the Snapshot until the returned func() is
// called. No WRITE transaction may occur on conn until the returned func() is
// called.
//
// The returned *Snapshot is threadsafe for creating additional read
// transactions that reflect its state with Conn.StartSnapshotRead.
//
// In theory, so long as at least one read transaction is open on the Snapshot,
// then the WAL file will not be checkpointed past that point, and the Snapshot
// will continue to be available for creating additional read transactions.
// However, if no read transaction is open on the Snapshot, then it is possible
// for the WAL to be checkpointed past the point of the Snapshot. If this
// occurs then there is no way to start a read on the Snapshot. In order to
// ensure that a Snapshot remains readable, always maintain at least one open
// read transaction on the Snapshot.
//
// In practice, this is generally reliable but sometimes the Snapshot can
// sometimes become unavailable for reads unless automatic checkpointing is
// entirely disabled from the start.
//
// The returned *Snapshot has a finalizer that calls Free if it has not been
// called, so it is safe to allow a Snapshot to be garbage collected. However,
// if you are sure that a Snapshot will never be used again by any thread, you
// may call Free once to release the memory earlier. No reads will be possible
// on the Snapshot after Free is called on it, however any open read
// transactions will not be interrupted.
//
// See sqlitex.Pool.GetSnapshot for a helper function for automatically keeping
// an open read transaction on a set aside connection until a Snapshot is GC'd.
//
// The following must be true for this function to succeed:
//
// - The schema of conn must be a WAL mode database.
//
// - There must not be any transaction open on schema of conn.
//
// - At least one transaction must have been written to the current WAL file
// since it was created on disk (by any connection). You can run the following
// SQL to ensure that a WAL file has been created.
//
//	BEGIN IMMEDIATE;
//	COMMIT;
//
// https://www.sqlite.org/c3ref/snapshot_get.html
func (conn *Conn) GetSnapshot(schema string) (*Snapshot, func(), error) {
	var s Snapshot
	if schema == "" || schema == "main" {
		s.schema = cmain
	} else {
		s.schema = C.CString(schema)
	}

	endRead, err := conn.disableAutoCommitMode()
	if err != nil {
		return nil, nil, err
	}

	res := C.sqlite3_snapshot_get(conn.conn, s.schema, &s.ptr)
	if res != 0 {
		endRead()
		return nil, nil, reserr("Conn.CreateSnapshot", "", "", res)
	}

	runtime.SetFinalizer(&s, func(s *Snapshot) {
		s.Free()
	})

	return &s, endRead, nil
}

// Free destroys a Snapshot. Free is not threadsafe but may be called more than
// once. However, it is not necessary to call Free on a Snapshot returned by
// conn.GetSnapshot or pool.GetSnapshot as these set a finalizer that calls
// free which will be run automatically by the GC in a finalizer. However if it
// is guaranteed that a Snapshot will never be used again, calling Free will
// allow memory to be freed earlier.
//
// A Snapshot may become unavailable for reads before Free is called if the WAL
// is checkpointed into the DB past the point of the Snapshot.
//
// https://www.sqlite.org/c3ref/snapshot_free.html
func (s *Snapshot) Free() {
	if s.ptr == nil {
		return
	}
	C.sqlite3_snapshot_free(s.ptr)
	if s.schema != cmain {
		C.free(unsafe.Pointer(s.schema))
	}
	s.ptr = nil
}

// CompareAges returns whether s1 is older, newer or the same age as s2. Age
// refers to writes on the database, not time since creation.
//
// If s is older than s2, a negative number is returned. If s and s2 are the
// same age, zero is returned. If s is newer than s2, a positive number is
// returned.
//
// The result is valid only if both of the following are true:
//
// - The two snapshot handles are associated with the same database file.
//
// - Both of the Snapshots were obtained since the last time the wal file was
// deleted.
//
// https://www.sqlite.org/c3ref/snapshot_cmp.html
func (s *Snapshot) CompareAges(s2 *Snapshot) int {
	return int(C.sqlite3_snapshot_cmp(s.ptr, s2.ptr))
}

// StartSnapshotRead starts a new read transaction on conn such that the read
// transaction refers to historical Snapshot s, rather than the most recent
// change to the database.
//
// There must be no open transaction on conn. Free must not have been called on
// s prior to or during this function call.
//
// If err is nil, then endRead is a function that will end the read transaction
// and return conn to its original state. Until endRead is called, no writes
// may occur on conn, and all reads on conn will refer to the Snapshot.
//
// https://www.sqlite.org/c3ref/snapshot_open.html
func (conn *Conn) StartSnapshotRead(s *Snapshot) (endRead func(), err error) {
	endRead, err = conn.disableAutoCommitMode()
	if err != nil {
		return
	}

	res := C.sqlite3_snapshot_open(conn.conn, s.schema, s.ptr)
	if res != 0 {
		endRead()
		return nil, reserr("Conn.StartSnapshotRead", "", "", res)
	}

	return endRead, nil
}

// disableAutoCommitMode starts a read transaction with `BEGIN;`, disabling
// autocommit mode, and returns a function which when called will end the read
// transaction with `ROLLBACK;`, re-enabling autocommit mode.
//
// https://sqlite.org/c3ref/get_autocommit.html
func (conn *Conn) disableAutoCommitMode() (func(), error) {
	begin := conn.Prep("BEGIN;")
	defer begin.Reset()
	if _, err := begin.Step(); err != nil {
		return nil, err
	}
	rollback := conn.Prep("ROLLBACK;")
	return func() {
		defer rollback.Reset()
		rollback.Step()
	}, nil
}
