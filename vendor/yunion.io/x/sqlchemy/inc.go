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

// Increment perform an incremental update on a record, the primary key of the record is specified in diff,
// the numeric fields of this record will be atomically added by the value of the corresponding field in diff
// if target is given as a pointer to a variable, the result will be stored in the target
// if target is not given, the updated result will be stored in diff
func (t *STableSpec) Increment(diff interface{}, target interface{}) error {
	if !t.Database().backend.CanUpdate() {
		return errors.ErrNotSupported
	}
	return t.incrementInternal(diff, "+", target)
}

// Decrement is similar to Increment methods, the difference is that this method will atomically decrease the numeric fields
// with the value of diff
func (t *STableSpec) Decrement(diff interface{}, target interface{}) error {
	if !t.Database().backend.CanUpdate() {
		return errors.ErrNotSupported
	}
	return t.incrementInternal(diff, "-", target)
}

func (t *STableSpec) incrementInternalSql(diff interface{}, opcode string, target interface{}) (*SUpdateSQLResult, error) {
	dataValue := reflect.Indirect(reflect.ValueOf(diff))
	fields := reflectutils.FetchStructFieldValueSet(dataValue)
	var targetFields reflectutils.SStructFieldValueSet
	if target != nil {
		targetValue := reflect.Indirect(reflect.ValueOf(target))
		targetFields = reflectutils.FetchStructFieldValueSet(targetValue)
	}

	qChar := t.Database().backend.QuoteChar()

	primaries := make([]sPrimaryKeyValue, 0)
	vars := make([]interface{}, 0)
	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	incFields := make([]string, 0)

	for _, c := range t.Columns() {
		k := c.Name()
		v, _ := fields.GetInterface(k)
		if c.IsPrimary() {
			if targetFields != nil {
				v, _ = targetFields.GetInterface(k)
			}
			if !gotypes.IsNil(v) && !c.IsZero(v) {
				primaries = append(primaries, sPrimaryKeyValue{
					key:   k,
					value: v,
				})
			} else if c.IsText() {
				primaries = append(primaries, sPrimaryKeyValue{
					key:   k,
					value: "",
				})
			} else {
				return nil, ErrEmptyPrimaryKey
			}
			continue
		}
		if c.IsUpdatedAt() {
			updatedFields = append(updatedFields, k)
			continue
		}
		if c.IsAutoVersion() {
			versionFields = append(versionFields, k)
			continue
		}
		if c.IsNumeric() && !c.IsZero(v) {
			incFields = append(incFields, k)
			vars = append(vars, v)
			continue
		}
	}

	if len(vars) == 0 {
		return nil, ErrNoDataToUpdate
	}
	if len(primaries) == 0 {
		return nil, ErrEmptyPrimaryKey
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE %s%s%s SET ", qChar, t.name, qChar))
	first := true
	for _, k := range incFields {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s%s%s = %s%s%s %s ?", qChar, k, qChar, qChar, k, qChar, opcode))
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", %s%s%s = %s%s%s + 1", qChar, versionField, qChar, qChar, versionField, qChar))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", %s%s%s = %s", qChar, updatedField, qChar, t.Database().backend.CurrentUTCTimeStampString()))
	}

	buf.WriteString(" WHERE ")
	for i, pkv := range primaries {
		if i > 0 {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("%s%s%s = ?", qChar, pkv.key, qChar))
		vars = append(vars, pkv.value)
	}

	if DEBUG_SQLCHEMY {
		log.Infof("Update: %s %s", buf.String(), vars)
	}

	return &SUpdateSQLResult{
		Sql:       buf.String(),
		Vars:      vars,
		primaries: primaries,
	}, nil
}

func (t *STableSpec) incrementInternal(diff interface{}, opcode string, target interface{}) error {
	if target == nil {
		if reflect.ValueOf(diff).Kind() != reflect.Ptr {
			return errors.Wrap(ErrNeedsPointer, "Incremental input must be a Pointer")
		}
	} else {
		if reflect.ValueOf(target).Kind() != reflect.Ptr {
			return errors.Wrap(ErrNeedsPointer, "Incremental update target must be a Pointer")
		}
	}

	intResult, err := t.incrementInternalSql(diff, opcode, target)

	if target != nil {
		err = t.execUpdateSql(target, intResult)
	} else {
		err = t.execUpdateSql(diff, intResult)
	}
	if err != nil {
		return errors.Wrap(err, "query after update failed")
	}

	return nil
}
