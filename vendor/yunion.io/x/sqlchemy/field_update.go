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
	"yunion.io/x/pkg/util/reflectutils"
)

/*
func (ts *STableSpec) GetUpdateColumnValue(dataType reflect.Type, dataValue reflect.Value, cv map[string]interface{}, fields map[string]interface{}) error {
	for i := 0; i < dataType.NumField(); i++ {
		fieldType := dataType.Field(i)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			newValue, ok := fields[fieldType.Name]
			if ok && fieldType.Anonymous {
				return errors.New("Unsupported update anonymous field")
			}
			if ok {
				columnName := reflectutils.GetStructFieldName(&fieldType)
				cv[columnName] = newValue
				continue
			}
			if fieldType.Anonymous {
				err := ts.GetUpdateColumnValue(fieldType.Type, fieldValue, cv, fields)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
*/

func (ts *STableSpec) UpdateFields(dt interface{}, fields map[string]interface{}) error {
	return ts.updateFields(dt, fields, false)
}

// params dt: model struct, fileds: {struct-field-name-string: update-value}
// find primary key and index key
// find fields correlatively columns
// joint sql and executed
func (ts *STableSpec) updateFields(dt interface{}, fields map[string]interface{}, debug bool) error {
	dataValue := reflect.Indirect(reflect.ValueOf(dt))

	// cv: {"column name": "update value"}
	cv := make(map[string]interface{})
	// dataType := dataValue.Type()
	// ts.GetUpdateColumnValue(dataType, dataValue, cv, fields)
	// if len(cv) == 0 {
	// 	log.Infof("Nothing update")
	// 	return nil
	// }

	fullFields := reflectutils.FetchStructFieldValueSet(dataValue)
	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaryCols := make(map[string]interface{}, 0)
	for _, col := range ts.Columns() {
		name := col.Name()
		colValue, ok := fullFields.GetInterface(name)
		if !ok {
			continue
		}
		if col.IsPrimary() && !col.IsZero(colValue) {
			primaryCols[name] = colValue
			continue
		}
		intCol, ok := col.(*SIntegerColumn)
		if ok && intCol.IsAutoVersion {
			versionFields = append(versionFields, name)
			continue
		}
		dateCol, ok := col.(*SDateTimeColumn)
		if ok && dateCol.IsUpdatedAt {
			updatedFields = append(updatedFields, name)
			continue
		}
		if _, exist := fields[name]; exist {
			cv[name] = col.ConvertFromValue(fields[name])
		}
	}

	vars := make([]interface{}, 0)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE `%s` SET ", ts.name))
	first := true
	for k, v := range cv {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", `%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", `%s` = UTC_TIMESTAMP()", updatedField))
	}
	buf.WriteString(" WHERE ")
	first = true
	if len(primaryCols) == 0 {
		return ErrEmptyPrimaryKey
	}

	for k, v := range primaryCols {
		if first {
			first = false
		} else {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}

	if DEBUG_SQLCHEMY || debug {
		log.Infof("Update: %s", buf.String())
	}
	results, err := _db.Exec(buf.String(), vars...)
	if err != nil {
		return err
	}
	aCnt, err := results.RowsAffected()
	if err != nil {
		return err
	}
	if aCnt > 1 {
		return errors.Wrapf(ErrUnexpectRowCount, "affected rows %d != 1", aCnt)
	}
	return nil
}
