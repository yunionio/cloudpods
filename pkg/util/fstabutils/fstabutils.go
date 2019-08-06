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

package fstabutils

import (
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/log"
)

type SFSRecord struct {
	Dev   string
	Mount string
	Fs    string
	Opt   string
	Dump  string
	Pas   string
}

func FSRecord(line string) *SFSRecord {
	data := regexp.MustCompile(`\s+`).Split(line, -1)
	if len(data) > 5 {
		return &SFSRecord{data[0], data[1], data[2], data[3], data[4], data[5]}
	}
	log.Errorf("Invalid fstab record %s", line)
	return nil
}

func (fsr *SFSRecord) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
		fsr.Dev, fsr.Mount, fsr.Fs, fsr.Opt, fsr.Dump, fsr.Pas)
}

var VDISK_PREFIX = "/dev/vd"

type FsTab []*SFSRecord

func FSTabFile(content string) *FsTab {
	if len(content) > 0 {
		var res = make(FsTab, 0)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 && line[0] != '#' {
				if fsr := FSRecord(line); fsr != nil {
					res = append(res, fsr)
				}
			}
		}
		return &res
	}
	return nil
}

func (ft *FsTab) IsExists(dev string) bool {
	for _, f := range *ft {
		if f.Dev == dev {
			return true
		}
	}
	return false
}

func (ft *FsTab) AddFsrec(line string) {
	rec := FSRecord(line)
	if rec != nil {
		*ft = append(*ft, rec)
	}
}

func (ft *FsTab) RemoveDevices(devCnt int) *FsTab {
	var newList = make(FsTab, 0)
	for _, f := range *ft {
		if strings.HasPrefix(f.Dev, VDISK_PREFIX) {
			devChar := f.Dev[len(VDISK_PREFIX)]
			devIdx := devChar - 'a'
			if devIdx >= 0 && int(devIdx) < devCnt {
				newList = append(newList, f)
			}
		} else {
			newList = append(newList, f)
		}
	}
	return &newList
}

func (ft *FsTab) ToConf() string {
	var res string
	for _, f := range *ft {
		res += fmt.Sprintf("%s\n", f)
	}
	return res
}

/* TODO: test
if __name__ == '__main__':
    with open('/etc/fstab') as f:
        cont = f.read(4096)
        print cont
        fstab = FSTabFile(cont)
        print fstab.is_exists('/dev/sdc2')
        fstab.add_fsrec('/dev/sdd1 /data ext4 defaults 0 0')
        fstab.add_fsrec('/dev/sdd2')
        print fstab.to_conf()
*/
