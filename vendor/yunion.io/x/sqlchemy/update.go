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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
)

// SUpdateSession is a struct to store the state of a update session
type SUpdateSession struct {
	oValue    reflect.Value
	tableSpec *STableSpec
}

func (ts *STableSpec) PrepareUpdate(dt interface{}) (*SUpdateSession, error) {
	if reflect.ValueOf(dt).Kind() != reflect.Ptr {
		return nil, errors.Wrap(ErrNeedsPointer, "Update input must be a Pointer")
	}
	dataValue := reflect.ValueOf(dt).Elem()
	fields := reflectutils.FetchStructFieldValueSet(dataValue) //  fetchStructFieldNameValue(dataType, dataValue)

	zeroPrimary := make([]string, 0)
	for _, c := range ts.Columns() {
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

// SUpdateDiff is a struct to store the differences for an update of a column
type SUpdateDiff struct {
	old interface{}
	new interface{}
	col IColumnSpec
}

// String of SUpdateDiff returns the string representation of a SUpdateDiff
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

// UpdateDiffs is a map of SUpdateDiff whose key is the column name
type UpdateDiffs map[string]SUpdateDiff

// String of UpdateDiffs returns the string representation of UpdateDiffs
func (uds UpdateDiffs) String() string {
	obj := jsonutils.NewDict()
	for i := range uds {
		obj.Set(uds[i].col.Name(), uds[i].jsonObj())
	}
	return obj.String()
}

func updateDiffList2Map(diffs []SUpdateDiff) UpdateDiffs {
	ret := make(map[string]SUpdateDiff)
	for i := range diffs {
		ret[diffs[i].col.Name()] = diffs[i]
	}
	return ret
}

type sPrimaryKeyValue struct {
	key   string
	value interface{}
}

type SUpdateSQLResult struct {
	Sql       string
	Vars      []interface{}
	setters   []SUpdateDiff
	primaries []sPrimaryKeyValue
}

func (us *SUpdateSession) SaveUpdateSql(dt interface{}) (*SUpdateSQLResult, error) {
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
	primaries := make([]sPrimaryKeyValue, 0)
	setters := make([]SUpdateDiff, 0)
	for _, c := range us.tableSpec.Columns() {
		k := c.Name()
		of, _ := ofields.GetInterface(k)
		nf, _ := fields.GetInterface(k)
		if c.IsPrimary() {
			if !gotypes.IsNil(of) && !c.IsZero(of) {
				if c.IsText() {
					ov, _ := of.(string)
					nv, _ := nf.(string)
					if ov != nv && strings.EqualFold(ov, nv) {
						setters = append(setters, SUpdateDiff{old: of, new: nf, col: c})
					}
				}
				primaries = append(primaries, sPrimaryKeyValue{
					key:   k,
					value: c.ConvertFromValue(of),
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
		if c.IsAutoVersion() {
			versionFields = append(versionFields, k)
			continue
		}
		if c.IsUpdatedAt() {
			updatedFields = append(updatedFields, k)
			continue
		}
		if reflect.DeepEqual(of, nf) {
			continue
		}
		if of != nil && nf != nil {
			ofJsonStr := jsonutils.Marshal(of).String()
			nfJsonStr := jsonutils.Marshal(nf).String()
			if ofJsonStr == nfJsonStr {
				continue
			}
		}
		if c.IsZero(nf) && c.IsText() {
			nf = nil
		}
		setters = append(setters, SUpdateDiff{old: of, new: nf, col: c})
	}

	if len(setters) == 0 {
		return nil, ErrNoDataToUpdate
	}

	if len(primaries) == 0 {
		return nil, ErrEmptyPrimaryKey
	}

	vars := make([]interface{}, 0)
	colsets := make([]string, 0)
	conditions := make([]string, 0)
	for _, udif := range setters {
		if gotypes.IsNil(udif.new) {
			colsets = append(colsets, fmt.Sprintf("`%s` = NULL", udif.col.Name()))
		} else {
			colsets = append(colsets, fmt.Sprintf("`%s` = ?", udif.col.Name()))
			vars = append(vars, udif.col.ConvertFromValue(udif.new))
		}
	}
	for _, versionField := range versionFields {
		colsets = append(colsets, fmt.Sprintf("`%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		colsets = append(colsets, fmt.Sprintf("`%s` = %s", updatedField, us.tableSpec.Database().backend.CurrentUTCTimeStampString()))
	}
	for _, pkv := range primaries {
		conditions = append(conditions, fmt.Sprintf("`%s` = ?", pkv.key))
		vars = append(vars, pkv.value)
	}

	updateSql := templateEval(us.tableSpec.Database().backend.UpdateSQLTemplate(), struct {
		Table      string
		Columns    string
		Conditions string
	}{
		Table:      us.tableSpec.name,
		Columns:    strings.Join(colsets, ", "),
		Conditions: strings.Join(conditions, " AND "),
	})

	if DEBUG_SQLCHEMY {
		log.Infof("Update: %s %s", updateSql, vars)
	}

	return &SUpdateSQLResult{
		Sql:       updateSql,
		Vars:      vars,
		setters:   setters,
		primaries: primaries,
	}, nil
}

func (us *SUpdateSession) saveUpdate(dt interface{}) (UpdateDiffs, error) {
	sqlResult, err := us.SaveUpdateSql(dt)
	if err != nil {
		return nil, errors.Wrap(err, "saveUpateSql")
	}

	err = us.tableSpec.execUpdateSql(dt, sqlResult)
	if err != nil {
		return nil, errors.Wrap(err, "execUpdateSql")
	}

	return updateDiffList2Map(sqlResult.setters), nil
}

func (ts *STableSpec) execUpdateSql(dt interface{}, result *SUpdateSQLResult) error {
	results, err := ts.Database().TxExec(result.Sql, result.Vars...)
	if err != nil {
		return errors.Wrap(err, "TxExec")
	}

	if ts.Database().backend.CanSupportRowAffected() {
		aCnt, err := results.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "results.RowsAffected")
		}
		if aCnt > 1 {
			return errors.Wrapf(ErrUnexpectRowCount, "affected rows %d != 1", aCnt)
		}
	}
	q := ts.Query()
	for _, pkv := range result.primaries {
		q = q.Equals(pkv.key, pkv.value)
	}
	err = q.First(dt)
	if err != nil {
		return errors.Wrap(err, "query after update failed")
	}
	return nil
}

// Update method of STableSpec updates a record of a table,
// dt is the point to the struct storing the record
// doUpdate provides method to update the field of the record
func (ts *STableSpec) Update(dt interface{}, doUpdate func() error) (UpdateDiffs, error) {
	if !ts.Database().backend.CanUpdate() {
		return nil, errors.ErrNotSupported
	}
	session, err := ts.PrepareUpdate(dt)
	if err != nil {
		return nil, errors.Wrap(err, "prepareUpdate")
	}
	err = doUpdate()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	uds, err := session.saveUpdate(dt)
	if err != nil && errors.Cause(err) == ErrNoDataToUpdate {
		return nil, nil
	} else if err == nil {
		if DEBUG_SQLCHEMY {
			log.Debugf("Update diff: %s", uds)
		}
	}
	return uds, errors.Wrap(err, "saveUpdate")
}
