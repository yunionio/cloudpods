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

package sysutils

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SHugepageInfo struct {
	SizeKb int
	Total  int
	Free   int
}

func (h SHugepageInfo) BytesMb() int64 {
	return int64(h.Total) * int64(h.SizeKb) / 1024
}

type THugepages []SHugepageInfo

func (a THugepages) Len() int           { return len(a) }
func (a THugepages) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a THugepages) Less(i, j int) bool { return a[i].SizeKb < a[j].SizeKb }

func (a THugepages) BytesMb() int64 {
	ret := int64(0)
	for _, h := range a {
		ret += h.BytesMb()
	}
	return ret
}

func (a THugepages) PageSizes() []int {
	ret := make([]int, 0)
	for _, h := range a {
		if h.Total > 0 {
			ret = append(ret, h.SizeKb)
		}
	}
	return ret
}

func fetchHugepageInfo(sizeKb int, dir string) (SHugepageInfo, error) {
	info := SHugepageInfo{
		SizeKb: sizeKb,
	}
	cont, err := ioutil.ReadFile(filepath.Join(dir, "nr_hugepages"))
	if err != nil {
		return info, errors.Wrap(err, "FileGetContents nr_hugepages")
	}
	total, _ := strconv.Atoi(strings.TrimSpace(string(cont)))
	cont, err = ioutil.ReadFile(filepath.Join(dir, "free_hugepages"))
	if err != nil {
		return info, errors.Wrap(err, "FileGetContents free_hugepages")
	}
	free, _ := strconv.Atoi(strings.TrimSpace(string(cont)))
	info.Total = int(total)
	info.Free = int(free)
	return info, nil
}

func GetHugepages() (THugepages, error) {
	const hugepageDir = "/sys/kernel/mm/hugepages"
	files, err := ioutil.ReadDir(hugepageDir)
	if err != nil {
		return nil, errors.Wrapf(err, "ReadDir %s", hugepageDir)
	}
	re := regexp.MustCompile(`hugepages-(\d+)kB`)
	infos := make(THugepages, 0)
	for _, dir := range files {
		if !dir.IsDir() {
			continue
		}
		ms := re.FindAllStringSubmatch(dir.Name(), -1)
		if len(ms) > 0 && len(ms[0]) > 1 {
			sizeKb, _ := strconv.Atoi(ms[0][1])
			if sizeKb > 0 {
				info, err := fetchHugepageInfo(sizeKb, filepath.Join(hugepageDir, dir.Name()))
				if err != nil {
					log.Errorf("fetchHugepageInfo %s fail %s", dir.Name(), err)
				} else {
					infos = append(infos, info)
				}
			}
		}
	}
	sort.Sort(infos)
	return infos, nil
}
