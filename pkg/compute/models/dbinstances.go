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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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
	SDeletePreventableResourceBase

	MasterInstanceId string `width:"128" charset:"ascii" list:"user" create:"optional"`
	VcpuCount        int    `nullable:"false" default:"1" list:"user" create:"optional"`
	VmemSizeMb       int    `nullable:"false" list:"user" create:"required"`
	StorageType      string `nullable:"false" list:"user" create:"required"` //存储类型
	DiskSizeGB       int    `nullable:"false" list:"user" create:"required"`
	Port             int    `nullable:"false" list:"user" create:"optional"`
	Category         string `nullable:"false" list:"user" create:"optional"` //实例类别，单机，高可用，只读

	Engine        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	InstanceType  string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	MaintainTime string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	SecgroupId            string `width:"128" charset:"ascii" list:"user" default:"default" create:"optional"`
	VpcId                 string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	ConnectionStr         string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	InternalConnectionStr string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	Zone1 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	Zone2 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	Zone3 string `width:"36" charset:"ascii" nullable:"false" create:"optional" list:"user"`

	ZoneId string `width:"36" charset:"ascii" nullable:"false" create:"optional"`
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

	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = managedResourceFilterByDomain(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (man *SDBInstanceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerId)
	addressV := validators.NewIPv4AddrValidator("address")
	secgroupV := validators.NewModelIdOrNameValidator("secgroup", "secgroup", ownerId)
	masterV := validators.NewModelIdOrNameValidator("master_instance", "dbinstance", ownerId)
	zone1V := validators.NewModelIdOrNameValidator("zone1", "zone", ownerId)
	zone2V := validators.NewModelIdOrNameValidator("zone2", "zone", ownerId)
	zone3V := validators.NewModelIdOrNameValidator("zone3", "zone", ownerId)
	keyV := map[string]validators.IValidator{
		"network":  networkV,
		"address":  addressV.Optional(true),
		"master":   masterV.ModelIdKey("master_instance_id").Optional(true),
		"secgroup": secgroupV.Optional(true),
		"zone1":    zone1V.ModelIdKey("zone1").Optional(true),
		"zone2":    zone2V.ModelIdKey("zone2").Optional(true),
		"zone3":    zone3V.ModelIdKey("zone3").Optional(true),
	}
	for _, v := range keyV {
		err := v.Validate(data)
		if err != nil {
			return nil, err
		}
	}

	input := &api.SDBInstanceCreateInput{}
	err := data.Unmarshal(input)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal input failed: %v", err)
	}

	network := networkV.Model.(*SNetwork)
	input.NetworkExternalId = network.ExternalId

	vpc := network.GetVpc()
	input.VpcId = vpc.Id
	input.ManagerId = vpc.ManagerId
	cloudprovider := vpc.GetCloudprovider()
	if cloudprovider == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed to get vpc %s(%s) cloudprovider", vpc.Name, vpc.Id))
	}
	if !cloudprovider.Enabled {
		return nil, httperrors.NewInputParameterError("cloudprovider %s(%s) disabled", cloudprovider.Name, cloudprovider.Id)
	}

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, err
	}
	input.CloudregionId = region.Id
	input.Cloudregion = region.Name
	input.Provider = region.Provider

	if addressV.IP != nil {
		ip, err := netutils.NewIPV4Addr(addressV.IP.String())
		if err != nil {
			return nil, err
		}
		if !network.IsAddressInRange(ip) {
			return nil, httperrors.NewInputParameterError("Ip %s not in network %s(%s) range", addressV.IP.String(), network.Name, network.Id)
		}
	}

	if duration, _ := data.GetString("duration"); len(duration) > 0 {
		billingCycle, err := billing.ParseBillingCycle(duration)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid duration %s", duration)
		}
		if !region.GetDriver().IsSupportedBillingCycle(billingCycle, man.KeywordPlural()) {
			return nil, httperrors.NewInputParameterError("unsupported duration %s", duration)
		}
		input.BillingType = billing_api.BILLING_TYPE_PREPAID
		input.BillingCycle = billingCycle.String()
	}

	if len(input.InstanceType) == 0 && (input.VcpuCount == 0 || input.VmemSizeMb == 0) {
		return nil, httperrors.NewMissingParameterError("Missing instance_type or vcpu_count, vmem_size_mb parameters")
	}

	engines, err := DBInstanceSkuManager.GetEngines(input.Provider, input.CloudregionId)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(input.Engine) == 0 {
		return nil, httperrors.NewMissingParameterError("engine")
	}

	if !utils.IsInStringArray(input.Engine, engines) {
		return nil, httperrors.NewInputParameterError("%s(%s) not support engine %s, only support %s", input.Provider, input.Cloudregion, input.Engine, engines)
	}

	if len(input.EngineVersion) == 0 {
		return nil, httperrors.NewMissingParameterError("engine_version")
	}

	versions, err := DBInstanceSkuManager.GetEngineVersions(input.Provider, input.CloudregionId, input.Engine)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !utils.IsInStringArray(input.EngineVersion, versions) {
		return nil, httperrors.NewInputParameterError("%s(%s) engine %s not support version %s, only support %s", input.Provider, input.Cloudregion, input.Engine, input.EngineVersion, versions)
	}

	if len(input.Category) == 0 {
		return nil, httperrors.NewMissingParameterError("category")
	}

	categories, err := DBInstanceSkuManager.GetCategories(input.Provider, input.CloudregionId, input.Engine, input.EngineVersion)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !utils.IsInStringArray(input.Category, categories) {
		return nil, httperrors.NewInputParameterError("%s(%s) engine %s(%s) not support category %s, only support %s", input.Provider, input.Cloudregion, input.Engine, input.EngineVersion, input.Category, categories)
	}

	if len(input.StorageType) == 0 {
		return nil, httperrors.NewMissingParameterError("storage_type")
	}

	storageTypes, err := DBInstanceSkuManager.GetStorageTypes(input.Provider, input.CloudregionId, input.Engine, input.EngineVersion, input.Category)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !utils.IsInStringArray(input.StorageType, storageTypes) {
		return nil, httperrors.NewInputParameterError("%s(%s) engine %s(%s) %s not support storage %s, only support %s", input.Provider, input.Cloudregion, input.Engine, input.EngineVersion, input.Category, input.StorageType, storageTypes)
	}

	instance := SDBInstance{}
	jsonutils.Update(&instance, input)
	skus, err := instance.GetAvailableDBInstanceSkus()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(skus) == 0 {
		return nil, httperrors.NewInputParameterError("not match any dbinstance sku")
	}

	if len(input.InstanceType) > 0 { //设置下cpu和内存的大小
		input.VcpuCount = skus[0].VcpuCount
		input.VmemSizeMb = skus[0].VmemSizeMb
	}

	input, err = region.GetDriver().ValidateCreateDBInstanceData(ctx, userCred, ownerId, input, skus, network)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SDBInstance) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SetStatus(userCred, api.DBINSTANCE_DEPLOYING, "")
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("DBInstanceCreateTask newTask error %s", err)
		return
	}
	task.ScheduleRun(nil)
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

