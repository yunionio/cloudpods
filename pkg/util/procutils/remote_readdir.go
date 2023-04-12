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
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
)

var (
	splitReg = regexp.MustCompile(`\s+`)
)

func split(line string) []string {
	return splitReg.Split(line, -1)
}

func parseLsLine(line string) (os.FileInfo, error) {
	// drwxr-xr-x.  7 1000   20       4096 2022-02-16 08:10:21.660000000 +0800 .vim
	// -rw-------.  1    0    0      23658 2022-04-21 17:31:04.320995359 +0800 .viminfo
	// lrwxr-xr-x    1 501  80      31 2022-11-02 21:01:10.000000000 +0800 zstdmt -> ../Cellar/zstd/1.5.2/bin/zstdmt
	line = strings.TrimSpace(line)
	if len(line) < 10 {
		return nil, errors.Error("invalid ls line: too short")
	}
	parts := split(line)
	if len(parts) < 9 {
		return nil, errors.Error(fmt.Sprintf("invalid ls line: parts %d", len(parts)))
	}
	ftype := "file"
	switch parts[0][0] {
	case 'd':
		ftype = "directory"
	case 'l':
		ftype = "link"
	}
	size, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "Parse size")
	}
	tmstr := fmt.Sprintf("%sT%s%s:00", parts[5], parts[6], parts[7][:3])
	atime, err := timeutils.ParseTimeStr(tmstr)
	if err != nil {
		return nil, errors.Wrap(err, "parse time")
	}
	tmstr = strings.Join(parts[5:8], " ")
	nameIndex := strings.Index(line, tmstr) + len(tmstr) + 1
	name := strings.TrimSpace(line[nameIndex:])
	if ftype == "link" {
		arrowPos := strings.Index(name, "->")
		if arrowPos > 0 {
			name = strings.TrimSpace(name[:arrowPos])
		}
	}
	fs := &sFileStat{
		FileSize:  size,
		FileType:  ftype,
		FileName:  name,
		LastModAt: atime,
	}
	return fs, nil
}

func RemoteReadDir(dirname string) ([]os.FileInfo, error) {
	args := []string{}
	switch runtime.GOOS {
	case "darwin":
		args = []string{"-la1n", "-D", `%Y-%m-%d %H:%M:%S.000000000 %z`, dirname}
	default:
		args = []string{"-la1n", "--full-time", dirname}
	}
	output, err := NewRemoteCommandAsFarAsPossible("ls", args...).Output()
	if err != nil {
		if strings.Contains(strings.ToLower(string(output)), "no such file or directory") {
			return nil, os.ErrNotExist
		}
		return nil, errors.Wrap(err, "NewRemoteCommandAsFarAsPossible")
	}
	files := make([]os.FileInfo, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		f, err := parseLsLine(line)
		if err != nil {
			//log.Errorf("parseLsLine %s fail %s", line, err)
		} else {
			files = append(files, f)
		}
	}
	return files, nil
}
