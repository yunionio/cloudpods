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

package fileutils

import (
	"os"

	"yunion.io/x/pkg/errors"
)

type SparseFileWriter struct {
	*os.File

	needNul bool
}

func NewSparseFileWriter(f *os.File) *SparseFileWriter {
	w := &SparseFileWriter{
		File: f,
	}
	return w
}

func (w *SparseFileWriter) Write(d []byte) (int, error) {
	for _, b := range d {
		if b != 0 {
			w.needNul = false
			return w.File.Write(d)
		}
	}
	_, err := w.File.Seek(int64(len(d)), os.SEEK_CUR)
	if err != nil {
		return 0, err
	}
	w.needNul = true
	return len(d), nil
}

func (w *SparseFileWriter) PreClose() error {
	if w.needNul {
		if _, err := w.File.Seek(-1, os.SEEK_CUR); err != nil {
			return errors.Wrap(err, "seek back 1 byte")
		}
		if _, err := w.File.Write([]byte{0}); err != nil {
			return errors.Wrap(err, "write 1 nul byte")
		}
		w.needNul = false
	}
	return nil
}

func (w *SparseFileWriter) Close() (err error) {
	defer func() {
		err = w.File.Close()
	}()
	if err := w.PreClose(); err != nil {
		panic("BUG: call PreClose() and handle error: " + err.Error())
	}
	return
}
