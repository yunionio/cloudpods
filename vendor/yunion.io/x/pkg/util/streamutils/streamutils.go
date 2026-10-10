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
	// io.ReadFull so that a stream handing its data over in more than one
	// piece is not mistaken for a stream that has already finished.
	n, err := io.ReadFull(stream, xzHdr)
	hdrEof := false
	if err != nil {
		cause := errors.Cause(err)
		if cause == io.EOF || cause == io.ErrUnexpectedEOF {
			// delay the EOF
			hdrEof = true
			xzHdr = xzHdr[:n]
		} else {
			return nil, errors.Wrap(err, "Read XZ header")
		}
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

// ErrSizeLimitExceeded is returned when the stream produces more bytes than
// the limit passed to StreamPipe or StreamPipe2 allows.
const ErrSizeLimitExceeded = errors.Error("stream size limit exceeded")

// StreamPipe streams upstream into writer. See StreamPipe2 for the optional
// size limit.
func StreamPipe(upstream io.Reader, writer io.Writer, CalChecksum bool, callback func(savedTotal int64), maxSize ...int64) (*SStreamProperty, error) {
	return StreamPipe2(upstream, writer, CalChecksum, func(savedTotal int64, savedOnce int64) {
		if callback != nil {
			callback(savedTotal)
		}
	}, maxSize...)
}

// StreamPipe2 streams upstream into writer, decompressing when the input is an
// xz stream.
//
// An optional maxSize stops the transfer once that many bytes have been
// produced, so that an input which expands to far more than it looks like it
// should cannot fill the writer. Omit it, or pass a value of zero or less, for
// no limit.
func StreamPipe2(upstream io.Reader, writer io.Writer, CalChecksum bool, callback func(savedTotal int64, savedOnce int64), maxSize ...int64) (*SStreamProperty, error) {
	sp := SStreamProperty{}

	var limit int64
	if len(maxSize) > 0 {
		limit = maxSize[0]
	}

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
			if limit > 0 && sp.Size > limit {
				// Stop before the excess reaches the writer.
				return nil, errors.Wrapf(ErrSizeLimitExceeded, "produced more than %d bytes", limit)
			}
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
