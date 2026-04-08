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

// A Writer writes dnstap frames to its destination.
type Writer interface {
	WriteFrame([]byte) (int, error)
	Close() error
}

// WriterOptions specifies configuration for the Writer
type WriterOptions struct {
	// If Bidirectional is true, the underlying io.Writer must also
	// satisfy io.Reader, and the dnstap Writer will use the bidirectional
	// Frame Streams protocol.
	Bidirectional bool
	// Timeout sets the write timeout for data and control messages and the
	// read timeout for handshake responses on the underlying Writer. Timeout
	// is only effective if the underlying Writer is a net.Conn.
	Timeout time.Duration
}

// NewWriter creates a Writer using the given io.Writer and options.
func NewWriter(w io.Writer, opt *WriterOptions) (Writer, error) {
	if opt == nil {
		opt = &WriterOptions{}
	}
	return framestream.NewWriter(w,
		&framestream.WriterOptions{
			ContentTypes:  [][]byte{FSContentType},
			Timeout:       opt.Timeout,
			Bidirectional: opt.Bidirectional,
		})
}
