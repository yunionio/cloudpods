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

package zeroclean

import (
	"os"
	"path/filepath"

	"yunion.io/x/pkg/errors"
)

func ZeroFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrap(err, "os.OpenFile")
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "f.Stat")
	}

	zeroBuf := make([]byte, 4096)
	offset := int64(0)

	for offset < info.Size() {
		if offset+int64(len(zeroBuf)) > info.Size() {
			zeroBuf = zeroBuf[:info.Size()-offset]
		}
		n, err := f.WriteAt(zeroBuf, offset)
		if err != nil {
			return errors.Wrapf(err, "zero at %d", offset)
		}
		offset += int64(n)
	}

	return nil
}

func ZeroDir(dirname string) error {
	err := filepath.Walk(dirname, func(path string, d os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "WalkDIr %s", path)
		}
		if !d.IsDir() {
			err := ZeroFile(path)
			if err != nil {
				return errors.Wrapf(err, "Zerofiles %s", path)
			}
		}
		return nil
	})
	return errors.Wrap(err, "filepath.WalkDir")
}
