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

//go:build linux
// +build linux

package sparsefile

import (
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	SEEK_DATA = 3
	SEEK_HOLE = 4
)

func detectHoles(file *os.File) ([]sSparseHole, error) {
	holes := []sSparseHole{}

	offset := int64(0)
	for {
		start, err := unix.Seek(int(file.Fd()), offset, SEEK_HOLE)
		if err != nil {
			if e, ok := err.(syscall.Errno); ok && e == syscall.ENXIO {
				break
			}
			return nil, err
		}
		end, err := unix.Seek(int(file.Fd()), start, SEEK_DATA)
		if err != nil {
			if e, ok := err.(syscall.Errno); ok && e == syscall.ENXIO {
				end, _ = file.Seek(0, io.SeekEnd)
				if end > start {
					holes = append(holes, sSparseHole{Offset: start, Length: end - start})
				}
				break
			}
			return nil, err
		}
		offset = end
		holes = append(holes, sSparseHole{Offset: start, Length: end - start})
	}
	return holes, nil
}
