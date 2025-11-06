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

// A Backup copies data between two databases.
//
// It is used to backup file based or in-memory databases.
//
// Equivalent to the sqlite3_backup* C object.
//
// https://www.sqlite.org/c3ref/backup_finish.html
type Backup struct {
	ptr *C.sqlite3_backup
}

// BackupToDB creates a complete backup of the srcDB on the src Conn to a new
// database Conn at dstPath. The resulting dst connection is returned. This
// will block until the entire backup is complete.
//
// If srcDB is "", then a default of "main" is used.
//
// This is very similar to the first example function implemented on the
// following page.
//
// https://www.sqlite.org/backup.html
func (src *Conn) BackupToDB(srcDB, dstPath string) (dst *Conn, err error) {
	if dst, err = OpenConn(dstPath, 0); err != nil {
		return
	}
	defer func() {
		if err != nil {
			dst.Close()
		}
	}()
	b, err := src.BackupInit(srcDB, "", dst)
	if err != nil {
		return
	}
	defer b.Finish()
	err = b.Step(-1)
	return
}

// BackupInit initializes a new Backup object to copy from src to dst.
//
// If srcDB or dstDB is "", then a default of "main" is used.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupinit
func (src *Conn) BackupInit(srcDB, dstDB string, dst *Conn) (*Backup, error) {
	var srcCDB, dstCDB *C.char
	defer setCDB(dstDB, &dstCDB)()
	defer setCDB(srcDB, &srcCDB)()
	var b Backup
	b.ptr = C.sqlite3_backup_init(dst.conn, dstCDB, src.conn, srcCDB)
	if b.ptr == nil {
		res := C.sqlite3_errcode(dst.conn)
		return nil, dst.extreserr("Conn.BackupInit", "", res)
	}
	runtime.SetFinalizer(&b, func(b *Backup) {
		if b.ptr != nil {
			panic("open *sqlite.Backup garbage collected, call Finish method")
		}
	})

	return &b, nil
}
func setCDB(db string, cdb **C.char) func() {
	if db == "" || db == "main" {
		*cdb = cmain
		return func() {}
	}
	*cdb = C.CString(db)
	return func() { C.free(unsafe.Pointer(cdb)) }
}

// Step is called one or more times to transfer nPage pages at a time between
// databases.
//
// Use -1 to transfer the entire database at once.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupstep
func (b *Backup) Step(nPage int) error {
	res := C.sqlite3_backup_step(b.ptr, C.int(nPage))
	if res != C.SQLITE_DONE {
		return reserr("Backup.Step", "", "", res)
	}
	return nil
}

// Finish is called to clean up the resources allocated by BackupInit.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupfinish
func (b *Backup) Finish() error {
	res := C.sqlite3_backup_finish(b.ptr)
	b.ptr = nil
	return reserr("Backup.Finish", "", "", res)
}

// Remaining returns the number of pages still to be backed up at the
// conclusion of the most recent b.Step().
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupremaining
func (b *Backup) Remaining() int {
	return int(C.sqlite3_backup_remaining(b.ptr))
}

// PageCount returns the total number of pages in the source database at the
// conclusion of the most recent b.Step().
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backuppagecount
func (b *Backup) PageCount() int {
	return int(C.sqlite3_backup_pagecount(b.ptr))
}
