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

package column

import (
	"reflect"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/binary"
)

type SimpleAggregateFunction struct {
	base   Interface
	chType Type
}

func (col *SimpleAggregateFunction) parse(t Type) (_ Interface, err error) {
	col.chType = t
	base := strings.TrimSpace(strings.SplitN(t.params(), ",", 2)[1])
	if col.base, err = Type(base).Column(); err == nil {
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *SimpleAggregateFunction) Type() Type {
	return col.chType
}
func (col *SimpleAggregateFunction) ScanType() reflect.Type {
	return col.base.ScanType()
}
func (col *SimpleAggregateFunction) Rows() int {
	return col.base.Rows()
}
func (col *SimpleAggregateFunction) Row(i int, ptr bool) interface{} {
	return col.base.Row(i, ptr)
}
func (col *SimpleAggregateFunction) ScanRow(dest interface{}, rows int) error {
	return col.base.ScanRow(dest, rows)
}
func (col *SimpleAggregateFunction) Append(v interface{}) ([]uint8, error) {
	return col.base.Append(v)
}
func (col *SimpleAggregateFunction) AppendRow(v interface{}) error {
	return col.base.AppendRow(v)
}
func (col *SimpleAggregateFunction) Decode(decoder *binary.Decoder, rows int) error {
	return col.base.Decode(decoder, rows)
}
func (col *SimpleAggregateFunction) Encode(encoder *binary.Encoder) error {
	return col.base.Encode(encoder)
}

var _ Interface = (*SimpleAggregateFunction)(nil)
