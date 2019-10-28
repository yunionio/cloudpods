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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceManager struct {
	db.SVirtualResourceBaseManager
}

var DBInstanceManager *SDBInstanceManager

func init() {
	DBInstanceManager = &SDBInstanceManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDBInstance{},
			"dbinstances_tbl",
			"dbinstance",
			"dbinstances",
		),
	}
	DBInstanceManager.SetVirtualObject(DBInstanceManager)
}

type SDBInstance struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase

	SCloudregionResourceBase
	SZoneResourceBase

	VcpuCount  int    `nullable:"false" default:"1" list:"user" create:"optional"` // Column(TINYINT, nullable=False, default=1)
	VmemSizeMb int    `nullable:"false" list:"user" create:"required"`             // Column(Integer, nullable=False)
	DiskSizeGB int    `nullable:"false" list:"user" create:"required"`
	Port       int    `nullable:"false" list:"user" create:"required"`
	Category   string `nullable:"true" list:"user" create:"optional"` //实例类别，单机，高可用，只读

	Engine        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	InstanceType  string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	MaintainTime string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	VpcId                 string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	ConnectionStr         string `width:"256" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	InternalConnectionStr string `width:"256" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (manager *SDBInstanceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SDBInstanceManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstance) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstance) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstance) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (man *SDBInstanceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "vpc", ModelKeyword: "vpc", OwnerId: userCred},
		{Key: "zone", ModelKeyword: "zone", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SDBInstanceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (self *SDBInstance) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, userCred, query, extra), nil
}

func (self *SDBInstance) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, err
	}
	return vpc.(*SVpc), nil
}

func (self *SDBInstance) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	accountInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if accountInfo != nil {
		extra.Update(accountInfo)
	}
	regionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	zoneInfo := self.SZoneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if zoneInfo != nil {
		extra.Update(zoneInfo)
	}
	vpc, _ := self.GetVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.Name), "vpc")
	}
	return extra
}

func (self *SDBInstance) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, userCred, query, extra)
}

func (manager *SDBInstanceManager) getDBInstancesByProviderId(providerId string) ([]SDBInstance, error) {
	instances := []SDBInstance{}
	err := fetchByManagerId(manager, providerId, &instances)
	if err != nil {
		return nil, errors.Wrapf(err, "getDBInstancesByProviderId.fetchByManagerId")
	}
	return instances, nil
}

func (self *SDBInstance) GetDBInstanceParameters() ([]SDBInstanceParameter, error) {
	params := []SDBInstanceParameter{}
	q := DBInstanceParameterManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceParameterManager, q, &params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceParameters.FetchModelObjects for instance %s", self.Id)
	}
	return params, nil
}

func (self *SDBInstance) GetDBInstanceDatabases() ([]SDBInstanceDatabase, error) {
	databases := []SDBInstanceDatabase{}
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceDatabaseManager, q, &databases)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceDatabases.FetchModelObjects for instance %s", self.Id)
	}
	return databases, nil
}

func (self *SDBInstance) GetDBInstanceAccounts() ([]SDBInstanceAccount, error) {
	accounts := []SDBInstanceAccount{}
	q := DBInstanceAccountManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceAccountManager, q, &accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceAccounts.FetchModelObjects for instance %s", self.Id)
	}
	return accounts, nil
}

func (self *SDBInstance) GetDBInstanceBackups() ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	q := DBInstanceBackupManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceBackupManager, q, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceBackups.FetchModelObjects")
	}
	return backups, nil
}

func (self *SDBInstance) GetDBDatabases() ([]SDBInstanceDatabase, error) {
	databases := []SDBInstanceDatabase{}
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceDatabaseManager, q, &databases)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBDatabases.FetchModelObjects")
	}
	return databases, nil
}

func (self *SDBInstance) GetDBParameters() ([]SDBInstanceParameter, error) {
	parameters := []SDBInstanceParameter{}
	q := DBInstanceParameterManager.Query().Equals("dbinstance_id", self.Id)
	err := db.FetchModelObjects(DBInstanceParameterManager, q, &parameters)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBParameters.FetchModelObjects")
	}
	return parameters, nil
}

func (self *SDBInstance) GetDBNetwork() (*SDBInstanceNetwork, error) {
	q := DBInstanceNetworkManager.Query().Equals("dbinstance_id", self.Id)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		network := &SDBInstanceNetwork{}
		network.SetModelManager(DBInstanceNetworkManager, network)
		err = q.First(network)
		if err != nil {
			return nil, err
		}
		return network, nil
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	return nil, sql.ErrNoRows
}

