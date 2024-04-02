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
	"context"
	"reflect"
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/informer"
	"yunion.io/x/onecloud/pkg/util/nopanic"
	"yunion.io/x/onecloud/pkg/util/splitable"
)

type ITableSpec interface {
	Name() string
	Columns() []sqlchemy.IColumnSpec
	PrimaryColumns() []sqlchemy.IColumnSpec
	DataType() reflect.Type
	// CreateSQL() string
	Instance() *sqlchemy.STable
	ColumnSpec(name string) sqlchemy.IColumnSpec
	Insert(ctx context.Context, dt interface{}) error
	InsertOrUpdate(ctx context.Context, dt interface{}) error
	Update(ctx context.Context, dt interface{}, doUpdate func() error) (sqlchemy.UpdateDiffs, error)
	Fetch(dt interface{}) error
	// FetchAll(dest interface{}) error
	SyncSQL() []string
	Sync() error
	DropForeignKeySQL() []string
	AddIndex(unique bool, cols ...string) bool
	Increment(ctx context.Context, diff interface{}, target interface{}) error
	Decrement(ctx context.Context, diff interface{}, target interface{}) error

	GetSplitTable() *splitable.SSplitTableSpec

	GetTableSpec() *sqlchemy.STableSpec

	GetDBName() sqlchemy.DBName

	InformUpdate(ctx context.Context, dt interface{}, oldObj *jsonutils.JSONDict)
}

type sTableSpec struct {
	sqlchemy.ITableSpec
}

func newTableSpec(model interface{}, tableName string, indexField string, dateField string, maxDuration time.Duration, maxSegments int, dbName sqlchemy.DBName) ITableSpec {
	var itbl sqlchemy.ITableSpec
	if len(indexField) > 0 && len(dateField) > 0 {
		var err error
		itbl, err = splitable.NewSplitTableSpec(model, tableName, indexField, dateField, maxDuration, maxSegments, dbName)
		if err != nil {
			log.Errorf("NewSplitTableSpec %s %s", tableName, err)
			return nil
		} else {
			log.Debugf("table %s maxDuration %d hour maxSegements %d", tableName, maxDuration/time.Hour, maxSegments)
		}
	} else if len(dbName) > 0 {
		itbl = sqlchemy.NewTableSpecFromStructWithDBName(model, tableName, dbName)
	} else {
		itbl = sqlchemy.NewTableSpecFromStruct(model, tableName)
	}
	return &sTableSpec{
		ITableSpec: itbl,
	}
}

func newClickhouseTableSpecFromMySQL(spec ITableSpec, name string, dbName sqlchemy.DBName, extraOpts sqlchemy.TableExtraOptions) ITableSpec {
	itbl := sqlchemy.NewTableSpecFromISpecWithDBName(spec.(*sTableSpec).ITableSpec, name, dbName, extraOpts)
	return &sTableSpec{
		ITableSpec: itbl,
	}
}

func (ts *sTableSpec) GetSplitTable() *splitable.SSplitTableSpec {
	sts, ok := ts.ITableSpec.(*splitable.SSplitTableSpec)
	if ok {
		return sts
	}
	return nil
}

func (ts *sTableSpec) GetDBName() sqlchemy.DBName {
	sts, ok := ts.ITableSpec.(*sqlchemy.STableSpec)
	if ok {
		dbName := sts.DBName()
		return dbName
	}
	return sqlchemy.DefaultDB
}

func (ts *sTableSpec) newInformerModel(dt interface{}) (*informer.ModelObject, error) {
	obj, ok := dt.(IModel)
	if !ok {
		return nil, errors.Errorf("informer model is not IModel")
	}
	if obj.GetVirtualObject() == nil {
		return nil, errors.Errorf("object %#v virtual object is nil", obj)
	}
	if obj.GetModelManager() == nil {
		return nil, errors.Errorf("object %#v model manager is nil", obj)
	}
	jointObj, isJoint := obj.(IJointModel)
	if isJoint {
		mObj := JointMaster(jointObj)
		if gotypes.IsNil(mObj) {
			return nil, errors.Errorf("object %#v master is nil", obj)
		}
		sObj := JointSlave(jointObj)
		if gotypes.IsNil(sObj) {
			return nil, errors.Errorf("object %#v slave is nil", obj)
		}
		return informer.NewJointModel(jointObj, jointObj.KeywordPlural(), mObj.GetId(), sObj.GetId()), nil
	}
	return informer.NewModel(obj, obj.KeywordPlural(), obj.GetId()), nil
}

func (ts *sTableSpec) isMarkDeleted(dt interface{}) (bool, error) {
	if vObj, ok := dt.(IVirtualModel); ok {
		if vObj.GetPendingDeleted() {
			return true, nil
		}
	}
	obj, ok := dt.(IModel)
	if !ok {
		return false, errors.Errorf("informer model is not IModel")
	}
	return obj.GetDeleted(), nil
}

func (ts *sTableSpec) rejectRecordChecksumAfterInsert(model IModel) error {
	obj, ok := IsModelEnableRecordChecksum(model)
	if !ok {
		return nil
	}
	return UpdateModelChecksum(obj)
}

