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
	"testing"

	"yunion.io/x/pkg/errors"
)

func TestFsErrorNormalize(t *testing.T) {
	cases := []struct {
		inErr error
		want  error
	}{
		{
			inErr: fs.ErrInvalid,
			want:  ErrInputParameter,
		},
		{
			inErr: fs.ErrPermission,
			want:  ErrForbidden,
		},
		{
			inErr: fs.ErrExist,
			want:  ErrConflict,
		},
		{
			inErr: fs.ErrNotExist,
			want:  ErrNotFound,
		},
		{
			inErr: fs.ErrClosed,
			want:  ErrInvalidStatus,
		},
		{
			inErr: os.ErrNoDeadline,
			want:  ErrNotSupported,
		},
		{
			inErr: os.ErrDeadlineExceeded,
			want:  ErrTimeout,
		},
	}
	for _, c := range cases {
		got := FsErrorNormalize(c.inErr)
		if errors.Cause(got) != c.want {
			t.Errorf("inErr %s want %s got %s", c.inErr, c.want, got)
		}
	}
}
