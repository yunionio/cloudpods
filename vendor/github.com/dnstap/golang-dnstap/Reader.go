/*
 * Copyright (c) 2019 by Farsight Security, Inc.
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

package dnstap

import (
	"io"
	"time"

	framestream "github.com/farsightsec/golang-framestream"
)

// A Reader is a source of dnstap frames.
type Reader interface {
	ReadFrame([]byte) (int, error)
}

// ReaderOptions specifies configuration for the Reader.
type ReaderOptions struct {
	// If Bidirectional is true, the underlying io.Reader must also
	// satisfy io.Writer, and the dnstap Reader will use the bidirectional
	// Frame Streams protocol.
	Bidirectional bool
	// Timeout sets the timeout for reading the initial handshake and
	// writing response control messages to the underlying Reader. Timeout
	// is only effective if the underlying Reader is a net.Conn.
	Timeout time.Duration
}

// NewReader creates a Reader using the given io.Reader and options.
func NewReader(r io.Reader, opt *ReaderOptions) (Reader, error) {
	if opt == nil {
		opt = &ReaderOptions{}
	}
	return framestream.NewReader(r,
		&framestream.ReaderOptions{
			ContentTypes:  [][]byte{FSContentType},
			Timeout:       opt.Timeout,
			Bidirectional: opt.Bidirectional,
		})
}
