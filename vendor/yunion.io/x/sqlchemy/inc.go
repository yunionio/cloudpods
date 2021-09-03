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
	"database/sql"
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
	return t.incrementInternal(diff, "+", target)
}

// Decrement is similar to Increment methods, the difference is that this method will atomically decrease the numeric fields
// with the value of diff
func (t *STableSpec) Decrement(diff interface{}, target interface{}) error {
	return t.incrementInternal(diff, "-", target)
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

	dataValue := reflect.Indirect(reflect.ValueOf(diff))
	fields := reflectutils.FetchStructFieldValueSet(dataValue)
	var targetFields reflectutils.SStructFieldValueSet
	if target != nil {
		targetValue := reflect.Indirect(reflect.ValueOf(target))
		targetFields = reflectutils.FetchStructFieldValueSet(targetValue)
	}

	primaries := make(map[string]interface{})
	vars := make([]interface{}, 0)
	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	incFields := make([]string, 0)

	for _, c := range t.columns {
		k := c.Name()
		v, _ := fields.GetInterface(k)
		if c.IsPrimary() {
			if targetFields != nil {
				v, _ = targetFields.GetInterface(k)
			}
			if !gotypes.IsNil(v) && !c.IsZero(v) {
				primaries[k] = v
			} else if c.IsText() {
				primaries[k] = ""
			} else {
				return ErrEmptyPrimaryKey
			}
			continue
		}
		dtc, ok := c.(*SDateTimeColumn)
		if ok && dtc.IsUpdatedAt {
			updatedFields = append(updatedFields, k)
			continue
		}
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoVersion {
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
		return ErrNoDataToUpdate
	}
	if len(primaries) == 0 {
		return ErrEmptyPrimaryKey
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE `%s` SET ", t.name))
	first := true
	for _, k := range incFields {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = `%s` %s ?", k, k, opcode))
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", `%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", `%s` = UTC_TIMESTAMP()", updatedField))
	}

	buf.WriteString(" WHERE ")
	first = true
	for k, v := range primaries {
		if first {
			first = false
		} else {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}

	if DEBUG_SQLCHEMY {
		log.Infof("Update: %s %s", buf.String(), vars)
	}

	results, err := _db.Exec(buf.String(), vars...)
	if err != nil {
		return errors.Wrapf(err, "_db.Exec %s %#v", buf.String(), vars)
	}
	aCnt, err := results.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "results.RowsAffected")
	}
	if aCnt != 1 {
		if aCnt == 0 {
			return sql.ErrNoRows
		}
		return errors.Wrapf(ErrUnexpectRowCount, "affected rows %d != 1", aCnt)
	}
	q := t.Query()
	for k, v := range primaries {
		q = q.Equals(k, v)
	}
	if target != nil {
		err = q.First(target)
	} else {
		err = q.First(diff)
	}
	if err != nil {
		return errors.Wrap(err, "query after update failed")
	}
	return nil
}
