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

package fileutils2

import (
	"os/exec"
	"regexp"

	"yunion.io/x/log"
)

const (
	blkidTypePattern = `TYPE="(?P<type>\w+)"`
)

var (
	blkidTypeRegexp = regexp.MustCompile(blkidTypePattern)
)

func GetBlkidType(filepath string) string {
	cmd := exec.Command("blkid", filepath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("blkid fail %s %s", filepath, err)
		return ""
	}
	matches := blkidTypeRegexp.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
