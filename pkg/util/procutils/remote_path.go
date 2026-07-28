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

package procutils

import (
	"strings"

	"yunion.io/x/pkg/errors"
)

// RemotePathExists checks whether path exists on the host via executor.
func RemotePathExists(path string) bool {
	_, err := RemoteStat(path)
	return err == nil
}

// RemoteReadlink resolves a symlink on the host via executor.
func RemoteReadlink(path string) (string, error) {
	out, err := NewRemoteCommandAsFarAsPossible("readlink", "-f", path).Output()
	if err != nil {
		return "", errors.Wrapf(err, "readlink -f %s: %s", path, out)
	}
	return strings.TrimSpace(string(out)), nil
}
