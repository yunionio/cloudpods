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
	framestream "github.com/farsightsec/golang-framestream"
	"google.golang.org/protobuf/proto"
)

// A Decoder reads and parses Dnstap messages from an io.Reader
type Decoder struct {
	buf []byte
	r   Reader
}

// NewDecoder creates a Decoder using the given dnstap Reader, accepting
// dnstap data frames up to maxSize in size.
func NewDecoder(r Reader, maxSize int) *Decoder {
	return &Decoder{
		buf: make([]byte, maxSize),
		r:   r,
	}
}

// Decode reads and parses a Dnstap message from the Decoder's Reader.
// Decode silently discards data frames larger than the Decoder's configured
// maxSize.
func (d *Decoder) Decode(m *Dnstap) error {
	for {
		n, err := d.r.ReadFrame(d.buf)

		switch err {
		case framestream.ErrDataFrameTooLarge:
			continue
		case nil:
			break
		default:
			return err
		}

		return proto.Unmarshal(d.buf[:n], m)
	}
}
