// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ovsdb

import (
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

// The columns ovsdb maintains on its own.  They are never sent as part of a
// row of an insert or update operation
const (
	ColUuid    = "_uuid"
	ColVersion = "_version"
)

func isReservedColumn(name string) bool {
	return name == ColUuid || name == ColVersion
}

// jsonColumnName returns the ovsdb column a struct field stands for.  The
// generated row types of yunion.io/x/ovsdb carry it in their json tag
func jsonColumnName(f reflect.StructField) string {
	tag, ok := f.Tag.Lookup("json")
	if !ok {
		return ""
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return ""
	}
	return name
}

// RowFromIRow converts a generated row struct to a rfc7047 <row>.
//
// Columns holding their zero value are left out unless includeZero is set.
// Leaving them out is what an insert wants, so that the remote fills in the
// schema defaults; including them is what a full row update wants, so that
// columns not set on the Go side are reset to their default.
//
// The _uuid and _version columns are never included, they are not writable
func RowFromIRow(schema *types.Schema, irow types.IRow, includeZero bool) (Row, error) {
	tblName := irow.OvsdbTableName()
	tbl, ok := schema.Tables[tblName]
	if !ok {
		return nil, errors.Wrapf(ErrSchema, "unknown table %s", tblName)
	}

	rv := reflect.ValueOf(irow)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, errors.Wrapf(ErrValue, "%s: nil row", tblName)
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.Wrapf(ErrValue, "%s: want a struct, got %s", tblName, rv.Kind())
	}

	rt := rv.Type()
	row := Row{}
	for i := 0; i < rt.NumField(); i++ {
		name := jsonColumnName(rt.Field(i))
		if name == "" || isReservedColumn(name) {
			continue
		}
		col, ok := tbl.Columns[name]
		if !ok {
			// the generated code was made from a schema newer than the
			// one the remote runs.  Leaving the column out gives the
			// remote a chance to accept the rest of the row
			continue
		}
		val, zero, err := encodeColumn(col, rv.Field(i))
		if err != nil {
			return nil, errors.Wrapf(err, "%s.%s", tblName, name)
		}
		if zero && !includeZero {
			continue
		}
		if val == nil {
			continue
		}
		row[name] = val
	}
	return row, nil
}

// encodeColumn converts one struct field to a rfc7047 <value>.  It also
// reports whether the field holds its zero value
func encodeColumn(col types.Column, fv reflect.Value) (interface{}, bool, error) {
	typ := col.Type
	switch {
	case typ.Value.Type != "":
		// <map>
		if fv.Kind() != reflect.Map {
			return nil, false, errors.Wrapf(ErrValue, "map column wants a map, got %s", fv.Kind())
		}
		if fv.Len() == 0 {
			return Map{}, true, nil
		}
		m, err := encodeMap(typ, fv)
		return m, false, err
	case typ.Max != 1:
		// <set>
		if fv.Kind() != reflect.Slice {
			return nil, false, errors.Wrapf(ErrValue, "set column wants a slice, got %s", fv.Kind())
		}
		if fv.Len() == 0 {
			return Set{}, true, nil
		}
		elems := make([]interface{}, fv.Len())
		for i := range elems {
			el, err := encodeAtomic(typ.Key, fv.Index(i))
			if err != nil {
				return nil, false, err
			}
			elems[i] = el
		}
		return Set(elems), false, nil
	case typ.Min == 0:
		// optional scalar, held by pointer on the Go side
		if fv.Kind() != reflect.Ptr {
			return nil, false, errors.Wrapf(ErrValue, "optional column wants a pointer, got %s", fv.Kind())
		}
		if fv.IsNil() {
			// rfc7047 5.1, the empty set stands for an absent value
			return Set{}, true, nil
		}
		val, err := encodeAtomic(typ.Key, fv.Elem())
		return val, false, err
	default:
		// mandatory scalar
		zero := fv.IsZero()
		if zero && typ.Key.Type == types.Uuid {
			// there is no such thing as an empty uuid on the wire
			return nil, true, nil
		}
		val, err := encodeAtomic(typ.Key, fv)
		return val, zero, err
	}
}

