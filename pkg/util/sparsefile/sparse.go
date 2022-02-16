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

package sparsefile

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"yunion.io/x/pkg/errors"
)

type sSparseHole struct {
	Offset int64
	Length int64
}

type SparseFileReader struct {
	file *os.File

	holes []sSparseHole

	header     []byte
	size       int64
	headerSize int64

	realReadLen int64
}

func (self *SparseFileReader) Close() error {
	return self.file.Close()
}

func (self *SparseFileReader) HeaderSize() int64 {
	return self.headerSize
}

func (self *SparseFileReader) GetHoles() []sSparseHole {
	return self.holes
}

func (self *SparseFileReader) Size() int64 {
	holeSize := int64(0)
	for _, hole := range self.holes {
		holeSize += hole.Length
	}
	return self.size - holeSize + self.headerSize
}

func (self *SparseFileReader) Read(p []byte) (int, error) {
	if len(self.header) > 0 {
		reader := bytes.NewReader(self.header)
		n, err := reader.Read(p)
		if err != nil {
			return n, err
		}
		self.header = self.header[n:]
		return n, nil
	}
	for _, hole := range self.holes {
		if self.realReadLen > hole.Offset {
			continue
		}
		if self.realReadLen < hole.Offset {
			body := io.LimitReader(self.file, hole.Offset-self.realReadLen)
			n, err := body.Read(p)
			if err != nil {
				return n, err
			}
			self.realReadLen += int64(n)
			return n, nil
		} else if self.realReadLen == hole.Offset {
			n, err := self.file.Seek(hole.Length, io.SeekCurrent)
			if err != nil {
				return int(n), err
			}
			self.realReadLen = int64(n)
		}
	}
	return self.file.Read(p)
}

func (self *SparseFileReader) probeHoles() error {
	var err error
	self.holes, err = detectHoles(self.file)
	if err != nil {
		return err
	}
	if len(self.holes) > 0 {
		self.header, err = json.Marshal(self.holes)
		if err != nil {
			return errors.Wrapf(err, "json.Marshal")
		}
		self.headerSize = int64(len(self.header))
	}
	return nil
}

func NewSparseFileReader(file *os.File) (*SparseFileReader, error) {
	ret := &SparseFileReader{file: file, holes: []sSparseHole{}, realReadLen: 0}
	stat, err := ret.file.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "Stat")
	}
	ret.size = stat.Size()
	err = ret.probeHoles()
	if err != nil {
		return nil, err
	}
	_, err = ret.file.Seek(0, io.SeekStart)
	return ret, err
}

type SparseFileWrite struct {
	f          *os.File
	headerSize int64

	size int64

	header []byte
	holes  []sSparseHole

	bodyWriteLen int64

	readed int
}

func (self *SparseFileWrite) Close() error {
	return self.f.Close()
}

type zero struct{}

func (zero) Read(p []byte) (int, error) {
	for index := range p {
		p[index] = 0
	}
	return len(p), nil
}

func (self *SparseFileWrite) initHeader() error {
	err := json.Unmarshal(self.header, &self.holes)
	if err != nil {
		return errors.Wrapf(err, "unmarshal header")
	}
	for _, h := range self.holes {
		self.size += h.Length
	}
	return nil
}

func (self *SparseFileWrite) Write(p []byte) (int, error) {
	if len(self.header) < int(self.headerSize) {
		n := int(self.headerSize) - len(self.header)
		if len(p) >= n {
			self.header = append(self.header, p[:n]...)
			err := self.initHeader()
			if err != nil {
				return n, err
			}
			self.readed = n
		} else {
			self.header = append(self.header, p...)
			return len(p), nil
		}
	}

	if self.readed == len(p) {
		self.readed = 0
		return len(p), nil
	}

	for _, hole := range self.holes {
		if self.bodyWriteLen > hole.Offset {
			continue
		}
		if self.bodyWriteLen < hole.Offset {
			data := p[self.readed:]
			if len(p[self.readed:]) > int(hole.Offset-self.bodyWriteLen) {
				data = p[self.readed : hole.Offset-self.bodyWriteLen]
			}
			n, err := self.f.Write(data)
			if err != nil {
				return n, err
			}
			self.readed += n
			self.bodyWriteLen += int64(n)
			if len(p) == self.readed {
				self.readed = 0
				return len(p), nil
			}
		} else if self.bodyWriteLen == hole.Offset {
			n, err := self.f.Seek(hole.Length, io.SeekCurrent)
			if err != nil {
				return int(n), err
			}
			self.bodyWriteLen = int64(n)
		}
	}
	return self.f.Write(p)
}

func NewSparseFileWriter(f *os.File, headerSize int64, size int64) *SparseFileWrite {
	return &SparseFileWrite{
		f:          f,
		headerSize: headerSize,
		header:     []byte{},
		holes:      []sSparseHole{},
		size:       size,
	}
}
