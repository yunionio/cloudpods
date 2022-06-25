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
	"fmt"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/binary"
	"github.com/paulmach/orb"
)

type Polygon struct {
	set *Array
}

func (col *Polygon) Type() Type {
	return "Polygon"
}

func (col *Polygon) ScanType() reflect.Type {
	return scanTypePolygon
}

func (col *Polygon) Rows() int {
	return col.set.Rows()
}

func (col *Polygon) Row(i int, ptr bool) interface{} {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *Polygon) ScanRow(dest interface{}, row int) error {
	switch d := dest.(type) {
	case *orb.Polygon:
		*d = col.row(row)
	case **orb.Polygon:
		*d = new(orb.Polygon)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Polygon",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *Polygon) Append(v interface{}) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.Polygon:
		values := make([][]orb.Ring, 0, len(v))
		for _, v := range v {
			values = append(values, v)
		}
		return col.set.Append(values)

	default:
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Polygon",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *Polygon) AppendRow(v interface{}) error {
	switch v := v.(type) {
	case orb.Polygon:
		return col.set.AppendRow([]orb.Ring(v))
	default:
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Polygon",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *Polygon) Decode(decoder *binary.Decoder, rows int) error {
	return col.set.Decode(decoder, rows)
}

func (col *Polygon) Encode(encoder *binary.Encoder) error {
	return col.set.Encode(encoder)
}

func (col *Polygon) row(i int) orb.Polygon {
	var value []orb.Ring
	{
		col.set.ScanRow(&value, i)
	}
	return value
}

var _ Interface = (*Polygon)(nil)
