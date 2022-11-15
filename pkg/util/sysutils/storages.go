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
	"strconv"

	"yunion.io/x/pkg/errors"
)

type SStorage struct {
	Device     string
	Rotational bool
	Size       int64
	ReadOnly   bool
	Removable  bool
	Partition  bool
	Scheduler  string
}

const (
	sysBlockDir = "/sys/class/block"
)

func DetectStorageType() (string, error) {
	ss, err := DetectStorages()
	if err != nil {
		return "", errors.Wrap(err, "DetectStorages")
	}
	var hdd, ssd int
	for _, s := range ss {
		if s.Partition {
			continue
		}
		if s.Removable {
			continue
		}
		if s.Rotational {
			hdd++
		} else {
			ssd++
		}
	}
	if hdd > 0 && ssd > 0 {
		return "hybrid", nil
	} else if hdd > 0 {
		return "hdd", nil
	} else if ssd > 0 {
		return "ssd", nil
	} else {
		return "", nil
	}
}

func DetectStorages() ([]SStorage, error) {
	files, err := ioutil.ReadDir(sysBlockDir)
	if err != nil {
		return nil, errors.Wrap(err, "ReadDir /sys/class/block")
	}
	ret := make([]SStorage, 0)
	for _, f := range files {
		s := detectStorage(filepath.Join(sysBlockDir, f.Name()), f.Name())
		ret = append(ret, s)
	}
	return ret, nil
}

func detectStorage(path string, name string) SStorage {
	s := SStorage{
		Device: name,
	}
	sizeBytes, _ := ioutil.ReadFile(filepath.Join(path, "size"))
	s.Size, _ = strconv.ParseInt(string(sizeBytes), 10, 64)
	removableBytes, _ := ioutil.ReadFile(filepath.Join(path, "removable"))
	if len(removableBytes) > 0 && removableBytes[0] == '0' {
		s.Removable = false
	} else {
		s.Removable = true
	}
	roBytes, _ := ioutil.ReadFile(filepath.Join(path, "ro"))
	if len(roBytes) > 0 && roBytes[0] == '0' {
		s.ReadOnly = false
	} else {
		s.ReadOnly = true
	}
	partitionBytes, _ := ioutil.ReadFile(filepath.Join(path, "partition"))
	if len(partitionBytes) > 0 && partitionBytes[0] == '1' {
		s.Partition = true
	} else {
		s.Partition = false
	}
	rotationalBytes, _ := ioutil.ReadFile(filepath.Join(path, "queue/rotational"))
	if len(rotationalBytes) > 0 && rotationalBytes[0] == '0' {
		s.Rotational = false
	} else {
		s.Rotational = true
	}
	schedulerBytes, _ := ioutil.ReadFile(filepath.Join(path, "queue/scheduler"))
	s.Scheduler = string(schedulerBytes)
	return s
}
