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
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"
)

func GetProcCmdline(pid uint) ([]string, error) {
	pPath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "cmdline")
	content, err := ioutil.ReadFile(pPath)
	if err != nil {
		return nil, errors.Wrapf(err, "ReadFile %q", pPath)
	}
	return parseProcCmdline(content), nil
}

func parseProcCmdline(content []byte) []string {
	return strings.Split(string(bytes.TrimRight(content, string("\x00"))), string(byte(0)))
}
