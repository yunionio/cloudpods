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

package httperrors

import (
	"io/fs"
	"os"

	"yunion.io/x/pkg/errors"
)

/*
ErrInvalid = fs.ErrInvalid // "invalid argument"

ErrPermission = fs.ErrPermission // "permission denied"
ErrExist      = fs.ErrExist      // "file already exists"
ErrNotExist   = fs.ErrNotExist   // "file does not exist"
ErrClosed     = fs.ErrClosed     // "file already closed"

ErrNoDeadline       = errNoDeadline()       // "file type does not support deadline"
ErrDeadlineExceeded = errDeadlineExceeded() // "i/o timeout"
*/
func FsErrorNormalize(err error) error {
	switch errors.Cause(err) {
	case fs.ErrInvalid:
		return errors.Wrap(ErrInputParameter, "invalid argument")
	case fs.ErrPermission:
		return errors.Wrap(ErrForbidden, "permission denied")
	case fs.ErrExist:
		return errors.Wrap(ErrConflict, "file already exists")
	case fs.ErrNotExist:
		return errors.Wrap(ErrNotFound, "file does not exist")
	case fs.ErrClosed:
		return errors.Wrap(ErrInvalidStatus, "file already closed")
	case os.ErrNoDeadline:
		return errors.Wrap(ErrNotSupported, "file type does not support deadline")
	case os.ErrDeadlineExceeded:
		return errors.Wrap(ErrTimeout, "i/o timeout")
	}
	return err
}
