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

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceParameterManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDBInstanceResourceBaseManager
}

var DBInstanceParameterManager *SDBInstanceParameterManager

func init() {
	DBInstanceParameterManager = &SDBInstanceParameterManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDBInstanceParameter{},
			"dbinstanceparameters_tbl",
			"dbinstanceparameter",
			"dbinstanceparameters",
		),
	}
	DBInstanceParameterManager.SetVirtualObject(DBInstanceParameterManager)
}

type SDBInstanceParameter struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase
	SDBInstanceResourceBase `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
	// DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`

	// 数据库参数名称
	Key string `width:"64" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required" json:"key"`

	// 数据库参数值
	Value string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required" json:"value"`
}

func (manager *SDBInstanceParameterManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

// 列出数据参数
func (manager *SDBInstanceParameterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceParameterListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SDBInstanceResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DBInstanceFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemFilter")
	}

	if len(query.Key) > 0 {
		q = q.In("key", query.Key)
	}
	if len(query.Value) > 0 {
		q = q.In("value", query.Value)
	}

	return q, nil
}

func (manager *SDBInstanceParameterManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceParameterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SDBInstanceResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DBInstanceFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SDBInstanceParameterManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDBInstanceResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDBInstanceParameterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstanceParameterManager) SyncDBInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, cloudParameters []cloudprovider.ICloudDBInstanceParameter) compare.SyncResult {
	lockman.LockRawObject(ctx, "dbinstance-parameters", instance.Id)
	defer lockman.ReleaseRawObject(ctx, "dbinstance-parameters", instance.Id)

	result := compare.SyncResult{}
	dbParameters, err := instance.GetDBInstanceParameters()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceParameter, 0)
	commondb := make([]SDBInstanceParameter, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceParameter, 0)
	added := make([]cloudprovider.ICloudDBInstanceParameter, 0)
	if err := compare.CompareSets(dbParameters, cloudParameters, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstanceParameter(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudDBInstanceParameter(ctx, userCred, instance, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SDBInstanceParameter) SyncWithCloudDBInstanceParameter(ctx context.Context, userCred mcclient.TokenCredential, extParameter cloudprovider.ICloudDBInstanceParameter) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Key = extParameter.GetKey()
		self.Value = extParameter.GetValue()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstanceParameter.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceParameterManager) newFromCloudDBInstanceParameter(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extParameter cloudprovider.ICloudDBInstanceParameter) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	parameter := SDBInstanceParameter{}
	parameter.SetModelManager(manager, &parameter)

	parameter.Name = fmt.Sprintf("%s-%s", instance.Name, extParameter.GetKey())
	parameter.DBInstanceId = instance.Id
	parameter.Description = extParameter.GetDescription()
	parameter.Key = extParameter.GetKey()
	parameter.Value = extParameter.GetValue()
	parameter.ExternalId = extParameter.GetGlobalId()

	err := manager.TableSpec().Insert(ctx, &parameter)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceParameter.Insert")
	}
	return nil
}

func (self *SDBInstanceParameter) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DBInstanceparameterDetails, error) {
	return api.DBInstanceparameterDetails{}, nil
}

func (manager *SDBInstanceParameterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceparameterDetails {
	rows := make([]api.DBInstanceparameterDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dbRows := manager.SDBInstanceResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.DBInstanceparameterDetails{
			StandaloneResourceDetails: stdRows[i],
			DBInstanceResourceInfo:    dbRows[i],
		}
	}

	return rows
}

func (manager *SDBInstanceParameterManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SDBInstanceResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SDBInstanceResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
