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

/*
Package sqlite provides a Go interface to SQLite 3.

The semantics of this package are deliberately close to the
SQLite3 C API, so it is helpful to be familiar with
http://www.sqlite.org/c3ref/intro.html.

An SQLite connection is represented by a *sqlite.Conn.
Connections cannot be used concurrently.
A typical Go program will create a pool of connections
(using Open to create a *sqlitex.Pool) so goroutines can
borrow a connection while they need to talk to the database.

This package assumes SQLite will be used concurrently by the
process through several connections, so the build options for
SQLite enable multi-threading and the shared cache:
https://www.sqlite.org/sharedcache.html

The implementation automatically handles shared cache locking,
see the documentation on Stmt.Step for details.

The optional SQLite3 compiled in are: FTS5, RTree, JSON1, Session, GeoPoly

This is not a database/sql driver.

# Statement Caching

Statements are prepared with the Prepare and PrepareTransient methods.
When using Prepare, statements are keyed inside a connection by the
original query string used to create them. This means long-running
high-performance code paths can write:

	stmt, err := conn.Prepare("SELECT ...")

After all the connections in a pool have been warmed up by passing
through one of these Prepare calls, subsequent calls are simply a
map lookup that returns an existing statement.

# Streaming Blobs

The sqlite package supports the SQLite incremental I/O interface for
streaming blob data into and out of the the database without loading
the entire blob into a single []byte.
(This is important when working either with very large blobs, or
more commonly, a large number of moderate-sized blobs concurrently.)

To write a blob, first use an INSERT statement to set the size of the
blob and assign a rowid:

	"INSERT INTO blobs (myblob) VALUES (?);"

Use BindZeroBlob or SetZeroBlob to set the size of myblob.
Then you can open the blob with:

	b, err := conn.OpenBlob("", "blobs", "myblob", conn.LastInsertRowID(), true)

# Deadlines and Cancellation

Every connection can have a done channel associated with it using
the SetInterrupt method. This is typically the channel returned by
a context.Context Done method.

For example, a timeout can be associated with a connection session:

	ctx := context.WithTimeout(context.Background(), 100*time.Millisecond)
	conn.SetInterrupt(ctx.Done())

As database connections are long-lived, the SetInterrupt method can
be called multiple times to reset the associated lifetime.

When using pools, the shorthand for associating a context with a
connection is:

	conn := dbpool.Get(ctx)
	if conn == nil {
		// ... handle error
	}
	defer dbpool.Put(c)

# Transactions

SQLite transactions have to be managed manually with this package
by directly calling BEGIN / COMMIT / ROLLBACK or
SAVEPOINT / RELEASE/ ROLLBACK. The sqlitex has a Savepoint
function that helps automate this.

# A typical HTTP Handler

Using a Pool to execute SQL in a concurrent HTTP handler.

	var dbpool *sqlitex.Pool

	func main() {
		var err error
		dbpool, err = sqlitex.Open("file:memory:?mode=memory", 0, 10)
		if err != nil {
			log.Fatal(err)
		}
		http.HandleFunc("/", handle)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}

	func handle(w http.ResponseWriter, r *http.Request) {
		conn := dbpool.Get(r.Context())
		if conn == nil {
			return
		}
		defer dbpool.Put(conn)
		stmt := conn.Prep("SELECT foo FROM footable WHERE id = $id;")
		stmt.SetText("$id", "_user_id_")
		for {
			if hasRow, err := stmt.Step(); err != nil {
				// ... handle error
			} else if !hasRow {
				break
			}
			foo := stmt.GetText("foo")
			// ... use foo
		}
	}

For helper functions that make some kinds of statements easier to
write see the sqlitex package.
*/
package sqlite // import "github.com/go-llsqlite/crawshaw"