func (self *SDBInstance) GetZone() (*SZone, error) {
	zone, err := ZoneManager.FetchById(self.ZoneId)
	if err != nil {
		return nil, err
	}
	return zone.(*SZone), nil
}

func (self *SDBInstance) GetNetwork() (*SNetwork, error) {
	dbnet := DBInstanceNetworkManager.Query().SubQuery()
	q := NetworkManager.Query()
	q = q.Join(dbnet, sqlchemy.Equals(q.Field("id"), dbnet.Field("network_id"))).Filter(sqlchemy.Equals(dbnet.Field("dbinstance_id"), self.Id))
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		network := &SNetwork{}
		network.SetModelManager(NetworkManager, network)
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

type sDBInstanceZone struct {
	Id   string
	Name string
}

func fetchDBInstanceZones(rdsIds []string) map[string][]sDBInstanceZone {
	instances := DBInstanceManager.Query().SubQuery()
	zones := ZoneManager.Query().SubQuery()

	result := map[string][]sDBInstanceZone{}

	for _, zone := range []string{"zone1", "zone2", "zone3"} {
		zoneInfo := []struct {
			InstanceId string
			Id         string
			Name       string
		}{}

		q := zones.Query(instances.Field("id").Label("instance_id"), zones.Field("id"), zones.Field("name"))
		q = q.Join(instances, sqlchemy.Equals(zones.Field("id"), instances.Field(zone)))
		q = q.Filter(sqlchemy.In(instances.Field("id"), rdsIds))
		q = q.Filter(sqlchemy.NOT(sqlchemy.IsNullOrEmpty(instances.Field(zone))))

		err := q.All(&zoneInfo)
		if err != nil {
			return nil
		}

		for _, _zone := range zoneInfo {
			if _, ok := result[_zone.InstanceId]; !ok {
				result[_zone.InstanceId] = []sDBInstanceZone{}
			}
			result[_zone.InstanceId] = append(result[_zone.InstanceId], sDBInstanceZone{
				Id:   fmt.Sprintf("%s_name", zone),
				Name: _zone.Name,
			})
		}
	}

	return result
}

func (manager *SDBInstanceManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	rdsIds := make([]string, len(objs))
	for i := range objs {
		rdsIds[i] = objs[i].GetId()
	}
	if len(fields) == 0 || fields.Contains("zone") {
		zoneInfo := fetchDBInstanceZones(rdsIds)
		if zoneInfo != nil {
			for i := range rows {
				if zones, ok := zoneInfo[objs[i].GetId()]; ok {
					for _, zone := range zones {
						rows[i].Add(jsonutils.NewString(zone.Name), zone.Id)
					}
				}
			}
		}
	}
	return rows
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
	vpc, _ := self.GetVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.Name), "vpc")
	}
	if len(self.SecgroupId) > 0 {
		if secgroup, _ := self.GetSecgroup(); secgroup != nil {
			extra.Add(jsonutils.NewString(secgroup.Name), "secgroup")
		}
	}

	if skus, _ := self.GetDBInstanceSkus(); len(skus) > 0 {
		extra.Add(jsonutils.NewInt(int64(skus[0].IOPS)), "iops")
	}

	network, _ := self.GetNetwork()
	if network != nil {
		extra.Add(jsonutils.NewString(network.Name), "network")
	}

	if metaData, err := self.GetAllMetadata(userCred); err == nil {
		extra.Add(jsonutils.Marshal(metaData), "metadata")
	}

	return extra
}

