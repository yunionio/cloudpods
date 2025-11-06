# `zombiezen.com/go/sqlite` Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

[Unreleased]: https://github.com/zombiezen/go-sqlite/compare/v0.13.1...main

## [0.13.1][] - 2023-08-15

Version 0.13.1 fixed a bug with the `sqlitemigration` package.

[0.13.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.13.1

### Fixed

- `sqlitemigration` will no longer disable foreign keys during operation
  ([#54](https://github.com/zombiezen/go-sqlite/issues/54)).

## [0.13.0][] - 2023-03-28

Version 0.13 added support for
user-defined [collating sequences](https://www.sqlite.org/datatype3.html#collation)
and user-defined [virtual tables](https://sqlite.org/vtab.html).

[0.13.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.13.0

### Added

- Support user-defined collating sequences
  ([#21](https://github.com/zombiezen/go-sqlite/issues/21)).
- Support user-defined virtual tables
  ([#15](https://github.com/zombiezen/go-sqlite/issues/15)).
- New package `ext/generateseries` provides
  an optional `generate_series` table-valued function extension.
- Exported the `regexp` function example as a new `ext/refunc` package.
- Add `*Conn.Serialize` and `*Conn.Deserialize` methods
  ([#52](https://github.com/zombiezen/go-sqlite/issues/52)).

### Changed

- The minimum supported Go version for this library is now Go 1.19.

### Fixed

- The documentation for `AggregateFunction.WindowValue`
  incorrectly stated that it would not be called in non-window contexts.
  The sentence has been removed, but the behavior has not changed.

## [0.12.0][] - 2023-02-08

Version 0.12 added support for the [online backup API](https://www.sqlite.org/backup.html).

[0.12.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.12.0

### Added

- Added support for the online backup API
  ([#47](https://github.com/zombiezen/go-sqlite/issues/47)).
- Documented the `OpenFlags`.

### Changed

- `OpenNoMutex` and `OpenFullMutex` no longer have an effect on `sqlite.OpenConn`.
  `OpenNoMutex` (i.e. [multi-thread mode](https://www.sqlite.org/threadsafe.html))
  is now the only supported mode.
  `*sqlite.Conn` has never been safe to use concurrently from multiple goroutines,
  so this is mostly to prevent unnecessary locking and to avoid confusion.
  ([#32](https://github.com/zombiezen/go-sqlite/issues/32)).

## [0.11.0][] - 2022-12-11

Version 0.11 changes the aggregate function API.

[0.11.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.11.0

### Changed

- User-defined aggregate functions are now encapsulated
  with a new interface, `AggregateFunction`.
  The previous 4-callback approach has been removed
  and replaced with a single `MakeAggregate` callback.
  Not only was the previous API unwieldy,
  but it failed to handle concurrent aggregate function calls
  in a single query.
- Minimum `modernc.org/sqlite` version updated to 1.20.0.

## [0.10.1][] - 2022-07-17

Version 0.10.1 fixes a bug in user-defined window functions.
Special thanks to Jan Mercl for assistance in debugging this issue.

[0.10.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.10.1

### Fixed

- `AggregateFinal` is now called correctly at the end of window functions' usages.

## [0.10.0][] - 2022-07-10

Version 0.10 adds support for user-defined window functions.

[0.10.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.10.0

### Added

- `FunctionImpl` has two new fields (`WindowValue` and `WindowInverse`)
  that allow creating [user-defined aggregate window functions][]
  ([#42](https://github.com/zombiezen/go-sqlite/issues/42)).

[user-defined aggregate window functions]: https://www.sqlite.org/windowfunctions.html#user_defined_aggregate_window_functions

### Changed

- The `AggregateStep` callback now returns an `error`.

## [0.9.3][] - 2022-05-30

Version 0.9.3 updates the version of `modernc.org/sqlite` used.

[0.9.3]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.9.3

### Changed

- Minimum `modernc.org/sqlite` version updated to v1.17.3.

## [0.9.2][] - 2022-01-25

Version 0.9 adds new `Execute` functions to `sqlitex`
and changes the default blocking behavior.
Version 0.9 also includes various fixes to the schema migration behavior.

[0.9.2]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.9.2

### Added

- Added `SetBlockOnBusy` method to set an indefinite timeout on acquiring a lock.
- Official support for `windows/amd64`.
- `sqlitex` has three new functions —
  `Execute`, `ExecuteTransient`, and `ExecuteScript` —
  that take in an `ExecOptions` struct.
  ([#5](https://github.com/zombiezen/go-sqlite/issues/5))
- New method `sqlite.ResultCode.ToError` to create error values.
- New methods `ColumnBool` and `GetBool` on `*sqlite.Stmt`
  ([#37](https://github.com/zombiezen/go-sqlite/issues/37)).

### Changed

- `OpenConn` calls `SetBlockOnBusy` on new connections
  instead of `SetBusyTimeout(10 * time.Second)`.
- The `sqlitex.Execute*` family of functions now verify that
  the arguments passed match the SQL parameters.
  ([#31](https://github.com/zombiezen/go-sqlite/issues/31))

### Deprecated

- `sqlitex.ExecFS` has been renamed to `sqlitex.ExecuteFS`,
  `sqlitex.ExecTransientFS` has been renamed to `sqlitex.ExecuteTransientFS`,
  and `sqlitex.ExecScriptFS` has been renamed to `sqlitex.ExecuteScriptFS`
  for consistency with the new `Execute` functions.
  Aliases remain in this version, but will be removed in the next version.
  Use `zombiezen-sqlite-migrate` to clean up existing references.
- `sqlitex.Exec` and `sqlitex.ExecTransient`
  have been marked deprecated because they do not perform the argument checks
  that the `Execute` functions now perform.
  These functions will remain into 1.0 and beyond for compatibility,
  but should not be used in new applications.

### Fixed

- `sqlitemigration.Schema.RepeatableMigration` is now run as part of the final transaction.
  This ensures that the repeatable migration for migration `N` has executed
  if and only if `user_version == N`.
  Previously, the repeatable migration could fail independently of the final transaction,
  which would mean that a subsequent migration run would not trigger a retry of the repeatable transaction,
  but report success.
- `sqlitemigration` will no longer skip applying the repeatable migration
  if the final migration is empty.
- `OpenConn` now sets a busy handler before enabling WAL
  (thanks @anacrolix!).

## 0.9.0 and 0.9.1

Versions 0.9.0 was accidentally released before CI ran.
A change in the underlying `modernc.org/libc` library
caused the memory leak detection to identify a false positive.
In an abundance of caution, 0.9.1 was released
to mark both 0.9.1 and 0.9.0 as retracted.
Version 0.9.2 is the first official release of 0.9.

## [0.8.0][] - 2021-11-07

Version 0.8 adds new transaction functions to `sqlitex`.

[0.8.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.8.0

### Added

- Added `sqlitex.Transaction`, `sqlitex.ImmediateTransaction`, and
  `sqlitex.ExclusiveTransaction`.

## [0.7.2][] - 2021-09-11

[0.7.2]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.7.2

### Fixed

- Updated `modernc.org/sqlite` dependency to a released version instead of a
  prerelease

## [0.7.1][] - 2021-09-09

[0.7.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.7.1

### Added

- Added an example to `sqlitemigration.Schema`

## [0.7.0][] - 2021-08-27

[0.7.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.7.0

### Added

- `sqlitemigration.Schema` has a new option for disabling foreign keys for
  individual migrations. This makes it easier to perform migrations that require
  [reconstructing a table][]. ([#20](https://github.com/zombiezen/go-sqlite/issues/20))

[reconstructing a table]: https://sqlite.org/lang_altertable.html#making_other_kinds_of_table_schema_changes

### Changed

- `sqlitemigration.Migrate` and `*sqlitemigration.Pool` no longer use a
  transaction to apply the entire set of migrations: they now only use
  transactions during each individual migration. This was never documented, so
  in theory no one should be depending on this behavior. However, this does mean
  that two processes trying to open and migrate a database concurrently may race
  to apply migrations, whereas before only one process would acquire the write
  lock and migrate.

### Fixed

- Fixed compile breakage on 32-bit architectures. Thanks to Jan Mercl for the
  report.

## [0.6.2][] - 2021-08-17

[0.6.2]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.6.2

### Changed

- `*sqlitex.Pool.Put` now accepts `nil` instead of panicing.
  ([#17](https://github.com/zombiezen/go-sqlite/issues/17))

## [0.6.1][] - 2021-08-16

[0.6.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.6.1

### Fixed

- Fixed a potential memory corruption issue introduced in 0.6.0. Thanks to
  Jan Mercl for the report.

## [0.6.0][] - 2021-08-15

[0.6.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.6.0

### Added

- Added back the session API: `Session`, `ChangesetIterator`, `Changegroup`, and
  various functions. There are some slight naming changes from the
  `crawshaw.io/sqlite` API, but they can all be migrated automatically with the
  migration tool. ([#16](https://github.com/zombiezen/go-sqlite/issues/16))

### Changed

- Method calls to a `nil` `*sqlite.Conn` will return an error rather than panic.
  ([#17](https://github.com/zombiezen/go-sqlite/issues/17))

### Removed

- Removed `OpenFlags` that are only used for VFS.

### Fixed

- Properly clean up WAL when using `sqlitex.Pool`
  ([#14](https://github.com/zombiezen/go-sqlite/issues/14))
- Disabled double-quoted string literals.

## [0.5.0][] - 2021-05-22

[0.5.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.5.0

### Added

- Added `shell` package with basic [REPL][]
- Added `SetAuthorizer`, `Limit`, and `SetDefensive` methods to `*Conn` for use
  in ([#12](https://github.com/zombiezen/go-sqlite/issues/12))
- Added `Version` and `VersionNumber` constants

[REPL]: https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop

### Fixed

- Documented compiled-in extensions ([#11](https://github.com/zombiezen/go-sqlite/issues/11))
- Internal objects are no longer susceptible to ID wraparound issues
  ([#13](https://github.com/zombiezen/go-sqlite/issues/13))

## [0.4.0][] - 2021-05-13

[0.4.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.4.0

### Added

- Add Context.Conn method ([#10](https://github.com/zombiezen/go-sqlite/issues/10))
- Add methods to get and set auxiliary function data
  ([#3](https://github.com/zombiezen/go-sqlite/issues/3))

## [0.3.1][] - 2021-05-03

[0.3.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.3.1

### Fixed

- Fix conversion of BLOB to TEXT when returning BLOB from a user-defined function

## [0.3.0][] - 2021-04-27

[0.3.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.3.0

### Added

- Implement `io.StringWriter`, `io.ReaderFrom`, and `io.WriterTo` on `Blob`
  ([#2](https://github.com/zombiezen/go-sqlite/issues/2))
- Add godoc examples for `Blob`, `sqlitemigration`, and `SetInterrupt`
- Add more README documentation

## [0.2.2][] - 2021-04-24

[0.2.2]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.2.2

### Changed

- Simplified license to [ISC](https://github.com/zombiezen/go-sqlite/blob/v0.2.2/LICENSE)

### Fixed

- Updated version of `modernc.org/sqlite` to 1.10.4 to use [mutex initialization](https://gitlab.com/cznic/sqlite/-/issues/52)
- Fixed doc comment for `BindZeroBlob`

## [0.2.1][] - 2021-04-17

[0.2.1]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.2.1

### Fixed

- Removed bogus import comment

## [0.2.0][] - 2021-04-03

[0.2.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.2.0

### Added

- New migration tool. See [the README](https://github.com/zombiezen/go-sqlite/blob/v0.2.0/cmd/zombiezen-sqlite-migrate/README.md)
  to get started. ([#1](https://github.com/zombiezen/go-sqlite/issues/1))

### Changed

- `*Conn.CreateFunction` has changed entirely. See the
  [reference](https://pkg.go.dev/zombiezen.com/go/sqlite#Conn.CreateFunction)
  for details.
- `sqlitex.File` and `sqlitex.Buffer` have been moved to the `sqlitefile` package
- The `sqlitefile.Exec*` functions have been moved to the `sqlitex` package
  as `Exec*FS`.

## [0.1.0][] - 2021-03-31

Initial release

[0.1.0]: https://github.com/zombiezen/go-sqlite/releases/tag/v0.1.0
