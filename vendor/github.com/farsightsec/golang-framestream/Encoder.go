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
	"io"
	"time"
)

// EncoderOptions specifies configuration for an Encoder
type EncoderOptions struct {
	// The ContentType of the data sent by the Encoder. May be left unset
	// for no content negotiation. If the Reader requests a different
	// content type, NewEncoder() will return ErrContentTypeMismatch.
	ContentType []byte
	// If Bidirectional is true, the underlying io.Writer must be an
	// io.ReadWriter, and the Encoder will engage in a bidirectional
	// handshake with its peer to establish content type and communicate
	// shutdown.
	Bidirectional bool
	// Timeout gives the timeout for writing both control and data frames,
	// and for reading responses to control frames sent. It is only
	//  effective for underlying Writers satisfying net.Conn.
	Timeout time.Duration
}

// An Encoder sends data frames over a FrameStream Writer.
//
// Encoder is provided for compatibility, use Writer instead.
type Encoder struct {
	*Writer
}

// NewEncoder creates an Encoder writing to the given io.Writer with the given
// EncoderOptions.
func NewEncoder(w io.Writer, opt *EncoderOptions) (enc *Encoder, err error) {
	if opt == nil {
		opt = &EncoderOptions{}
	}
	wopt := &WriterOptions{
		Bidirectional: opt.Bidirectional,
		Timeout:       opt.Timeout,
	}
	if opt.ContentType != nil {
		wopt.ContentTypes = append(wopt.ContentTypes, opt.ContentType)
	}
	writer, err := NewWriter(w, wopt)
	if err != nil {
		return nil, err
	}
	return &Encoder{Writer: writer}, nil
}

func (e *Encoder) Write(frame []byte) (int, error) {
	return e.WriteFrame(frame)
}
