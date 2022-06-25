// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package binary

import (
	"encoding/binary"
	"io"
	"math"
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		input: r,
	}
}

type Decoder struct {
	input   io.Reader
	scratch [binary.MaxVarintLen64]byte
}

func (decoder *Decoder) Raw(b []byte) error {
	n, err := decoder.input.Read(b)
	if err != nil {
		return nil
	}
	if n != len(b) {
		return io.EOF
	}
	return nil
}

func (decoder *Decoder) Bool() (bool, error) {
	v, err := decoder.ReadByte()
	if err != nil {
		return false, err
	}
	return v == 1, nil
}

func (decoder *Decoder) Uvarint() (uint64, error) {
	return binary.ReadUvarint(decoder)
}

func (decoder *Decoder) Int8() (int8, error) {
	v, err := decoder.ReadByte()
	if err != nil {
		return 0, err
	}
	return int8(v), nil
}

func (decoder *Decoder) Int16() (int16, error) {
	v, err := decoder.UInt16()
	if err != nil {
		return 0, err
	}
	return int16(v), nil
}

func (decoder *Decoder) Int32() (int32, error) {
	v, err := decoder.UInt32()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func (decoder *Decoder) Int64() (int64, error) {
	v, err := decoder.UInt64()
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func (decoder *Decoder) UInt8() (uint8, error) {
	v, err := decoder.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint8(v), nil
}

func (decoder *Decoder) UInt16() (uint16, error) {
	if _, err := decoder.input.Read(decoder.scratch[:2]); err != nil {
		return 0, err
	}
	return uint16(decoder.scratch[0]) | uint16(decoder.scratch[1])<<8, nil
}

func (decoder *Decoder) UInt32() (uint32, error) {
	if _, err := decoder.input.Read(decoder.scratch[:4]); err != nil {
		return 0, err
	}
	return uint32(decoder.scratch[0]) |
		uint32(decoder.scratch[1])<<8 |
		uint32(decoder.scratch[2])<<16 |
		uint32(decoder.scratch[3])<<24, nil
}

func (decoder *Decoder) UInt64() (uint64, error) {
	if _, err := decoder.input.Read(decoder.scratch[:8]); err != nil {
		return 0, err
	}
	return uint64(decoder.scratch[0]) |
		uint64(decoder.scratch[1])<<8 |
		uint64(decoder.scratch[2])<<16 |
		uint64(decoder.scratch[3])<<24 |
		uint64(decoder.scratch[4])<<32 |
		uint64(decoder.scratch[5])<<40 |
		uint64(decoder.scratch[6])<<48 |
		uint64(decoder.scratch[7])<<56, nil
}

func (decoder *Decoder) Float32() (float32, error) {
	v, err := decoder.UInt32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

func (decoder *Decoder) Float64() (float64, error) {
	v, err := decoder.UInt64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

func (decoder *Decoder) Fixed(ln int) ([]byte, error) {
	if reader, ok := decoder.input.(interface{ Fixed(ln int) ([]byte, error) }); ok {
		return reader.Fixed(ln)
	}
	buf := make([]byte, ln)
	if _, err := decoder.input.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (decoder *Decoder) String() (string, error) {
	strlen, err := decoder.Uvarint()
	if err != nil {
		return "", err
	}
	str, err := decoder.Fixed(int(strlen))
	if err != nil {
		return "", err
	}
	return string(str), nil
}

func (decoder *Decoder) ReadByte() (byte, error) {
	if _, err := decoder.input.Read(decoder.scratch[:1]); err != nil {
		return 0x0, err
	}
	return decoder.scratch[0], nil
}