func (manager *SDBInstanceManager) SyncDBInstances(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, cloudDBInstances []cloudprovider.ICloudDBInstance) ([]SDBInstance, []cloudprovider.ICloudDBInstance, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

	localDBInstances := []SDBInstance{}
	remoteDBInstances := []cloudprovider.ICloudDBInstance{}
	syncResult := compare.SyncResult{}

	dbInstances, err := region.GetDBInstances(provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SDBInstance, 0)
	commondb := make([]SDBInstance, 0)
	commonext := make([]cloudprovider.ICloudDBInstance, 0)
	added := make([]cloudprovider.ICloudDBInstance, 0)
	if err := compare.CompareSets(dbInstances, cloudDBInstances, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudDBInstance(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstance(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localDBInstances = append(localDBInstances, commondb[i])
		remoteDBInstances = append(remoteDBInstances, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		instance, err := manager.newFromCloudDBInstance(ctx, userCred, syncOwnerId, provider, region, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, instance, added[i])
		localDBInstances = append(localDBInstances, *instance)
		remoteDBInstances = append(remoteDBInstances, added[i])
		syncResult.Add()
	}
	return localDBInstances, remoteDBInstances, syncResult
}

func (self *SDBInstance) syncRemoveCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SDBInstance) SyncWithCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extInstance cloudprovider.ICloudDBInstance) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Engine = extInstance.GetEngine()
		self.EngineVersion = extInstance.GetEngineVersion()
		self.InstanceType = extInstance.GetInstanceType()
		self.VcpuCount = extInstance.GetVcpuCount()
		self.VmemSizeMb = extInstance.GetVmemSizeMB()
		self.DiskSizeGB = extInstance.GetDiskSizeGB()
		self.Status = extInstance.GetStatus()

		self.ConnectionStr = extInstance.GetConnectionStr()
		self.InternalConnectionStr = extInstance.GetInternalConnectionStr()

		self.MaintainTime = extInstance.GetMaintainTime()

		if zoneId := extInstance.GetIZoneId(); len(zoneId) > 0 {
			zone, err := db.FetchByExternalId(ZoneManager, zoneId)
			if err != nil {
				return errors.Wrapf(err, "SyncWithCloudDBInstance.FetchZoneId")
			}
			self.ZoneId = zone.GetId()
		}

		if createdAt := extInstance.GetCreatedAt(); !createdAt.IsZero() {
			self.CreatedAt = createdAt
		}

		factory, err := provider.GetProviderFactory()
		if err != nil {
			return errors.Wrap(err, "SyncWithCloudDBInstance.GetProviderFactory")
		}

		if factory.IsSupportPrepaidResources() {
			self.BillingType = extInstance.GetBillingType()
			self.ExpiredAt = extInstance.GetExpiredAt()
		}

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SDBInstanceManager) newFromCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, extInstance cloudprovider.ICloudDBInstance) (*SDBInstance, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	instance := SDBInstance{}
	instance.SetModelManager(manager, &instance)

	newName, err := db.GenerateName(manager, ownerId, extInstance.GetName())
	if err != nil {
		return nil, err
	}
	instance.Name = newName

	instance.ExternalId = extInstance.GetGlobalId()
	instance.CloudregionId = region.Id
	instance.ManagerId = provider.Id
	instance.IsEmulated = extInstance.IsEmulated()
	instance.Status = extInstance.GetStatus()
	instance.Port = extInstance.GetPort()

	instance.Engine = extInstance.GetEngine()
	instance.EngineVersion = extInstance.GetEngineVersion()
	instance.InstanceType = extInstance.GetInstanceType()
	instance.Category = extInstance.GetCategory()
	instance.VcpuCount = extInstance.GetVcpuCount()
	instance.VmemSizeMb = extInstance.GetVmemSizeMB()
	instance.DiskSizeGB = extInstance.GetDiskSizeGB()
	instance.ConnectionStr = extInstance.GetConnectionStr()
	instance.InternalConnectionStr = extInstance.GetInternalConnectionStr()

	instance.MaintainTime = extInstance.GetMaintainTime()

	if zoneId := extInstance.GetIZoneId(); len(zoneId) > 0 {
		zone, err := db.FetchByExternalId(ZoneManager, zoneId)
		if err != nil {
			return nil, errors.Wrapf(err, "newFromCloudDBInstance.FetchZoneId")
		}
		instance.ZoneId = zone.GetId()
	}

	if vpcId := extInstance.GetIVpcId(); len(vpcId) > 0 {
		vpc, err := db.FetchByExternalId(VpcManager, vpcId)
		if err != nil {
			return nil, errors.Wrapf(err, "newFromCloudDBInstance.FetchVpcId")
		}
		instance.VpcId = vpc.GetId()
	}

	if createdAt := extInstance.GetCreatedAt(); !createdAt.IsZero() {
		instance.CreatedAt = createdAt
	}

	factory, err := provider.GetProviderFactory()
	if err != nil {
		return nil, errors.Wrap(err, "newFromCloudDBInstance.GetProviderFactory")
	}

	if factory.IsSupportPrepaidResources() {
		instance.BillingType = extInstance.GetBillingType()
		instance.ExpiredAt = extInstance.GetExpiredAt()
	}

	err = manager.TableSpec().Insert(&instance)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudDBInstance.Insert")
	}

	SyncCloudProject(userCred, &instance, ownerId, extInstance, instance.ManagerId)

	db.OpsLog.LogEvent(&instance, db.ACT_CREATE, instance.GetShortDesc(ctx), userCred)

	return &instance, nil
}