func (self *SDBInstance) GetSecgroup() (*SSecurityGroup, error) {
	secgroup, err := SecurityGroupManager.FetchById(self.SecgroupId)
	if err != nil {
		return nil, err
	}
	return secgroup.(*SSecurityGroup), nil
}

func (self *SDBInstance) GetMasterInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.MasterInstanceId)
	if err != nil {
		return nil, err
	}
	return instance.(*SDBInstance), nil
}

func (self *SDBInstance) GetIRegion() (cloudprovider.ICloudRegion, error) {
	driver, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to found region for rds %s(%s)", self.Name, self.Id)
	}
	return driver.GetIRegionById(region.ExternalId)
}

func (self *SDBInstance) GetIDBInstance() (cloudprovider.ICloudDBInstance, error) {
	iregion, err := self.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIRegion")
	}
	return iregion.GetIDBInstanceById(self.ExternalId)
}

func (self *SDBInstance) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	dataCopy := jsonutils.DeepCopy(data)
	backups, err := self.GetDBInstanceBackups()
	if err != nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed get backups: %v", err))
	}
	for i := range backups {
		_, err := backups[i].PerformChangeOwner(ctx, userCred, query, dataCopy)
		if err != nil {
			return nil, err
		}
	}
	return self.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, data)
}

func (self *SDBInstance) AllowPerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "recovery")
}

func (self *SDBInstance) PerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do recovery dbinstance in status %s required status %s", self.Status, api.DBINSTANCE_RUNNING)
	}

	params := data.(*jsonutils.JSONDict)
	backupV := validators.NewModelIdOrNameValidator("dbinstancebackup", "dbinstancebackup", userCred)
	err := backupV.Validate(params)
	if err != nil {
		return nil, err
	}
	input := &api.SDBInstanceRecoveryConfigInput{}
	err = params.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to unmarshal input config: %v", err)
	}

	databases, err := self.GetDBInstanceDatabases()
	if err != nil {
		return nil, err
	}

	dbDatabases := []string{}
	for _, database := range databases {
		dbDatabases = append(dbDatabases, database.Name)
	}

	backup := backupV.Model.(*SDBInstanceBackup)
	for src, dest := range input.Databases {
		if len(dest) == 0 {
			dest = src
		}
		if strings.Index(backup.DBNames, src) < 0 {
			return nil, httperrors.NewInputParameterError("backup %s(%s) not contain database %s", backup.Name, backup.Id, src)
		}

		if utils.IsInStringArray(dest, dbDatabases) {
			return nil, httperrors.NewConflictError("conflict database %s for instance %s(%s)", dest, self.Name, self.Id)
		}
		input.Databases[src] = dest
	}

	if backup.ManagerId != self.ManagerId {
		return nil, httperrors.NewInputParameterError("back and instance not in same cloudaccount")
	}

	if backup.CloudregionId != self.CloudregionId {
		return nil, httperrors.NewInputParameterError("backup and instance not in same cloudregion")
	}

	return nil, self.StartDBInstanceRecoveryTask(ctx, userCred, input.JSON(input), "")
}