func (ts *sTableSpec) Insert(ctx context.Context, dt interface{}) error {
	if err := ts.ITableSpec.Insert(dt); err != nil {
		return err
	}
	ts.rejectRecordChecksumAfterInsert(dt.(IModel))
	ts.inform(ctx, dt, informer.Create)
	return nil
}

func (ts *sTableSpec) GetTableSpec() *sqlchemy.STableSpec {
	return ts.ITableSpec.(*sqlchemy.STableSpec)
}

func (ts *sTableSpec) calculateRecordChecksum(dt interface{}) (string, error) {
	return "", errors.ErrNotImplemented
}

func (ts *sTableSpec) InsertOrUpdate(ctx context.Context, dt interface{}) error {
	if err := ts.ITableSpec.InsertOrUpdate(dt); err != nil {
		return err
	}
	ts.rejectRecordChecksumAfterInsert(dt.(IModel))
	ts.inform(ctx, dt, informer.Create)
	return nil
}

func (ts *sTableSpec) CheckRecordChanged(dbObj IModel) error {
	return ts.CheckRecordChecksumConsistent(dbObj)
}

func (ts *sTableSpec) CheckRecordChecksumConsistent(model IModel) error {
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
		return errors.Errorf("Record %s(%s) checksum changed, expected(%s) != calculated(%s)", obj.Keyword(), obj.GetId(), savedChecksum, calChecksum)
	}
	return nil
}

func checksumTestNotify(ctx context.Context, action api.SAction, resType string, obj jsonutils.JSONObject) {

}

func (ts *sTableSpec) Update(ctx context.Context, dt interface{}, doUpdate func() error) (sqlchemy.UpdateDiffs, error) {
	model := dt.(IModel)
	dbObj, isEnableRecordChecksum := IsModelEnableRecordChecksum(model)
	if isEnableRecordChecksum {
		if err := ts.CheckRecordChanged(dbObj); err != nil {
			log.Errorf("checkRecordChanged when update error: %s", err)
			return nil, errors.Wrap(err, "checkRecordChanged when update")
		}
	}

	oldObj := jsonutils.Marshal(dt)
	diffs, err := ts.ITableSpec.Update(dt, func() error {
		if err := doUpdate(); err != nil {
			return err
		}
		if isEnableRecordChecksum {
			dbObj = dt.(IRecordChecksumModel)
			updateChecksum, err := CalculateModelChecksum(dbObj)
			if err != nil {
				return errors.Wrap(err, "CalculateModelChecksum for update")
			}
			dbObj.SetRecordChecksum(updateChecksum)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if diffs == nil {
		// no data to update
		return nil, nil
	}
	isDeleted, err := ts.isMarkDeleted(dt)
	if err != nil {
		return nil, errors.Wrap(err, "check is mark deleted")
	}
	if isDeleted {
		ts.inform(ctx, dt, informer.Delete)
	} else {
		ts.informUpdate(ctx, dt, oldObj.(*jsonutils.JSONDict))
	}
	return diffs, nil
}

func (ts *sTableSpec) Increment(ctx context.Context, diff, target interface{}) error {
	oldObj := jsonutils.Marshal(target)
	err := ts.ITableSpec.Increment(diff, target)
	if err != nil {
		return errors.Wrap(err, "Increment")
	}
	ts.informUpdate(ctx, target, oldObj.(*jsonutils.JSONDict))
	return nil
}

func (ts *sTableSpec) Decrement(ctx context.Context, diff, target interface{}) error {
	oldObj := jsonutils.Marshal(target)
	err := ts.ITableSpec.Decrement(diff, target)
	if err != nil {
		return err
	}
	ts.informUpdate(ctx, target, oldObj.(*jsonutils.JSONDict))
	return nil
}

func (ts *sTableSpec) inform(ctx context.Context, dt interface{}, f func(ctx context.Context, obj *informer.ModelObject) error) {
	if !informer.IsInit() {
		return
	}
	nf := func() {
		obj, err := ts.newInformerModel(dt)
		if err != nil {
			log.Warningf("newInformerModel error: %v", err)
			debug.PrintStack()
			return
		}
		if err := f(ctx, obj); err != nil {
			if errors.Cause(err) == informer.ErrBackendNotInit {
				log.V(4).Warningf("informer backend not init")
			} else {
				log.Errorf("call informer func error: %v", err)
			}
		}
	}
	nopanic.Run(nf)
}

func (ts *sTableSpec) informUpdate(ctx context.Context, dt interface{}, oldObj *jsonutils.JSONDict) {
	if !informer.IsInit() {
		return
	}
	nf := func() {
		obj, err := ts.newInformerModel(dt)
		if err != nil {
			log.Warningf("newInformerModel error: %v", err)
			debug.PrintStack()
			return
		}
		if err := informer.Update(ctx, obj, oldObj); err != nil {
			if errors.Cause(err) == informer.ErrBackendNotInit {
				log.V(4).Warningf("informer backend not init")
			} else {
				log.Errorf("call informer update func error: %v", err)
			}
		}
	}
	nopanic.Run(nf)
}

func (ts *sTableSpec) InformUpdate(ctx context.Context, dt interface{}, oldObj *jsonutils.JSONDict) {
	ts.informUpdate(ctx, dt, oldObj)
}
