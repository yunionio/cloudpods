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

package db

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/util/splitable"
)

var checksumTestFailedNotifier func(obj *jsonutils.JSONDict)

func SetChecksumTestFailedNotifier(notifier func(obj *jsonutils.JSONDict)) {
	checksumTestFailedNotifier = notifier
}

type IRecordChecksumResourceBase interface {
	GetRecordChecksum() string
	SetRecordChecksum(checksum string)
}

type IRecordChecksumModelManager interface {
	IModelManager
	EnableRecordChecksum() bool
	SetEnableRecordChecksum(bool)
}

type IRecordChecksumModel interface {
	IModel
	IRecordChecksumResourceBase
}

type SRecordChecksumResourceBaseManager struct {
	enableRecordChecksum bool
}

func NewRecordChecksumResourceBaseManager() *SRecordChecksumResourceBaseManager {
	return &SRecordChecksumResourceBaseManager{
		enableRecordChecksum: true,
	}
}

func (man *SRecordChecksumResourceBaseManager) EnableRecordChecksum() bool {
	return man.enableRecordChecksum
}

func (man *SRecordChecksumResourceBaseManager) SetEnableRecordChecksum(enable bool) {
	man.enableRecordChecksum = enable
}

// +onecloud:model-api-gen
type SRecordChecksumResourceBase struct {
	RecordChecksum string `width:"256" charset:"ascii" nullable:"true" list:"user" get:"user" json:"record_checksum"`
}

func (model *SRecordChecksumResourceBase) SetRecordChecksum(checksum string) {
	model.RecordChecksum = checksum
}

func (model *SRecordChecksumResourceBase) GetRecordChecksum() string {
	return model.RecordChecksum
}

func IsModelEnableRecordChecksum(model IModel) (IRecordChecksumModel, bool) {
	obj, ok := model.(IRecordChecksumModel)
	if !ok {
		return nil, false
	}
	man := obj.GetModelManager().(IRecordChecksumModelManager)
	return obj, man.EnableRecordChecksum()
}

func CheckRecordChecksumConsistent(model IModel) error {
	obj, ok := IsModelEnableRecordChecksum(model)
	if !ok {
		return nil
	}
	calChecksum, err := CalculateModelChecksum(obj)
	if err != nil {
		return errors.Wrap(err, "CalculateModelChecksum")
	}
	savedChecksum := obj.GetRecordChecksum()
	if calChecksum != savedChecksum {
		log.Errorf("Record %s(%s) checksum changed, expected(%s) != calculated(%s)", obj.Keyword(), obj.GetId(), savedChecksum, calChecksum)
		ts := model.GetModelManager().TableSpec()
		// notify
		data := jsonutils.NewDict()
		spt := ts.GetSplitTable()
		tableName := ts.Name()
		if spt != nil {
			tableName = spt.Name()
		}
		data.Set("db_name", jsonutils.NewString(string(ts.GetDBName())))
		data.Set("table_name", jsonutils.NewString(tableName))
		data.Set("name", jsonutils.NewString(fmt.Sprintf("%s(%s)", obj.Keyword(), obj.GetId())))
		data.Set("expected_checksum", jsonutils.NewString(savedChecksum))
		data.Set("calculated_checksum", jsonutils.NewString(calChecksum))
		if checksumTestFailedNotifier != nil {
			checksumTestFailedNotifier(data)
		}
		return errors.Errorf("Record %s(%s) checksum changed, expected(%s) != calculated(%s)", obj.Keyword(), obj.GetId(), savedChecksum, calChecksum)
	}
	return nil
}

func calculateRecordChecksumByValues(vals []interface{}) string {
	ss := ""
	for _, val := range vals {
		ss += fmt.Sprintf("\n%v", val)
	}
	hStr := md5.Sum([]byte(ss))
	sum := fmt.Sprintf("%x", hStr)
	log.Debugf("calculate values string: %s checksum: %s", ss, sum)
	return sum
}

func CalculateModelChecksum(dbObj IRecordChecksumModel) (string, error) {
	objMan := dbObj.GetModelManager()
	if objMan == nil {
		return "", errors.Errorf("Object %#v not set model manager", dbObj)
	}

	dataValue := reflect.ValueOf(dbObj).Elem()
	dataFields := reflectutils.FetchStructFieldValueSet(dataValue)

	cols := objMan.TableSpec().Columns()
	vals := []interface{}{}
	keys := make([]string, 0)
	for _, c := range cols {
		keys = append(keys, c.Name())
	}
	sort.Strings(keys)
	for _, k := range keys {

		if utils.IsInStringArray(k, []string{COLUMN_RECORD_CHECKSUM, COLUMN_UPDATE_VERSION, COLUMN_UPDATED_AT}) {
			continue
		}
		v, find := dataFields.GetInterface(k)
		if !find {
			continue
		}
		vals = append(vals, v)
	}

	return calculateRecordChecksumByValues(vals), nil
}

func UpdateModelChecksum(dbObj IRecordChecksumModel) error {
	var ts sqlchemy.ITableSpec
	tss := dbObj.GetModelManager().TableSpec().GetSplitTable()
	if tss != nil {
		meta, err := tss.GetTableMetaByObject(dbObj.(splitable.ISplitTableObject))
		if err != nil {
			return errors.Wrapf(err, "GetTableMetaByObject")
		}
		ts = tss.GetTableSpec(*meta)
	} else {
		ts = dbObj.GetModelManager().TableSpec().GetTableSpec()
	}
	updateChecksum, err := CalculateModelChecksum(dbObj)
	if err != nil {
		return errors.Wrap(err, "CalculateModelChecksum for update")
	}
	_, err = ts.Update(dbObj, func() error {
		dbObj.SetRecordChecksum(updateChecksum)
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "UpdateModelChecksum to %s", updateChecksum)
	}
	return nil
}

func InjectModelsChecksum(man IRecordChecksumModelManager) error {
	limit := 2048
	q := man.Query()
	totalCnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "Get total records count")
	}
	setCnt := 0
	for {
		if setCnt >= totalCnt {
			break
		}
		q = q.Limit(limit).Offset(setCnt)
		objs, err := FetchIModelObjects(man, q)
		if err != nil {
			return errors.Wrap(err, "FetchModelObjects")
		}
		for i := range objs {
			obj := objs[i].(IRecordChecksumModel)
			err := UpdateModelChecksum(obj)
			if err != nil {
				return errors.Wrapf(err, "UpdateModelChecksum for %s %s", man.Keyword(), obj.GetId())
			} else {
				log.Debugf("object %s %s %s calculate checksum completed", obj.Keyword(), obj.GetId(), obj.GetName())
			}
		}
		setCnt += len(objs)
	}
	return nil
}