func (self *SDBInstance) StartDBInstanceRecoveryTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_RESTORING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRecoveryTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SDBInstance) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("purge", jsonutils.JSONTrue)
	return nil, self.StartDBInstanceDeleteTask(ctx, userCred, params, "")
}

func (self *SDBInstance) AllowPerformReboot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "reboot")
}

func (self *SDBInstance) PerformReboot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING, api.DBINSTANCE_REBOOT_FAILED}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do reboot dbinstance in status %s", self.Status)
	}
	return nil, self.StartDBInstanceRebootTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SDBInstance) AllowPerformSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync-status")
}

func (self *SDBInstance) PerformSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, self.StartDBInstanceSyncStatusTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SDBInstance) AllowPerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "renew")
}

func (self *SDBInstance) PerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do renew dbinstance in status %s required status %s", self.Status, api.DBINSTANCE_RUNNING)
	}

	durationStr := jsonutils.GetAnyString(data, []string{"duration"})
	if len(durationStr) == 0 {
		return nil, httperrors.NewInputParameterError("missong duration")
	}

	bc, err := billing.ParseBillingCycle(durationStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", durationStr, err)
	}

	if !self.GetRegion().GetDriver().IsSupportedBillingCycle(bc, DBInstanceManager.KeywordPlural()) {
		return nil, httperrors.NewInputParameterError("unsupported duration %s", durationStr)
	}

	return nil, self.StartDBInstanceRenewTask(ctx, userCred, durationStr, "")
}

func (self *SDBInstance) AllowPerformPublicConnection(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "public-connection")
}

func (self *SDBInstance) PerformPublicConnection(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	open := jsonutils.QueryBoolean(data, "open", true)
	if open && len(self.ConnectionStr) > 0 {
		return nil, httperrors.NewInputParameterError("DBInstance has opened the outer network connection")
	}
	if !open && len(self.ConnectionStr) == 0 {
		return nil, httperrors.NewInputParameterError("The extranet connection is not open")
	}

	region := self.GetRegion()
	if region == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("failed to found region for dbinstance %s(%s)", self.Name, self.Id))
	}

	if !region.GetDriver().IsSupportDBInstancePublicConnection() {
		return nil, httperrors.NewInputParameterError("%s not support this operation", region.Provider)
	}

	return nil, self.StartDBInstancePublicConnectionTask(ctx, userCred, "", open)
}

func (self *SDBInstance) StartDBInstancePublicConnectionTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, open bool) error {
	self.SetStatus(userCred, api.DBINSTANCE_DEPLOYING, "")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(open), "open")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstancePublicConnectionTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) AllowPerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-config")
}

