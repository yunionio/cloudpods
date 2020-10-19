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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
)

type SUpdateSession struct {
	oValue    reflect.Value
	tableSpec *STableSpec
}

func (ts *STableSpec) prepareUpdate(dt interface{}) (*SUpdateSession, error) {
	if reflect.ValueOf(dt).Kind() != reflect.Ptr {
		return nil, errors.Wrap(ErrNeedsPointer, "Update input must be a Pointer")
	}
	dataValue := reflect.ValueOf(dt).Elem()
	fields := reflectutils.FetchStructFieldValueSet(dataValue) //  fetchStructFieldNameValue(dataType, dataValue)

	zeroPrimary := make([]string, 0)
	for _, c := range ts.columns {
		k := c.Name()
		ov, ok := fields.GetInterface(k)
		if !ok {
			continue
		}
		if c.IsPrimary() && c.IsZero(ov) && !c.IsText() {
			zeroPrimary = append(zeroPrimary, k)
		}
	}

	if len(zeroPrimary) > 0 {
		return nil, errors.Wrapf(ErrEmptyPrimaryKey, "not a valid data, primary key %s empty",
			strings.Join(zeroPrimary, ","))
	}

	originValue := gotypes.DeepCopyRv(dataValue)
	us := SUpdateSession{oValue: originValue, tableSpec: ts}
	return &us, nil
}

type SUpdateDiff struct {
	old interface{}
	new interface{}
	col IColumnSpec
}

func (ud *SUpdateDiff) String() string {
	return fmt.Sprintf("%s->%s",
		utils.TruncateString(ud.old, 32),
		utils.TruncateString(ud.new, 32))
}

func (ud SUpdateDiff) jsonObj() jsonutils.JSONObject {
	r := jsonutils.NewDict()
	r.Set("old", jsonutils.Marshal(ud.old))
	r.Set("new", jsonutils.Marshal(ud.new))
	return r
}

type UpdateDiffs map[string]SUpdateDiff

func (uds UpdateDiffs) String() string {
	obj := jsonutils.NewDict()
	for k := range uds {
		obj.Set(k, uds[k].jsonObj())
	}
	return obj.String()
}

func (us *SUpdateSession) saveUpdate(dt interface{}) (UpdateDiffs, error) {
	beforeUpdateFunc := reflect.ValueOf(dt).MethodByName("BeforeUpdate")
	if beforeUpdateFunc.IsValid() && !beforeUpdateFunc.IsNil() {
		beforeUpdateFunc.Call([]reflect.Value{})
	}

	// dataType := reflect.TypeOf(dt).Elem()
	dataValue := reflect.ValueOf(dt).Elem()
	ofields := reflectutils.FetchStructFieldValueSet(us.oValue)
	fields := reflectutils.FetchStructFieldValueSet(dataValue)

	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaries := make(map[string]interface{})
	setters := UpdateDiffs{}
	for _, c := range us.tableSpec.columns {
		k := c.Name()
		of, _ := ofields.GetInterface(k)
		nf, _ := fields.GetInterface(k)
		if c.IsPrimary() {
			if !gotypes.IsNil(of) && !c.IsZero(of) {
				primaries[k] = c.ConvertFromValue(of)
			} else if c.IsText() {
				primaries[k] = ""
			} else {
				return nil, ErrEmptyPrimaryKey
			}
			continue
		}
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoVersion {
			versionFields = append(versionFields, k)
			continue
		}
		dtc, ok := c.(*SDateTimeColumn)
		if ok && dtc.IsUpdatedAt {
			updatedFields = append(updatedFields, k)
			continue
		}
		if reflect.DeepEqual(of, nf) {
			continue
		}
		if c.IsZero(nf) && c.IsText() {
			nf = nil
		}
		setters[k] = SUpdateDiff{old: of, new: nf, col: c}
	}

	if len(setters) == 0 {
		return nil, ErrNoDataToUpdate
	}

	vars := make([]interface{}, 0)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE `%s` SET ", us.tableSpec.name))
	first := true
	for k, v := range setters {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		if gotypes.IsNil(v.new) {
			buf.WriteString(fmt.Sprintf("`%s` = NULL", k))
		} else {
			buf.WriteString(fmt.Sprintf("`%s` = ?", k))
			vars = append(vars, v.col.ConvertFromValue(v.new))
		}
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", `%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", `%s` = UTC_TIMESTAMP()", updatedField))
	}
	buf.WriteString(" WHERE ")
	first = true
	if len(primaries) == 0 {
		return nil, ErrEmptyPrimaryKey
	}
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
		return nil, err
	}
	aCnt, err := results.RowsAffected()
	if err != nil {
		return nil, err
	}
	if aCnt != 1 {
		if aCnt == 0 {
			return nil, sql.ErrNoRows
		} else {
			return nil, errors.Wrapf(ErrUnexpectRowCount, "affected rows %d != 1", aCnt)
		}
	}
	q := us.tableSpec.Query()
	for k, v := range primaries {
		q = q.Equals(k, v)
	}
	err = q.First(dt)
	if err != nil {
		return nil, errors.Wrap(err, "query after update failed")
	}
	return setters, nil
}

func (ts *STableSpec) Update(dt interface{}, doUpdate func() error) (UpdateDiffs, error) {
	session, err := ts.prepareUpdate(dt)
	if err != nil {
		return nil, err
	}
	err = doUpdate()
	if err != nil {
		return nil, err
	}
	uds, err := session.saveUpdate(dt)
	if err == ErrNoDataToUpdate {
		return nil, nil
	} else if err == nil {
		if DEBUG_SQLCHEMY {
			log.Debugf("Update diff: %s", uds)
		}
	}
	return uds, err
}
