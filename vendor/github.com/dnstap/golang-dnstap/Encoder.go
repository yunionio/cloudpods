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
	"google.golang.org/protobuf/proto"
)

// An Encoder serializes and writes Dnstap messages to an underlying
// dnstap Writer
type Encoder struct {
	w Writer
}

// NewEncoder creates an Encoder using the given dnstap Writer
func NewEncoder(w Writer) *Encoder {
	return &Encoder{w}
}

// Encode serializes and writes the Dnstap message m to the encoder's
// Writer.
func (e *Encoder) Encode(m *Dnstap) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	_, err = e.w.WriteFrame(b)
	return err
}
