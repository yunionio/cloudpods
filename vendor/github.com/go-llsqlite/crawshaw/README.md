# Low-level Go interface to SQLite

[![Go Reference](https://pkg.go.dev/badge/github.com/go-llsqlite/llsqlite.svg)](https://pkg.go.dev/github.com/go-llsqlite/llsqlite)

This project is a community-managed fork of https://github.com/crawshaw/sqlite.

This package provides a low-level Go interface to SQLite 3. Connections are [pooled](https://pkg.go.dev/github.com/go-llsqlite/llsqlite#Pool) and if the SQLite [shared cache](https://www.sqlite.org/sharedcache.html) mode is enabled the package takes advantage of the [unlock-notify API](https://www.sqlite.org/unlock_notify.html) to minimize the amount of handling user code needs for dealing with database lock contention.

It has interfaces for some of SQLite's more interesting extensions, such as [incremental BLOB I/O](https://www.sqlite.org/c3ref/blob_open.html) and the [session extension](https://www.sqlite.org/sessionintro.html).

A utility package, [sqlitex](https://pkg.go.dev/github.com/go-llsqlite/llsqlite/sqlitex), provides some higher-level tools for making it easier to perform common tasks with SQLite. In particular it provides support to make nested transactions easy to use via [sqlitex.Save](https://pkg.go.dev/github.com/go-llsqlite/llsqlite/sqlitex#Save).

This is not a database/sql driver.

```go get -u github.com/go-llsqlite/llsqlite```

## Example

A HTTP handler that uses a multi-threaded pool of SQLite connections via a shared cache.

```go
var dbpool *sqlitex.Pool

func main() {
	var err error
	dbpool, err = sqlitex.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
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
```

https://pkg.go.dev/github.com/go-llsqlite/llsqlite

## Platform specific considerations

By default it requires some pthreads DLL on Windows. To avoid it, supply `CGO_LDFLAGS="-static"` when building your application.
