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
	"github.com/go-sql-driver/mysql"

	"yunion.io/x/pkg/errors"
)

const (
	ErrNoDataToUpdate   = errors.Error("No data to update")
	ErrDuplicateEntry   = errors.Error("duplicate entry")
	ErrEmptyQuery       = errors.Error("empty query")
	ErrEmptyPrimaryKey  = errors.Error("empty primary keys")
	ErrUnexpectRowCount = errors.Error("unexpected row count")

	ErrNeedsPointer = errors.Error("input needs pointer input")
	ErrNeedsArray   = errors.Error("input needs slice or array")
	ErrReadOnly     = errors.Error("read only input")

	ErrNotSupported = errors.ErrNotSupported

	ErrTableNotExists = errors.Error("TableNotExists")
)

const (
	mysqlErrorTableNotExist = 0x47a
)

func isMysqlError(err error, code uint16) bool {
	if myErr, ok := err.(*mysql.MySQLError); ok {
		return myErr.Number == code
	}
	return false
}
