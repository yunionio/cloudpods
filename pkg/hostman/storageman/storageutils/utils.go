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

//go:build !windows
// +build !windows

package storageutils

import (
	"syscall"
)

func GetTotalSizeMb(path string) (int, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return -1, err
	}
	return int(stat.Blocks * uint64(stat.Bsize) / 1024 / 1024), nil
}

func GetFreeSizeMb(path string) (int, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return -1, err
	}
	return int(stat.Bavail * uint64(stat.Bsize) / 1024 / 1024), nil
}

func GetUsedSizeMb(path string) (int, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return -1, err
	}
	return int((stat.Blocks - stat.Bfree) * uint64(stat.Bsize) / 1024 / 1024), nil
}
