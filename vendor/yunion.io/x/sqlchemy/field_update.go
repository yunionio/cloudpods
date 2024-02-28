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

package sqlchemy

import (
	"bytes"
	"fmt"
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

// UpdateFields update a record with the values provided by fields stringmap
// params dt: model struct, fileds: {struct-field-name-string: update-value}
func (ts *STableSpec) UpdateFields(dt interface{}, fields map[string]interface{}) error {
	return ts.updateFields(dt, fields, false)
}

// params dt: model struct, fileds: {struct-field-name-string: update-value}
// find primary key and index key
// find fields correlatively columns
// joint sql and executed
func (ts *STableSpec) updateFieldSql(dt interface{}, fields map[string]interface{}, debug bool) (*SUpdateSQLResult, error) {
	dataValue := reflect.Indirect(reflect.ValueOf(dt))

	cv := make(map[string]interface{})
	// use field to store field order
	cnames := make([]string, 0)

	fullFields := reflectutils.FetchStructFieldValueSet(dataValue)
	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaryCols := make([]sPrimaryKeyValue, 0)
	for _, col := range ts.Columns() {
		name := col.Name()
		colValue, ok := fullFields.GetInterface(name)
		if !ok {
			continue
		}
		if col.IsPrimary() {
			if !gotypes.IsNil(colValue) && !col.IsZero(colValue) {
				primaryCols = append(primaryCols, sPrimaryKeyValue{
					key:   name,
					value: colValue,
				})
			} else if col.IsText() {
				primaryCols = append(primaryCols, sPrimaryKeyValue{
					key:   name,
					value: "",
				})
			} else {
				return nil, ErrEmptyPrimaryKey
			}
			continue
		}
		if col.IsAutoVersion() {
			versionFields = append(versionFields, name)
			continue
		}
		if col.IsUpdatedAt() {
			updatedFields = append(updatedFields, name)
			continue
		}
		if _, exist := fields[name]; exist {
			cv[name] = col.ConvertFromValue(fields[name])
			cnames = append(cnames, name)
		}
	}

	if len(primaryCols) == 0 {
		return nil, ErrEmptyPrimaryKey
	}

	qChar := ts.Database().backend.QuoteChar()

	vars := make([]interface{}, 0)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE %s%s%s SET ", qChar, ts.name, qChar))
	for i, k := range cnames {
		v := cv[k]
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s%s%s = ?", qChar, k, qChar))
		vars = append(vars, v)
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", %s%s%s = %s%s%s + 1", qChar, versionField, qChar, qChar, versionField, qChar))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", %s%s%s = %s", qChar, updatedField, qChar, ts.Database().backend.CurrentUTCTimeStampString()))
	}
	buf.WriteString(" WHERE ")
	for i, pkv := range primaryCols {
		if i > 0 {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("%s%s%s = ?", qChar, pkv.key, qChar))
		vars = append(vars, pkv.value)
	}

	if DEBUG_SQLCHEMY || debug {
		log.Infof("Update: %s", buf.String())
	}

	return &SUpdateSQLResult{
		Sql:       buf.String(),
		Vars:      vars,
		primaries: primaryCols,
	}, nil
}

func (ts *STableSpec) updateFields(dt interface{}, fields map[string]interface{}, debug bool) error {
	results, err := ts.updateFieldSql(dt, fields, debug)
	if err != nil {
		return errors.Wrap(err, "updateFieldSql")
	}

	err = ts.execUpdateSql(dt, results)
	if err != nil {
		return errors.Wrap(err, "execUpdateSql")
	}

	return nil
}
