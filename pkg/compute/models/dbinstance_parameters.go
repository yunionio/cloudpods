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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceParameterManager struct {
	db.SStandaloneResourceBaseManager
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
	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`

	Key   string `width:"64" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Value string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
}

func (manager *SDBInstanceParameterManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (self *SDBInstanceParameterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceParameterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceParameter) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceParameter) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstanceParameter) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SDBInstanceParameterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
	})
}

func (manager *SDBInstanceParameterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstanceParameterManager) SyncDBInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, cloudParameters []cloudprovider.ICloudDBInstanceParameter) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))

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

	err := manager.TableSpec().Insert(&parameter)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceParameter.Insert")
	}
	return nil
}
