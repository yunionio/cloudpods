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

package fsdriver

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// SFileInfo implements os.FileInfo interface
type SFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
	stat  *syscall.Stat_t
}

func NewFileInfo(name string, size int64, mode os.FileMode, isDir bool, stat *syscall.Stat_t) *SFileInfo {
	return &SFileInfo{name, size, mode, isDir, stat}
}

func (info SFileInfo) Name() string {
	return info.name
}

func (info SFileInfo) Size() int64 {
	return info.size
}

func (info SFileInfo) Mode() os.FileMode {
	return info.mode
}

func (info SFileInfo) IsDir() bool {
	return info.isDir
}

func (info SFileInfo) ModTime() time.Time {
	// TODO: impl
	return time.Now()
}

func (info SFileInfo) Sys() interface{} {
	return info.stat
}

func ModeStr2Bin(mode string) (uint32, error) {
	table := []map[byte]uint32{
		{'-': syscall.S_IRUSR, 'd': syscall.S_IFDIR, 'l': syscall.S_IFLNK},
		{'r': syscall.S_IRUSR},
		{'w': syscall.S_IWUSR},
		{'x': syscall.S_IXUSR, 's': syscall.S_ISUID},
		{'r': syscall.S_IRGRP},
		{'w': syscall.S_IWGRP},
		{'x': syscall.S_IXGRP, 's': syscall.S_ISGID},
		{'r': syscall.S_IROTH},
		{'w': syscall.S_IWOTH},
		{'x': syscall.S_IXOTH},
	}
	if len(mode) != len(table) {
		return 0, fmt.Errorf("Invalid mod %q", mode)
	}
	var ret uint32 = 0
	for i := 0; i < len(table); i++ {
		ret |= table[i][mode[i]]
	}
	return ret, nil
}
