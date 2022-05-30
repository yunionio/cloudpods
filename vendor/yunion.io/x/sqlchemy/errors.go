// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlchemy

import (
	"yunion.io/x/pkg/errors"
)

const (
	// ErrNoDataToUpdate is an Error constant: no data to update
	ErrNoDataToUpdate = errors.Error("No data to update")

	// ErrDuplicateEntry is an Error constant: duplicate entry
	ErrDuplicateEntry = errors.Error("duplicate entry")

	// ErrEmptyQuery is an Error constant: empty query
	ErrEmptyQuery = errors.Error("empty query")

	// ErrEmptyPrimaryKey is an Error constant: no primary key
	ErrEmptyPrimaryKey = errors.Error("empty primary keys")

	// ErrUnexpectRowCount is an Error constant: the number of rows impacted by modification unexpected
	ErrUnexpectRowCount = errors.Error("unexpected row count")

	// ErrNeedsPointer is an Error constant: input should be a pointer
	ErrNeedsPointer = errors.Error("input needs pointer input")

	// ErrNeedsArray is an Error constant: input should be an Array or Slice
	ErrNeedsArray = errors.Error("input needs slice or array")

	// ErrReadOnly is an Error constant: database is read-only
	ErrReadOnly = errors.Error("read only input")

	// ErrNotSupported is an Error constant: method not supported yet
	ErrNotSupported = errors.ErrNotSupported

	// ErrTableNotExists is an Error constant: table not exists
	ErrTableNotExists = errors.Error("TableNotExists")

	// ErrUnionFieldsNotMatch is an Error constant: fields of union queries not match
	ErrUnionFieldsNotMatch = errors.Error("cannot union, name of fields not match")

	// ErrUnionDatabasesNotMatch is an Error constant: backend database of union queries not match
	ErrUnionAcrossDatabases = errors.Error("cannot union across different databases")
)