func encodeMap(typ types.Type, fv reflect.Value) (Map, error) {
	keys := fv.MapKeys()
	entries := make(Map, 0, len(keys))
	for _, kv := range keys {
		k, err := encodeAtomic(typ.Key, kv)
		if err != nil {
			return nil, errors.Wrap(err, "map key")
		}
		v, err := encodeAtomic(typ.Value, fv.MapIndex(kv))
		if err != nil {
			return nil, errors.Wrap(err, "map value")
		}
		entries = append(entries, MapEntry{Key: k, Value: v})
	}
	// ovsdb maps are unordered, sorting them keeps the requests we make
	// stable and reviewable
	sort.Slice(entries, func(i, j int) bool {
		return mapKeyLess(entries[i].Key, entries[j].Key)
	})
	return entries, nil
}

func mapKeyLess(a, b interface{}) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av < bv
	case int64:
		bv, ok := b.(int64)
		return ok && av < bv
	case float64:
		bv, ok := b.(float64)
		return ok && av < bv
	case Uuid:
		bv, ok := b.(Uuid)
		return ok && av < bv
	}
	return false
}

func encodeAtomic(bt types.BaseType, fv reflect.Value) (interface{}, error) {
	switch bt.Type {
	case types.Uuid:
		if fv.Kind() != reflect.String {
			return nil, errors.Wrapf(ErrValue, "uuid wants a string, got %s", fv.Kind())
		}
		s := fv.String()
		if s == "" {
			return nil, errors.Wrap(ErrValue, "empty uuid")
		}
		return UuidValue(s), nil
	case types.String:
		if fv.Kind() != reflect.String {
			return nil, errors.Wrapf(ErrValue, "string wants a string, got %s", fv.Kind())
		}
		return fv.String(), nil
	case types.Integer:
		switch fv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(fv.Uint()), nil
		}
		return nil, errors.Wrapf(ErrValue, "integer wants an int, got %s", fv.Kind())
	case types.Real:
		switch fv.Kind() {
		case reflect.Float32, reflect.Float64:
			return fv.Float(), nil
		}
		return nil, errors.Wrapf(ErrValue, "real wants a float, got %s", fv.Kind())
	case types.Boolean:
		if fv.Kind() != reflect.Bool {
			return nil, errors.Wrapf(ErrValue, "boolean wants a bool, got %s", fv.Kind())
		}
		return fv.Bool(), nil
	}
	return nil, errors.Wrapf(ErrValue, "unknown atomic type %q", string(bt.Type))
}

// SetIRowFromRow fills a generated row struct with the columns of a rfc7047
// <row>.
//
// Columns the struct has no field for are skipped.  That happens when the
// remote runs a schema newer than the one the code was generated from
func SetIRowFromRow(irow types.IRow, row Row) error {
	names := make([]string, 0, len(row))
	for name := range row {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := irow.SetColumn(name, row[name]); err != nil {
			if errors.Cause(err) == types.ErrUnknownColumn {
				continue
			}
			return errors.Wrapf(err, "%s.%s", irow.OvsdbTableName(), name)
		}
	}
	return nil
}

// IRowFromRow makes a new row of the table and fills it with the <row>
func IRowFromRow(itbl types.ITable, row Row) (types.IRow, error) {
	irow := itbl.NewRow()
	if err := SetIRowFromRow(irow, row); err != nil {
		return nil, err
	}
	return irow, nil
}

// FillITable replaces the content of a generated table with the rows given
func FillITable(itbl types.ITable, rows []Row) error {
	tv := reflect.ValueOf(itbl)
	if tv.Kind() == reflect.Ptr && tv.Elem().Kind() == reflect.Slice {
		tv.Elem().Set(reflect.MakeSlice(tv.Elem().Type(), 0, len(rows)))
	}
	for _, row := range rows {
		irow, err := IRowFromRow(itbl, row)
		if err != nil {
			return err
		}
		itbl.AppendRow(irow)
	}
	return nil
}

// ColumnsOf returns the columns of a table as the schema has them, sorted by
// name.  The _uuid and _version columns are included
func ColumnsOf(schema *types.Schema, table string) ([]string, error) {
	tbl, ok := schema.Tables[table]
	if !ok {
		return nil, errors.Wrapf(ErrSchema, "unknown table %s", table)
	}
	cols := make([]string, 0, len(tbl.Columns))
	for name := range tbl.Columns {
		cols = append(cols, name)
	}
	sort.Strings(cols)
	return cols, nil
}
