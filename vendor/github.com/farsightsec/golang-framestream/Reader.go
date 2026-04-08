/*
 * Copyright (c) 2014 by Farsight Security, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package framestream

import (
	"bufio"
	"encoding/binary"
	"io"
	"io/ioutil"
	"time"
)

type ReaderOptions struct {
	// The ContentTypes accepted by the Reader. May be left unset for no
	// content negotiation. If the corresponding Writer offers a disjoint
	// set of ContentTypes, NewReader() will return ErrContentTypeMismatch.
	ContentTypes [][]byte
	// If Bidirectional is true, the underlying io.Reader must be an
	// io.ReadWriter, and the Reader will engage in a bidirectional
	// handshake with its peer to establish content type and communicate
	// shutdown.
	Bidirectional bool
	// Timeout gives the timeout for reading the initial handshake messages
	// from the peer and writing response messages if Bidirectional. It is
	// only effective for underlying Readers satisfying net.Conn.
	Timeout time.Duration
}

// Reader reads data frames from an underlying io.Reader using the Frame
// Streams framing protocol.
type Reader struct {
	contentType   []byte
	bidirectional bool
	r             *bufio.Reader
	w             *bufio.Writer
	stopped       bool
}

// NewReader creates a Frame Streams Reader reading from the given io.Reader
// with the given ReaderOptions.
func NewReader(r io.Reader, opt *ReaderOptions) (*Reader, error) {
	if opt == nil {
		opt = &ReaderOptions{}
	}
	tr := timeoutReader(r, opt)
	reader := &Reader{
		bidirectional: opt.Bidirectional,
		r:             bufio.NewReader(tr),
		w:             nil,
	}

	if len(opt.ContentTypes) > 0 {
		reader.contentType = opt.ContentTypes[0]
	}

	var cf ControlFrame
	if opt.Bidirectional {
		w, ok := tr.(io.Writer)
		if !ok {
			return nil, ErrType
		}
		reader.w = bufio.NewWriter(w)

		// Read the ready control frame.
		err := cf.DecodeTypeEscape(reader.r, CONTROL_READY)
		if err != nil {
			return nil, err
		}

		// Check content type.
		if t, ok := cf.ChooseContentType(opt.ContentTypes); ok {
			reader.contentType = t
		} else {
			return nil, ErrContentTypeMismatch
		}

		// Send the accept control frame.
		accept := ControlAccept
		accept.SetContentType(reader.contentType)
		err = accept.EncodeFlush(reader.w)
		if err != nil {
			return nil, err
		}
	}

	// Read the start control frame.
	err := cf.DecodeTypeEscape(reader.r, CONTROL_START)
	if err != nil {
		return nil, err
	}

	// Disable the read timeout to prevent killing idle connections.
	disableReadTimeout(tr)

	// Check content type.
	if !cf.MatchContentType(reader.contentType) {
		return nil, ErrContentTypeMismatch
	}

	return reader, nil
}

// ReadFrame reads a data frame into the supplied buffer, returning its length.
// If the frame is longer than the supplied buffer, Read returns
// ErrDataFrameTooLarge and discards the frame. Subsequent calls to Read()
// after this error may succeed.
func (r *Reader) ReadFrame(b []byte) (length int, err error) {
	if r.stopped {
		return 0, EOF
	}

	for length == 0 {
		length, err = r.readFrame(b)
		if err != nil {
			return
		}
	}

	return
}

// ContentType returns the content type negotiated with the Writer.
func (r *Reader) ContentType() []byte {
	return r.contentType
}

func (r *Reader) readFrame(b []byte) (int, error) {
	// Read the frame length.
	var frameLen uint32
	err := binary.Read(r.r, binary.BigEndian, &frameLen)
	if err != nil {
		return 0, err
	}

	if frameLen > uint32(len(b)) {
		io.CopyN(ioutil.Discard, r.r, int64(frameLen))
		return 0, ErrDataFrameTooLarge
	}

	if frameLen == 0 {
		// This is a control frame.
		var cf ControlFrame
		err = cf.Decode(r.r)
		if err != nil {
			return 0, err
		}
		if cf.ControlType == CONTROL_STOP {
			r.stopped = true
			if r.bidirectional {
				ff := &ControlFrame{ControlType: CONTROL_FINISH}
				err = ff.EncodeFlush(r.w)
				if err != nil {
					return 0, err
				}
			}
			return 0, EOF
		}
		return 0, err
	}

	return io.ReadFull(r.r, b[0:frameLen])
}
