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
	"io"
	"io/ioutil"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SReadSeeker struct {
	reader io.Reader

	offset int64

	readerOffset int64
	readerSize   int64

	tmpFile *os.File
}

func NewReadSeeker(reader io.Reader, size int64) (*SReadSeeker, error) {
	tmpfile, err := ioutil.TempFile("", "fakeseeker")
	if err != nil {
		return nil, errors.Wrap(err, "TempFile")
	}
	return &SReadSeeker{
		reader:       reader,
		readerOffset: 0,
		readerSize:   size,

		tmpFile: tmpfile,
	}, nil
}

func (s *SReadSeeker) Read(p []byte) (int, error) {
	if s.offset == s.readerOffset && s.offset < s.readerSize {
		n, err := s.reader.Read(p)
		if n > 0 {
			wn, werr := s.tmpFile.Write(p[:n])
			if werr != nil {
				return n, werr
			}
			if wn < n {
				return n, errors.Error("sFakeSeeker write less bytes")
			}
			s.offset += int64(n)
			s.readerOffset += int64(n)
		}
		return n, err
	} else {
		n, err := s.tmpFile.ReadAt(p, s.offset)
		if n > 0 {
			s.offset += int64(n)
		}
		return n, err
	}
}

func (s *SReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		// offset = offset
	case io.SeekCurrent:
		offset = s.offset + offset
	case io.SeekEnd:
		offset = s.readerSize + offset
	}
	if offset < 0 || offset > s.readerSize {
		log.Debugf("offset out of range: %d", offset)
		return -1, io.ErrUnexpectedEOF
	}
	s.offset = offset
	return offset, nil
}

func (s *SReadSeeker) Close() error {
	defer os.Remove(s.tmpFile.Name())
	return s.tmpFile.Close()
}
