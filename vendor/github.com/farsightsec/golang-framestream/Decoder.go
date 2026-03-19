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

// DecoderOptions specifies configuration for a framestream Decoder.
type DecoderOptions struct {
	// MaxPayloadSize is the largest frame size accepted by the Decoder.
	//
	// If the Frame Streams Writer sends a frame in excess of this size,
	// Decode() will return the error ErrDataFrameTooLarge. The Decoder
	// attempts to recover from this error, so calls to Decode() after
	// receiving this error may succeed.
	MaxPayloadSize uint32
	// The ContentType expected by the Decoder. May be left unset for no
	// content negotiation. If the Writer requests a different content type,
	// NewDecoder() will return ErrContentTypeMismatch.
	ContentType []byte
	// If Bidirectional is true, the underlying io.Reader must be an
	// io.ReadWriter, and the Decoder will engage in a bidirectional
	// handshake with its peer to establish content type and communicate
	// shutdown.
	Bidirectional bool
	// Timeout gives the timeout for reading the initial handshake messages
	// from the peer and writing response messages if Bidirectional. It is
	// only effective for underlying Readers satisfying net.Conn.
	Timeout time.Duration
}

// A Decoder decodes Frame Streams frames read from an underlying io.Reader.
//
// It is provided for compatibility. Use Reader instead.
type Decoder struct {
	buf []byte
	r   *Reader
}

// NewDecoder returns a Decoder using the given io.Reader and options.
func NewDecoder(r io.Reader, opt *DecoderOptions) (*Decoder, error) {
	if opt == nil {
		opt = &DecoderOptions{}
	}
	if opt.MaxPayloadSize == 0 {
		opt.MaxPayloadSize = DEFAULT_MAX_PAYLOAD_SIZE
	}
	ropt := &ReaderOptions{
		Bidirectional: opt.Bidirectional,
		Timeout:       opt.Timeout,
	}
	if opt.ContentType != nil {
		ropt.ContentTypes = append(ropt.ContentTypes, opt.ContentType)
	}
	dr, err := NewReader(r, ropt)
	if err != nil {
		return nil, err
	}
	dec := &Decoder{
		buf: make([]byte, opt.MaxPayloadSize),
		r:   dr,
	}
	return dec, nil
}

// Decode returns the data from a Frame Streams data frame. The slice returned
// is valid until the next call to Decode.
func (dec *Decoder) Decode() (frameData []byte, err error) {
	n, err := dec.r.ReadFrame(dec.buf)
	if err != nil {
		return nil, err
	}
	return dec.buf[:n], nil
}
