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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceDatabaseManager struct {
	db.SVirtualResourceBaseManager
}

var DBInstanceDatabaseManager *SDBInstanceDatabaseManager

func init() {
	DBInstanceDatabaseManager = &SDBInstanceDatabaseManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDBInstanceDatabase{},
			"dbinstancedatabases_tbl",
			"dbinstancedatabase",
			"dbinstancedatabases",
		),
	}
	DBInstanceDatabaseManager.SetVirtualObject(DBInstanceDatabaseManager)
}

type SDBInstanceDatabase struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	CharacterSet string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SDBInstanceDatabaseManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (self *SDBInstanceDatabaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceDatabaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceDatabase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceDatabase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	//只能创建或删除，避免update name后造成登录数据库名称异常
	return false
}

func (self *SDBInstanceDatabase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SDBInstanceDatabaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
	})
}

func (self *SDBInstanceDatabase) GetParentId() string {
	return self.DBInstanceId
}

func (manager *SDBInstanceDatabaseManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	parentId, _ := data.GetString("dbinstance_id")
	return parentId
}

func (manager *SDBInstanceDatabaseManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("dbinstance_id", parentId)
	}
	return q
}

func (manager *SDBInstanceDatabaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	instanceV := validators.NewModelIdOrNameValidator("dbinstance", "dbinstance", userCred)
	err := instanceV.Validate(data)
	if err != nil {
		return nil, err
	}

	input := &api.SDBInstanceDatabaseCreateInput{}
	err = data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to unmarshal input params: %v", err)
	}
	instance := instanceV.Model.(*SDBInstance)
	if instance.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInputParameterError("DBInstance %s(%s) status is %s require status is %s", instance.Name, instance.Id, instance.Status, api.DBINSTANCE_RUNNING)
	}
	region := instance.GetRegion()
	if region == nil {
		return nil, httperrors.NewInputParameterError("failed to found region for dbinstance %s(%s)", instance.Name, instance.Id)
	}
	for i, _account := range input.Accounts {
		account, err := instance.GetDBInstanceAccount(_account.Account)
		if err != nil {
			return nil, httperrors.NewInputParameterError("failed to found dbinstance %s(%s) account %s: %v", instance.Name, instance.Id, _account.Account, err)
		}
		input.Accounts[i].DBInstanceaccountId = account.Id
	}

	input, err = region.GetDriver().ValidateCreateDBInstanceDatabaseData(ctx, userCred, ownerId, instance, input)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SDBInstanceDatabase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	instance, _ := self.GetDBInstance()
	if instance != nil {
		self.SetProjectInfo(ctx, userCred, instance.ProjectId, instance.DomainId)
	}
	self.StartDBInstanceDatabaseCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SDBInstanceDatabase) StartDBInstanceDatabaseCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_DATABASE_CREATING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceDatabaseCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceDatabase) GetDBInstancePrivileges() ([]SDBInstancePrivilege, error) {
	privileges := []SDBInstancePrivilege{}
	q := DBInstancePrivilegeManager.Query().Equals("dbinstancedatabase_id", self.Id)
	err := db.FetchModelObjects(DBInstancePrivilegeManager, q, &privileges)
	if err != nil {
		return nil, err
	}
	return privileges, nil
}

func (self *SDBInstanceDatabase) GetDBInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.DBInstanceId)
	if err != nil {
		return nil, err
	}
	return instance.(*SDBInstance), nil
}

func (self *SDBInstanceDatabase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}

	return self.getMoreDetails(ctx, userCred, extra)
}

func (self *SDBInstanceDatabase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra, _ = self.getMoreDetails(ctx, userCred, extra)
	return extra
}

func (self *SDBInstanceDatabase) getPrivilegesDetails() (*jsonutils.JSONArray, error) {
	result := jsonutils.NewArray()
	privileges, err := self.GetDBInstancePrivileges()
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstancePrivileges")
	}
	for _, privilege := range privileges {
		detail, err := privilege.GetDetailedJson()
		if err != nil {
			return nil, errors.Wrap(err, "GetDetailedJson")
		}
		result.Add(detail)
	}
	return result, nil
}

func (self *SDBInstanceDatabase) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, extra *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	privileges, err := self.getPrivilegesDetails()
	if err != nil {
		return extra, err
	}
	extra.Add(privileges, "dbinstanceprivileges")
	return extra, nil
}

func (manager *SDBInstanceDatabaseManager) SyncDBInstanceDatabases(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, cloudDatabases []cloudprovider.ICloudDBInstanceDatabase) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))

	result := compare.SyncResult{}
	dbDatabases, err := instance.GetDBInstanceDatabases()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceDatabase, 0)
	commondb := make([]SDBInstanceDatabase, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
	added := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
	if err := compare.CompareSets(dbDatabases, cloudDatabases, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Purge(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstanceDatabase(ctx, userCred, instance, commonext[i])
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudDBInstanceDatabase(ctx, userCred, instance, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SDBInstanceDatabase) SyncWithCloudDBInstanceDatabase(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extDatabase cloudprovider.ICloudDBInstanceDatabase) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extDatabase.GetStatus()
		self.Name = extDatabase.GetName()
		self.CharacterSet = extDatabase.GetCharacterSet()
		self.ProjectId = instance.ProjectId
		self.DomainId = instance.DomainId

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstanceDatabase.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceDatabaseManager) newFromCloudDBInstanceDatabase(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extDatabase cloudprovider.ICloudDBInstanceDatabase) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	database := SDBInstanceDatabase{}
	database.SetModelManager(manager, &database)

	database.Name = extDatabase.GetName()
	database.DBInstanceId = instance.Id
	database.Status = extDatabase.GetStatus()
	database.CharacterSet = extDatabase.GetCharacterSet()
	database.ExternalId = extDatabase.GetGlobalId()
	database.ProjectId = instance.ProjectId
	database.DomainId = instance.DomainId

	err := manager.TableSpec().Insert(&database)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceDatabase.Insert")
	}
	return nil
}

func (self *SDBInstanceDatabase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("dbinstance database delete do nothing")
	return nil
}

func (self *SDBInstanceDatabase) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDBInstanceDatabase) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDBInstanceDatabaseDeleteTask(ctx, userCred, "")
}

func (self *SDBInstanceDatabase) StartDBInstanceDatabaseDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_DATABASE_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceDatabaseDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
