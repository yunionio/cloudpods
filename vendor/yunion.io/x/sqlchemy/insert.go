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
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

func (t *STableSpec) Insert(dt interface{}) error {
	return t.insert(dt, false, false)
}

//
// MySQL: INSERT INTO ... ON DUPLICATE KEY UPDATE ...
// works only for the cases that all values of primary keys are determeted before insert
func (t *STableSpec) InsertOrUpdate(dt interface{}) error {
	return t.insert(dt, true, false)
}

func (t *STableSpec) insertSqlPrep(dataFields reflectutils.SStructFieldValueSet, update bool) (string, []interface{}, error) {
	var autoIncField string
	createdAtFields := make([]string, 0)

	names := make([]string, 0)
	format := make([]string, 0)
	values := make([]interface{}, 0)

	updates := make([]string, 0)
	updateValues := make([]interface{}, 0)

	for _, c := range t.columns {
		isAutoInc := false
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoIncrement {
			isAutoInc = true
		}

		k := c.Name()

		dtc, isDate := c.(*SDateTimeColumn)
		inc, isInt := c.(*SIntegerColumn)
		ov, find := dataFields.GetInterface(k)

		if !find {
			continue
		}

		if isDate && (dtc.IsCreatedAt || dtc.IsUpdatedAt) {
			createdAtFields = append(createdAtFields, k)
			names = append(names, fmt.Sprintf("`%s`", k))
			if c.IsZero(ov) {
				format = append(format, "UTC_TIMESTAMP()")
			} else {
				values = append(values, ov)
				format = append(format, "?")
			}

			if update && dtc.IsUpdatedAt {
				if c.IsZero(ov) {
					updates = append(updates, fmt.Sprintf("`%s` = UTC_TIMESTAMP()", k))
				} else {
					updates = append(updates, fmt.Sprintf("`%s` = ?", k))
					updateValues = append(updateValues, ov)
				}
			}

			continue
		}

		if update && isInt && inc.IsAutoVersion {
			updates = append(updates, fmt.Sprintf("`%s` = `%s` + 1", k, k))
			continue
		}

		_, isTextCol := c.(*STextColumn)
		if c.IsSupportDefault() && (len(c.Default()) > 0 || isTextCol) && !gotypes.IsNil(ov) && c.IsZero(ov) && !c.AllowZero() { // empty text value
			val := c.ConvertFromString(c.Default())
			values = append(values, val)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "?")

			if update {
				updates = append(updates, fmt.Sprintf("`%s` = ?", k))
				updateValues = append(updateValues, val)
			}
			continue
		}

		if !gotypes.IsNil(ov) && (!c.IsZero(ov) || (!c.IsPointer() && !c.IsText())) && !isAutoInc {
			v := c.ConvertFromValue(ov)
			values = append(values, v)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "?")

			if update {
				updates = append(updates, fmt.Sprintf("`%s` = ?", k))
				updateValues = append(updateValues, v)
			}
			continue
		}

		if c.IsPrimary() {
			if isAutoInc {
				if len(autoIncField) > 0 {
					panic(fmt.Sprintf("multiple auto_increment columns: %q, %q", autoIncField, k))
				}
				autoIncField = k
			} else if c.IsText() {
				values = append(values, "")
				names = append(names, fmt.Sprintf("`%s`", k))
				format = append(format, "?")
			} else {
				return "", nil, errors.Wrapf(ErrEmptyPrimaryKey, "cannot insert for null primary key %q", k)
			}

			continue
		}
	}

	insertSql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES(%s)",
		t.name,
		strings.Join(names, ", "),
		strings.Join(format, ", "))

	if update {
		insertSql += " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ", ")
		values = append(values, updateValues...)
	}
	return insertSql, values, nil
}

func (t *STableSpec) insert(data interface{}, update bool, debug bool) error {
	beforeInsertFunc := reflect.ValueOf(data).MethodByName("BeforeInsert")
	if beforeInsertFunc.IsValid() && !beforeInsertFunc.IsNil() {
		beforeInsertFunc.Call([]reflect.Value{})
	}

	dataValue := reflect.ValueOf(data).Elem()
	dataFields := reflectutils.FetchStructFieldValueSet(dataValue)
	insertSql, values, err := t.insertSqlPrep(dataFields, update)
	if err != nil {
		return err
	}

	if DEBUG_SQLCHEMY || debug {
		log.Debugf("%s values: %v", insertSql, values)
	}

	results, err := _db.Exec(insertSql, values...)
	if err != nil {
		return err
	}
	affectCnt, err := results.RowsAffected()
	if err != nil {
		return err
	}

	targetCnt := int64(1)
	if update {
		// for insertOrUpdate cases, if no duplication, targetCnt=1, else targetCnt=2
		targetCnt = 2
	}
	if affectCnt < 1 || affectCnt > targetCnt {
		return errors.Wrapf(ErrUnexpectRowCount, "Insert affected cnt %d != (1, %d)", affectCnt, targetCnt)
	}

	/*
		if len(autoIncField) > 0 {
			lastId, err := results.LastInsertId()
			if err == nil {
				val, ok := reflectutils.FindStructFieldValue(dataValue, autoIncField)
				if ok {
					gotypes.SetValue(val, fmt.Sprint(lastId))
				}
			}
		}
	*/

	// query the value, so default value can be feedback into the object
	// fields = reflectutils.FetchStructFieldNameValueInterfaces(dataValue)
	q := t.Query()
	for _, c := range t.columns {
		if c.IsPrimary() {
			nc, ok := c.(*SIntegerColumn)
			if ok && nc.IsAutoIncrement {
				lastId, err := results.LastInsertId()
				if err != nil {
					return errors.Wrap(err, "fetching lastInsertId failed")
				} else {
					q = q.Equals(c.Name(), lastId)
				}
			} else {
				priVal, _ := dataFields.GetInterface(c.Name())
				if !gotypes.IsNil(priVal) {
					q = q.Equals(c.Name(), priVal)
				}
			}
		}
	}
	err = q.First(data)
	if err != nil {
		return errors.Wrap(err, "query after insert failed")
	}

	return nil
}
