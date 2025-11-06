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

// #include <blocking_step.h>
// #include <sqlite3.h>
// #include <stdlib.h>
// #include <stdint.h>
import "C"
import (
	"errors"
	"fmt"
	"io"
	"unsafe"
)

var cmain = C.CString("main")
var ctemp = C.CString("temp")

// OpenBlob opens a blob in a particular {database,table,column,row}.
//
// https://www.sqlite.org/c3ref/blob_open.html
func (conn *Conn) OpenBlob(dbn, table, column string, row int64, write bool) (*Blob, error) {
	var cdb *C.char
	switch dbn {
	case "", "main":
		cdb = cmain
	case "temp":
		cdb = ctemp
	default:
		cdb = C.CString(dbn)
		defer C.free(unsafe.Pointer(cdb))
	}
	var flags C.int
	if write {
		flags = 1
	}

	ctable := C.CString(table)
	ccolumn := C.CString(column)
	defer func() {
		C.free(unsafe.Pointer(ctable))
		C.free(unsafe.Pointer(ccolumn))
	}()

	blob := &Blob{conn: conn}

	for {
		conn.count++
		if err := conn.interrupted("Conn.OpenBlob", ""); err != nil {
			return nil, err
		}
		switch res := C.sqlite3_blob_open(conn.conn, cdb, ctable, ccolumn,
			C.sqlite3_int64(row), flags, &blob.blob); res {
		case C.SQLITE_LOCKED_SHAREDCACHE:
			if res := C.wait_for_unlock_notify(
				conn.conn, conn.unlockNote); res != C.SQLITE_OK {
				return nil, conn.reserr("Conn.OpenBlob(Wait)", "", res)
			}
			// loop
		case C.SQLITE_OK:
			blob.size = int64(C.sqlite3_blob_bytes(blob.blob))
			return blob, nil
		default:
			return nil, conn.extreserr("Conn.OpenBlob", "", res)
		}
	}
}

// Blob provides streaming access to SQLite blobs.
type Blob struct {
	conn *Conn
	blob *C.sqlite3_blob
	size int64
}

func (blob *Blob) Reopen(rowid int64) (err error) {
	rc := C.sqlite3_blob_reopen(blob.blob, C.sqlite3_int64(rowid))
	err = blob.conn.reserr("Blob.Reopen", "", rc)
	if err != nil {
		return
	}
	blob.setSize()
	return
}

func (blob *Blob) setSize() {
	blob.size = int64(C.sqlite3_blob_bytes(blob.blob))
}

// https://www.sqlite.org/c3ref/blob_read.html
func (blob *Blob) ReadAt(p []byte, off int64) (n int, err error) {
	if blob.blob == nil {
		return 0, ErrBlobClosed
	}
	if off < 0 {
		err = fmt.Errorf("bad offset %v", off)
		return
	}
	if off >= blob.size {
		err = io.EOF
		return
	}
	if err := blob.conn.interrupted("Blob.ReadAt", ""); err != nil {
		return 0, err
	}
	if int64(len(p)) > blob.size-off {
		p = p[:blob.size-off]
	}
	lenp := C.int(len(p))
	res := C.sqlite3_blob_read(blob.blob, unsafe.Pointer(&p[0]), lenp, C.int(off))
	if err := blob.conn.reserr("Blob.ReadAt", "", res); err != nil {
		return 0, err
	}
	n = len(p)
	if off+int64(len(p)) >= blob.size {
		err = io.EOF
	}
	return
}

// https://www.sqlite.org/c3ref/blob_write.html
func (blob *Blob) WriteAt(p []byte, off int64) (n int, err error) {
	if blob.blob == nil {
		return 0, ErrBlobClosed
	}
	if err := blob.conn.interrupted("Blob.WriteAt", ""); err != nil {
		return 0, err
	}
	lenp := C.int(len(p))
	res := C.sqlite3_blob_write(blob.blob, unsafe.Pointer(&p[0]), lenp, C.int(off))
	if err := blob.conn.reserr("Blob.WriteAt", "", res); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Size returns the total size of a blob.
func (blob *Blob) Size() int64 {
	return blob.size
}

// https://www.sqlite.org/c3ref/blob_close.html
func (blob *Blob) Close() error {
	if blob.blob == nil {
		return ErrBlobClosed
	}
	err := blob.conn.reserr("Blob.Close", "", C.sqlite3_blob_close(blob.blob))
	blob.blob = nil
	return err
}

var ErrBlobClosed = errors.New("blob closed")
