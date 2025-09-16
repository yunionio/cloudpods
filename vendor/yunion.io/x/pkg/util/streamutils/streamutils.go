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

package streamutils

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"

	"github.com/ulikunitz/xz"

	"yunion.io/x/pkg/errors"
)

type SStreamProperty struct {
	CheckSum string
	Size     int64
}

type sXZReadAheadReader struct {
	offset   int64
	header   []byte
	hdrEof   bool
	upstream io.Reader
}

func newXZReadAheadReader(stream io.Reader) (*sXZReadAheadReader, error) {
	xzHdr := make([]byte, xz.HeaderLen)
	n, err := stream.Read(xzHdr)
	hdrEof := false
	if err != nil {
		if errors.Cause(err) == io.EOF {
			// delay the EOF
			hdrEof = true
			xzHdr = xzHdr[:n]
		} else {
			return nil, errors.Wrap(err, "Read XZ header")
		}
	} else if n != len(xzHdr) {
		hdrEof = true
		xzHdr = xzHdr[:n]
	}
	return &sXZReadAheadReader{
		offset:   0,
		header:   xzHdr,
		hdrEof:   hdrEof,
		upstream: stream,
	}, nil
}

func (s *sXZReadAheadReader) IsXz() bool {
	return xz.ValidHeader(s.header)
}

func (s *sXZReadAheadReader) Read(buf []byte) (int, error) {
	bufOffset := 0
	if s.offset < int64(len(s.header)) {
		// read from header
		rdSize := len(s.header) - int(s.offset)
		if rdSize > len(buf) {
			rdSize = len(buf)
		}
		n := copy(buf, s.header[s.offset:s.offset+int64(rdSize)])
		s.offset += int64(n)
		bufOffset = n
	}
	// read buffer is full
	if bufOffset >= len(buf) {
		return bufOffset, nil
	}
	if s.offset >= int64(len(s.header)) && s.hdrEof {
		return bufOffset, io.EOF
	}

	n, err := s.upstream.Read(buf[bufOffset:])
	s.offset += int64(n)
	return n + bufOffset, err
}

func StreamPipe(upstream io.Reader, writer io.Writer, CalChecksum bool, callback func(savedTotal int64)) (*SStreamProperty, error) {
	return StreamPipe2(upstream, writer, CalChecksum, func(savedTotal int64, savedOnce int64) {
		if callback != nil {
			callback(savedTotal)
		}
	})
}

func StreamPipe2(upstream io.Reader, writer io.Writer, CalChecksum bool, callback func(savedTotal int64, savedOnce int64)) (*SStreamProperty, error) {
	sp := SStreamProperty{}

	var md5sum hash.Hash
	if CalChecksum {
		md5sum = md5.New()
	}

	aheadReader, err := newXZReadAheadReader(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAheadReader")
	}

	var reader io.Reader

	if aheadReader.IsXz() {
		xzReader, err := xz.NewReader(aheadReader)
		if err != nil {
			return nil, errors.Wrap(err, "xz.NewReader")
		}
		reader = xzReader
	} else {
		reader = aheadReader
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			sp.Size += int64(n)
			if callback != nil {
				callback(sp.Size, int64(n))
			}
			if CalChecksum {
				md5sum.Write(buf[:n])
			}
			offset := 0
			for offset < n {
				m, err := writer.Write(buf[offset:n])
				if err != nil {
					return nil, err
				}
				offset += m
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	if CalChecksum {
		sp.CheckSum = fmt.Sprintf("%x", md5sum.Sum(nil))
	}
	return &sp, nil
}
