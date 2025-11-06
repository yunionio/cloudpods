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

package sqlitex

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/go-llsqlite/crawshaw"
)

// Save creates a named SQLite transaction using SAVEPOINT.
//
// On success Savepoint returns a releaseFn that will call either
// RELEASE or ROLLBACK depending on whether the parameter *error
// points to a nil or non-nil error. This is designed to be deferred.
//
// Example:
//
//	func doWork(conn *sqlite.Conn) (err error) {
//		defer sqlitex.Save(conn)(&err)
//
//		// ... do work in the transaction
//	}
//
// https://www.sqlite.org/lang_savepoint.html
func Save(conn *sqlite.Conn) (releaseFn func(*error)) {
	name := "sqlitex.Save" // safe as names can be reused
	var pc [3]uintptr
	if n := runtime.Callers(0, pc[:]); n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		if _, more := frames.Next(); more { // runtime.Callers
			if _, more := frames.Next(); more { // savepoint.Save
				frame, _ := frames.Next() // caller we care about
				if frame.Function != "" {
					name = frame.Function
				}
			}
		}
	}

	releaseFn, err := savepoint(conn, name)
	if err != nil {
		if sqlite.ErrCode(err) == sqlite.SQLITE_INTERRUPT {
			return func(errp *error) {
				if *errp == nil {
					*errp = err
				}
			}
		}
		panic(err)
	}
	return releaseFn
}

func savepoint(conn *sqlite.Conn, name string) (releaseFn func(*error), err error) {
	if strings.Contains(name, `"`) {
		return nil, fmt.Errorf("sqlitex.Savepoint: invalid name: %q", name)
	}
	if err := Exec(conn, fmt.Sprintf("SAVEPOINT %q;", name), nil); err != nil {
		return nil, err
	}
	tracer := conn.Tracer()
	if tracer != nil {
		tracer.Push("TX " + name)
	}
	releaseFn = func(errp *error) {
		if tracer != nil {
			tracer.Pop()
		}
		recoverP := recover()

		// If a query was interrupted or if a user exec'd COMMIT or
		// ROLLBACK, then everything was already rolled back
		// automatically, thus returning the connection to autocommit
		// mode.
		if conn.GetAutocommit() {
			// There is nothing to rollback.
			if recoverP != nil {
				panic(recoverP)
			}
			return
		}

		if *errp == nil && recoverP == nil {
			// Success path. Release the savepoint successfully.
			*errp = Exec(conn, fmt.Sprintf("RELEASE %q;", name), nil)
			if *errp == nil {
				return
			}
			// Possible interrupt. Fall through to the error path.
			if conn.GetAutocommit() {
				// There is nothing to rollback.
				if recoverP != nil {
					panic(recoverP)
				}
				return
			}
		}

		orig := ""
		if *errp != nil {
			orig = (*errp).Error() + "\n\t"
		}

		// Error path.

		// Always run ROLLBACK even if the connection has been interrupted.
		oldDoneCh := conn.SetInterrupt(nil)
		defer conn.SetInterrupt(oldDoneCh)

		err := Exec(conn, fmt.Sprintf("ROLLBACK TO %q;", name), nil)
		if err != nil {
			panic(orig + err.Error())
		}
		err = Exec(conn, fmt.Sprintf("RELEASE %q;", name), nil)
		if err != nil {
			panic(orig + err.Error())
		}

		if recoverP != nil {
			panic(recoverP)
		}
	}
	return releaseFn, nil
}
