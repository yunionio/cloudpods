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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/utils"

	regutils "yunion.io/x/onecloud/pkg/util/regutils2"
)

type Partition struct {
	Index    int
	Bootable bool
	Start    int64
	End      int64
	Count    int64
	DiskType string
	Fs       string
	DevName  string
}

func NewPartition(idx int, bootable bool, start int64, end int64, count int64, diskType string, fs string, devName string) Partition {
	return Partition{
		Index:    idx,
		Bootable: bootable,
		Start:    start,
		End:      end,
		Count:    count,
		DiskType: diskType,
		Fs:       fs,
		DevName:  devName,
	}
}

// ParseDiskPartitions parse command `parted -s /dev/sda -- unit s print` result
func ParseDiskPartitions(dev string, lines []string) ([]Partition, string) {
	parts := make([]Partition, 0)
	labelPattern := `Partition Table:\s+(?P<label>\w+)`
	pattern := `(?P<idx>\d+)\s+(?P<start>\d+)s\s+(?P<end>\d+)s\s+(?P<count>\d+)s`
	var label string
	for _, l := range lines {
		if label == "" {
			m := regutils.SubGroupMatch(labelPattern, l)
			if len(m) != 0 {
				label = m["label"]
			}
		}
		m := regutils.SubGroupMatch(pattern, l)
		if len(m) != 0 {
			idx := m["idx"]
			devName := dev
			if strings.Contains("0123456789", string(dev[len(dev)-1])) {
				devName = fmt.Sprintf("%sp", devName)
			}
			devName = fmt.Sprintf("%s%s", devName, idx)
			start := m["start"]
			end := m["end"]
			count := m["count"]
			data := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(l), -1)
			diskType := ""
			fs := ""
			flag := ""
			offset := 0
			if len(data) > 4 {
				if label == "msdos" {
					diskType = data[4]
					if len(data) > 5 && isPartedFsString(data[5]) {
						fs = data[5]
						offset += 1
					}
					if len(data) > 5+offset {
						flag = data[5+offset]
					}
				} else if label == "gpt" {
					if isPartedFsString(data[4]) {
						fs = data[4]
						offset += 1
					}
					if len(data) > 4+offset {
						diskType = data[4+offset]
					}
					if len(data) > 4+offset+1 {
						flag = data[4+offset+1]
					}
				}
			}
			bootable := false
			if flag != "" && strings.Contains(flag, "boot") {
				bootable = true
			}
			index, _ := strconv.Atoi(idx)
			startI, _ := strconv.Atoi(start)
			endI, _ := strconv.Atoi(end)
			countI, _ := strconv.Atoi(count)
			parts = append(parts, NewPartition(index, bootable, int64(startI), int64(endI), int64(countI), diskType, fs, devName))
		}
	}
	return parts, label
}

func isPartedFsString(fs string) bool {
	return utils.IsInStringArray(strings.ToLower(fs),
		[]string{
			"ext2", "ext3", "ext4", "xfs",
			"fat16", "fat32",
			"hfs", "hfs+", "hfsx",
			"linux-swap", "linux-swap(v1)",
			"ntfs", "reiserfs", "ufs", "btrfs",
		})
}