func (self *SDBInstance) PerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DBINSTANCE_RUNNING}) {
		return nil, httperrors.NewInputParameterError("Cannot change config in status %s", self.Status)
	}
	input := api.SDBInstanceChangeConfigInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input error: %v", err)
	}

	tmp := &SDBInstance{}
	jsonutils.Update(tmp, self)

	if len(input.StorageType) > 0 {
		self.StorageType = input.StorageType
	}

	changed := false
	if len(input.InstanceType) > 0 {
		tmp.InstanceType = input.InstanceType
		changed = true
	} else if input.VCpuCount > 0 {
		tmp.VcpuCount = input.VCpuCount
		self.InstanceType = ""
		changed = true
	} else if input.VmemSizeMb > 0 {
		tmp.VmemSizeMb = input.VmemSizeMb
		tmp.InstanceType = ""
		changed = true
	} else if len(input.Category) > 0 {
		tmp.Category = input.Category
		tmp.InstanceType = ""
		changed = true
	}

	if changed {
		skus, err := tmp.GetAvailableDBInstanceSkus()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrap(err, "self.GetAvailableDBInstanceSkus"))
		}
		if len(skus) == 0 {
			return nil, httperrors.NewInputParameterError("failed to match any skus for change config")
		}
	}

	err = self.GetRegion().GetDriver().ValidateChangeDBInstanceConfigData(ctx, userCred, self, &input)
	if err != nil {
		return nil, err
	}

	return nil, self.StartDBInstanceChangeConfig(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SDBInstance) StartDBInstanceChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_CHANGE_CONFIG, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceChangeConfigTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceRenewTask(ctx context.Context, userCred mcclient.TokenCredential, duration string, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_RENEWING, "")
	params := jsonutils.NewDict()
	params.Set("duration", jsonutils.NewString(duration))
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRenewTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) SaveRenewInfo(ctx context.Context, userCred mcclient.TokenCredential, bc *billing.SBillingCycle, expireAt *time.Time) error {
	_, err := db.Update(self, func() error {
		if self.BillingType != billing_api.BILLING_TYPE_PREPAID {
			self.BillingType = billing_api.BILLING_TYPE_PREPAID
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		log.Errorf("Update error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SDBInstance) StartDBInstanceDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceRebootTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_REBOOTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceRebootTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstance) StartDBInstanceSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceSyncStatusTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
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

func (self *SDBInstance) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	params.Set("keep_backup", jsonutils.NewBool(jsonutils.QueryBoolean(data, "keep_backup", false)))
	return self.StartDBInstanceDeleteTask(ctx, userCred, params, "")
}

func (self *SDBInstance) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("dbinstance delete do nothing")
	return nil
}

func (self *SDBInstance) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
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

func (self *SDBInstance) GetDBInstanceBackup(name string) (*SDBInstanceBackup, error) {
	q := DBInstanceBackupManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d backup %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found backup %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	backup := &SDBInstanceBackup{}
	backup.SetModelManager(DBInstanceBackupManager, backup)
	return backup, q.First(backup)
}

func (self *SDBInstance) GetDBInstanceDatabase(name string) (*SDBInstanceDatabase, error) {
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d database %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found database %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	database := &SDBInstanceDatabase{}
	database.SetModelManager(DBInstanceDatabaseManager, database)
	return database, q.First(database)
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

func (self *SDBInstance) GetDBInstanceAccount(name string) (*SDBInstanceAccount, error) {
	q := DBInstanceAccountManager.Query().Equals("dbinstance_id", self.Id)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("name"), name),
			sqlchemy.Equals(q.Field("id"), name),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("Duplicate %d account %s for dbinstance %s(%s)", count, name, self.Name, self.Id)
	}
	if count == 0 {
		return nil, fmt.Errorf("Failed to found account %s for dbinstance %s(%s)", name, self.Name, self.Id)
	}
	account := &SDBInstanceAccount{}
	account.SetModelManager(DBInstanceAccountManager, account)
	return account, q.First(account)
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

func (self *SDBInstance) GetDBInstancePrivilege(account, database string) (*SDBInstancePrivilege, error) {
	instances := DBInstanceManager.Query().SubQuery()
	accounts := DBInstanceAccountManager.Query().SubQuery()
	databases := DBInstanceDatabaseManager.Query().SubQuery()
	q := DBInstancePrivilegeManager.Query()
	q = q.Join(accounts, sqlchemy.Equals(accounts.Field("id"), q.Field("dbinstanceaccount_id"))).
		Join(databases, sqlchemy.Equals(databases.Field("id"), q.Field("dbinstancedatabase_id"))).
		Join(instances, sqlchemy.AND(sqlchemy.Equals(instances.Field("id"), accounts.Field("dbinstance_id")), sqlchemy.Equals(instances.Field("id"), databases.Field("dbinstance_id"))))
	q = q.Filter(
		sqlchemy.AND(
			sqlchemy.OR(
				sqlchemy.Equals(accounts.Field("id"), account),
				sqlchemy.Equals(accounts.Field("name"), account),
			),
			sqlchemy.OR(
				sqlchemy.Equals(databases.Field("id"), database),
				sqlchemy.Equals(databases.Field("name"), database),
			),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if count == 0 {
		return nil, sql.ErrNoRows
	}
	privilege := &SDBInstancePrivilege{}
	privilege.SetModelManager(DBInstancePrivilegeManager, privilege)
	err = q.First(privilege)
	if err != nil {
		return nil, errors.Wrap(err, "q.First()")
	}
	return privilege, nil
}

func (self *SDBInstance) GetDBInstanceBackupByMode(mode string) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	q := DBInstanceBackupManager.Query().Equals("dbinstance_id", self.Id)
	switch mode {
	case api.BACKUP_MODE_MANUAL, api.BACKUP_MODE_AUTOMATED:
		q = q.Equals("backup_mode", mode)
	}
	err := db.FetchModelObjects(DBInstanceBackupManager, q, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceBackups.FetchModelObjects")
	}
	return backups, nil

}

func (self *SDBInstance) GetDBInstanceBackups() ([]SDBInstanceBackup, error) {
	return self.GetDBInstanceBackupByMode("")
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

func (manager *SDBInstanceManager) SyncDBInstanceMasterId(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, cloudDBInstances []cloudprovider.ICloudDBInstance) {
	for _, instance := range cloudDBInstances {
		masterId := instance.GetMasterInstanceId()
		if len(masterId) > 0 {
			master, err := db.FetchByExternalId(manager, masterId)
			if err != nil {
				log.Errorf("failed to found master dbinstance by externalId: %s error: %v", masterId, err)
				continue
			}
			slave, err := db.FetchByExternalId(manager, instance.GetGlobalId())
			if err != nil {
				log.Errorf("failed to found local dbinstance by externalId %s error: %v", instance.GetGlobalId(), err)
				continue
			}
			localInstance := slave.(*SDBInstance)
			_, err = db.Update(localInstance, func() error {
				localInstance.MasterInstanceId = master.GetId()
				return nil
			})
			if err != nil {
				log.Errorf("failed to update dbinstance %s(%s) master instanceId error: %v", localInstance.Name, localInstance.Id, err)
			}
		}
	}
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
	return self.RealDelete(ctx, userCred)
}

func (self *SDBInstance) ValidateDeleteCondition(ctx context.Context) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("DBInstance is locked, cannot delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDBInstance) GetDBInstanceSkuQuery() *sqlchemy.SQuery {
	q := DBInstanceSkuManager.Query().Equals("storage_type", self.StorageType).Equals("category", self.Category).
		Equals("cloudregion_id", self.CloudregionId).Equals("engine", self.Engine).Equals("engine_version", self.EngineVersion)
	for k, v := range map[string]string{"zone1": self.Zone1, "zone2": self.Zone2, "zone3": self.Zone3, "zone_id": self.ZoneId} {
		if len(v) > 0 {
			q = q.Equals(k, v)
		}
	}
	if len(self.InstanceType) > 0 {
		q = q.Equals("name", self.InstanceType)
	} else {
		q = q.Equals("vcpu_count", self.VcpuCount).Equals("vmem_size_mb", self.VmemSizeMb)
	}
	return q
}

func (self *SDBInstance) GetAvailableDBInstanceSkus() ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery().Equals("status", api.DBINSTANCE_SKU_AVAILABLE)
	err := db.FetchModelObjects(DBInstanceSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil

}

func (self *SDBInstance) GetDBInstanceSkus() ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery()
	err := db.FetchModelObjects(DBInstanceSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func (self *SDBInstance) GetAvailableZoneIds() ([]string, error) {
	zoneIds := []string{}
	skus, err := self.GetDBInstanceSkus()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetDBInstanceSkus")
	}
	for _, sku := range skus {
		if !utils.IsInStringArray(sku.ZoneId, zoneIds) {
			zoneIds = append(zoneIds, sku.ZoneId)
		}
	}
	return zoneIds, nil
}

func (self *SDBInstance) GetAvailableInstanceTypes() ([]cloudprovider.SInstanceType, error) {
	instanceTypes := map[string]cloudprovider.SInstanceType{}
	skus, err := self.GetAvailableDBInstanceSkus()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetAvailableDBInstanceSkus")
	}

	for _, sku := range skus {
		if instanceType, ok := instanceTypes[sku.Name]; !ok {
			instanceTypes[sku.Name] = cloudprovider.SInstanceType{InstanceType: sku.Name, ZoneIds: []string{sku.ZoneId}}
		} else if !utils.IsInStringArray(sku.ZoneId, instanceType.ZoneIds) {
			instanceType.ZoneIds = append(instanceType.ZoneIds, sku.ZoneId)
		}
	}

	result := []cloudprovider.SInstanceType{}
	for _, instanceType := range instanceTypes {
		result = append(result, instanceType)
	}

	return result, nil
}

func (self *SDBInstance) setZoneInfo() error {
	sku := SDBInstanceSku{}
	q := self.GetDBInstanceSkuQuery()
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrapf(err, "q.CountWithError")
	}
	if count == 0 {
		q.DebugQuery()
		return fmt.Errorf("failed to fetch any sku for dbinstance %s(%s)", self.Name, self.Id)
	}
	if count > 1 {
		q.DebugQuery()
		return fmt.Errorf("fetch %d skus for dbinstance %s(%s)", count, self.Name, self.Id)
	}
	err = q.First(&sku)
	if err != nil {
		return errors.Wrap(err, "q.First()")
	}
	self.Zone1 = sku.Zone1
	self.Zone2 = sku.Zone2
	self.Zone3 = sku.Zone3
	return nil
}

func (self *SDBInstance) SetZoneInfo(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		return self.setZoneInfo()
	})
	return err
}

func (self *SDBInstance) SyncWithCloudDBInstance(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extInstance cloudprovider.ICloudDBInstance) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Engine = extInstance.GetEngine()
		self.EngineVersion = extInstance.GetEngineVersion()
		self.InstanceType = extInstance.GetInstanceType()
		self.VcpuCount = extInstance.GetVcpuCount()
		self.VmemSizeMb = extInstance.GetVmemSizeMB()
		self.DiskSizeGB = extInstance.GetDiskSizeGB()
		self.StorageType = extInstance.GetStorageType()
		self.Status = extInstance.GetStatus()

		self.ConnectionStr = extInstance.GetConnectionStr()
		self.InternalConnectionStr = extInstance.GetInternalConnectionStr()

		self.MaintainTime = extInstance.GetMaintainTime()

		self.ZoneId = extInstance.GetIZoneId()
		err := self.setZoneInfo()
		if err != nil {
			return errors.Wrap(err, "setZoneInfo")
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

func (self *SDBInstance) GetSlaveDBInstances() ([]SDBInstance, error) {
	dbinstances := []SDBInstance{}
	q := DBInstanceManager.Query().Equals("master_instance_id", self.Id)
	return dbinstances, db.FetchModelObjects(DBInstanceManager, q, &dbinstances)
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
	instance.StorageType = extInstance.GetStorageType()
	instance.InternalConnectionStr = extInstance.GetInternalConnectionStr()

	instance.MaintainTime = extInstance.GetMaintainTime()
	instance.ZoneId = extInstance.GetIZoneId()
	err = instance.setZoneInfo()
	if err != nil {
		return nil, errors.Wrap(err, "instance.setZoneInfo")
	}

	if secgroupId := extInstance.GetSecurityGroupId(); len(secgroupId) > 0 {
		q := SecurityGroupCacheManager.Query().Equals("manager_id", provider.Id).Equals("external_id", secgroupId)
		count, err := q.CountWithError()
		if err != nil {
			log.Errorf("failed get secgroup cache by externalId %s error: %v", secgroupId, err)
		} else if count > 0 {
			cache := SSecurityGroupCache{}
			err = q.First(&cache)
			if err != nil {
				log.Errorf("failed get secgroup cache by externalId %s error: %v", secgroupId, err)
			} else {
				instance.SecgroupId = cache.SecgroupId
			}
		}
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

func (man *SDBInstanceManager) TotalCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
) (int, error) {
	q := man.Query()
	q = scopeOwnerIdFilter(q, scope, ownerId)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = rangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"))
	return q.CountWithError()
}

func (dbinstance *SDBInstance) GetQuotaKeys() quotas.IQuotaKeys {
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		dbinstance.GetOwnerId(),
		dbinstance.GetRegion(),
		dbinstance.GetCloudprovider(),
	)
}

func (dbinstance *SDBInstance) GetUsages() []db.IUsage {
	if dbinstance.PendingDeleted || dbinstance.Deleted {
		return nil
	}
	usage := SRegionQuota{Rds: 1}
	keys := dbinstance.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}
