/*
 * Copyright (c) 2013-2019 by Farsight Security, Inc.
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
	"os"
	"time"
)

// MaxPayloadSize sets the upper limit on input Dnstap payload sizes. If an Input
// receives a Dnstap payload over this size limit, ReadInto will log an error and
// return.
//
// EDNS0 and DNS over TCP use 2 octets for DNS message size, imposing a maximum
// size of 65535 octets for the DNS message, which is the bulk of the data carried
// in a Dnstap message. Protobuf encoding overhead and metadata with some size
// guidance (e.g., identity and version being DNS strings, which have a maximum
// length of 255) add up to less than 1KB. The default 96KiB size of the buffer
// allows a bit over 30KB space for "extra" metadata.
//
var MaxPayloadSize uint32 = 96 * 1024

// A FrameStreamInput reads dnstap data from an io.ReadWriter.
type FrameStreamInput struct {
	wait   chan bool
	reader Reader
	log    Logger
}

// NewFrameStreamInput creates a FrameStreamInput reading data from the given
// io.ReadWriter. If bi is true, the input will use the bidirectional
// framestream protocol suitable for TCP and unix domain socket connections.
func NewFrameStreamInput(r io.ReadWriter, bi bool) (input *FrameStreamInput, err error) {
	return NewFrameStreamInputTimeout(r, bi, 0)
}

// NewFrameStreamInputTimeout creates a FramestreamInput reading data from the
// given io.ReadWriter with a timeout applied to reading and (for bidirectional
// inputs) writing control messages.
func NewFrameStreamInputTimeout(r io.ReadWriter, bi bool, timeout time.Duration) (input *FrameStreamInput, err error) {
	reader, err := NewReader(r, &ReaderOptions{
		Bidirectional: bi,
		Timeout:       timeout,
	})

	if err != nil {
		return nil, err
	}

	return &FrameStreamInput{
		wait:   make(chan bool),
		reader: reader,
		log:    nullLogger{},
	}, nil
}

// NewFrameStreamInputFromFilename creates a FrameStreamInput reading from
// the named file.
func NewFrameStreamInputFromFilename(fname string) (input *FrameStreamInput, err error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	return NewFrameStreamInput(file, false)
}

// SetLogger configures a logger for FrameStreamInput read error reporting.
func (input *FrameStreamInput) SetLogger(logger Logger) {
	input.log = logger
}

// ReadInto reads data from the FrameStreamInput into the output channel.
//
// ReadInto satisfies the dnstap Input interface.
func (input *FrameStreamInput) ReadInto(output chan []byte) {
	buf := make([]byte, MaxPayloadSize)
	for {
		n, err := input.reader.ReadFrame(buf)
		if err == nil {
			newbuf := make([]byte, n)
			copy(newbuf, buf)
			output <- newbuf
			continue
		}

		if err != io.EOF {
			input.log.Printf("FrameStreamInput: Read error: %v", err)
		}

		break
	}
	close(input.wait)
}

// Wait reeturns when ReadInto has finished.
//
// Wait satisfies the dnstap Input interface.
func (input *FrameStreamInput) Wait() {
	<-input.wait
}
