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

import "unsafe"

// #include <stdlib.h>
// #include <sqlite3.h>
// static int db_config_onoff(sqlite3* db, int op, int onoff) {
//   return sqlite3_db_config(db, op, onoff, NULL);
// }
import "C"

const (
	SQLITE_DBCONFIG_ENABLE_LOAD_EXTENSION = C.int(C.SQLITE_DBCONFIG_ENABLE_LOAD_EXTENSION)
)

// EnableLoadExtension allows extensions to be loaded via LoadExtension().  The
// SQL interface is left disabled as recommended.
//
// https://www.sqlite.org/c3ref/enable_load_extension.html
func (conn *Conn) EnableLoadExtension(on bool) error {
	var enable C.int
	if on {
		enable = 1
	}
	res := C.db_config_onoff(conn.conn, SQLITE_DBCONFIG_ENABLE_LOAD_EXTENSION, enable)
	return reserr("Conn.EnableLoadExtension", "", "", res)
}

// LoadExtension attempts to load a runtime-loadable extension.
//
// https://www.sqlite.org/c3ref/load_extension.html
func (conn *Conn) LoadExtension(ext, entry string) error {
	cext := C.CString(ext)
	defer C.free(unsafe.Pointer(cext))
	var centry *C.char
	if entry != "" {
		centry = C.CString(entry)
		defer C.free(unsafe.Pointer(centry))
	}
	var cerr *C.char
	res := C.sqlite3_load_extension(conn.conn, cext, centry, &cerr)
	err := C.GoString(cerr)
	C.sqlite3_free(unsafe.Pointer(cerr))
	return reserr("Conn.LoadExtension", "", err, res)
}
