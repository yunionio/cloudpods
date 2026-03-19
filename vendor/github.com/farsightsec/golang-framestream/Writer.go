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
	"time"
)

type WriterOptions struct {
	// The ContentTypes available to be written to the Writer. May be
	// left unset for no content negotiation. If the Reader requests a
	// disjoint set of content types, NewEncoder() will return
	// ErrContentTypeMismatch.
	ContentTypes [][]byte
	// If Bidirectional is true, the underlying io.Writer must be an
	// io.ReadWriter, and the Writer will engage in a bidirectional
	// handshake with its peer to establish content type and communicate
	// shutdown.
	Bidirectional bool
	// Timeout gives the timeout for writing both control and data frames,
	// and for reading responses to control frames sent. It is only
	//  effective for underlying Writers satisfying net.Conn.
	Timeout time.Duration
}

// A Writer writes data frames to a Frame Streams file or connection.
type Writer struct {
	contentType []byte
	w           *bufio.Writer
	r           *bufio.Reader
	opt         WriterOptions
	buf         []byte
}

// NewWriter returns a Frame Streams Writer using the given io.Writer and options.
func NewWriter(w io.Writer, opt *WriterOptions) (writer *Writer, err error) {
	if opt == nil {
		opt = &WriterOptions{}
	}
	w = timeoutWriter(w, opt)
	writer = &Writer{
		w:   bufio.NewWriter(w),
		opt: *opt,
	}

	if len(opt.ContentTypes) > 0 {
		writer.contentType = opt.ContentTypes[0]
	}

	if opt.Bidirectional {
		r, ok := w.(io.Reader)
		if !ok {
			return nil, ErrType
		}
		writer.r = bufio.NewReader(r)
		ready := ControlReady
		ready.SetContentTypes(opt.ContentTypes)
		if err = ready.EncodeFlush(writer.w); err != nil {
			return
		}

		var accept ControlFrame
		if err = accept.DecodeTypeEscape(writer.r, CONTROL_ACCEPT); err != nil {
			return
		}

		if t, ok := accept.ChooseContentType(opt.ContentTypes); ok {
			writer.contentType = t
		} else {
			return nil, ErrContentTypeMismatch
		}
	}

	// Write the start control frame.
	start := ControlStart
	start.SetContentType(writer.contentType)
	err = start.EncodeFlush(writer.w)
	if err != nil {
		return
	}

	return
}

// ContentType returns the content type negotiated with Reader.
func (w *Writer) ContentType() []byte {
	return w.contentType
}

// Close shuts down the Frame Streams stream by writing a CONTROL_STOP message.
// If the Writer is Bidirectional, Close will wait for an acknowledgement
// (CONTROL_FINISH) from its peer.
func (w *Writer) Close() (err error) {
	err = ControlStop.EncodeFlush(w.w)
	if err != nil || !w.opt.Bidirectional {
		return
	}

	var finish ControlFrame
	return finish.DecodeTypeEscape(w.r, CONTROL_FINISH)
}

// WriteFrame writes the given frame to the underlying io.Writer with Frame Streams
// framing.
func (w *Writer) WriteFrame(frame []byte) (n int, err error) {
	err = binary.Write(w.w, binary.BigEndian, uint32(len(frame)))
	if err != nil {
		return
	}
	return w.w.Write(frame)
}

// Flush ensures that any buffered data frames are written to the underlying
// io.Writer.
func (w *Writer) Flush() error {
	return w.w.Flush()
}
