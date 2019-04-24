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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SElasticipManager struct {
	db.SVirtualResourceBaseManager
}

var ElasticipManager *SElasticipManager

func init() {
	ElasticipManager = &SElasticipManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SElasticip{},
			"elasticips_tbl",
			"eip",
			"eips",
		),
	}
	ElasticipManager.TableSpec().AddIndex(true, "associate_id", "associate_type")
}

type SElasticip struct {
	db.SVirtualResourceBase

	SManagedResourceBase
	SBillingResourceBase

	Mode string `width:"32" charset:"ascii" list:"user"`

	IpAddr string `width:"17" charset:"ascii" list:"user"`

	AssociateType string `width:"32" charset:"ascii" list:"user"`
	AssociateId   string `width:"256" charset:"ascii" list:"user"`

	Bandwidth int `list:"user" create:"required"`

	ChargeType string `name:"charge_type" list:"user" create:"required"`
	BgpType    string `list:"user" create:"optional"` // 目前只有华为云此字段是必需填写的。

	AutoDellocate tristate.TriState `default:"false" get:"user" create:"optional" update:"user"`

	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SElasticipManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	/*managerFilter, _ := query.GetString("manager")
	if len(managerFilter) > 0 {
		managerI, err := CloudproviderManager.FetchByIdOrName(userCred, managerFilter)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("cloud provider %s not found", managerFilter)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("manager_id", managerI.GetId())
	}*/

	regionFilter, _ := query.GetString("region")
	if len(regionFilter) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionFilter)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("cloud region %s not found", regionFilter)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("cloudregion_id", regionObj.GetId())
	}

	/*accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", account.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		subq := CloudproviderManager.Query("id").Equals("provider", providerStr).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}*/

	if query.Contains("usable") {
		usable := jsonutils.QueryBoolean(query, "usable", false)
		if usable {
			q = q.Equals("status", api.EIP_STATUS_READY)
			q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("associate_id")), sqlchemy.IsEmpty(q.Field("associate_id"))))
		}
	}

	return q, nil
}

func (manager *SElasticipManager) getEipsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SElasticip, error) {
	eips := make([]SElasticip, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(manager, q, &eips)
	if err != nil {
		return nil, err
	}
	return eips, nil
}

func (self *SElasticip) GetRegion() *SCloudregion {
	return CloudregionManager.FetchRegionById(self.CloudregionId)
}

func (self *SElasticip) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)

	// desc.Add(jsonutils.NewString(self.ChargeType), "charge_type")

	desc.Add(jsonutils.NewInt(int64(self.Bandwidth)), "bandwidth")
	desc.Add(jsonutils.NewString(self.Mode), "mode")
	desc.Add(jsonutils.NewString(self.IpAddr), "ip_addr")

	// region := self.GetRegion()
	// if len(region.ExternalId) > 0 {
	// regionInfo := strings.Split(region.ExternalId, "/")
	// if len(regionInfo) == 2 {
	// desc.Add(jsonutils.NewString(strings.ToLower(regionInfo[0])), "hypervisor")
	// desc.Add(jsonutils.NewString(regionInfo[1]), "region")
	// }
	//}

	billingInfo := SCloudBillingInfo{}

	billingInfo.SCloudProviderInfo = self.getCloudProviderInfo()

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	billingInfo.InternetChargeType = self.ChargeType

	if priceKey := self.GetMetadata("ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (manager *SElasticipManager) SyncEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, eips []cloudprovider.ICloudEIP, projectId string) compare.SyncResult {
	ownerProjId := projectId

	lockman.LockClass(ctx, manager, ownerProjId)
	defer lockman.ReleaseClass(ctx, manager, ownerProjId)

	// localEips := make([]SElasticip, 0)
	// remoteEips := make([]cloudprovider.ICloudEIP, 0)
	syncResult := compare.SyncResult{}

	dbEips, err := manager.getEipsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := range dbEips {
		if taskman.TaskManager.IsInTask(&dbEips[i]) {
			syncResult.Error(fmt.Errorf("object in task"))
			return syncResult
		}
	}

	removed := make([]SElasticip, 0)
	commondb := make([]SElasticip, 0)
	commonext := make([]cloudprovider.ICloudEIP, 0)
	added := make([]cloudprovider.ICloudEIP, 0)

	err = compare.CompareSets(dbEips, eips, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudEip(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudEip(ctx, userCred, provider, commonext[i], projectId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudEip(ctx, userCred, added[i], region, ownerProjId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			syncResult.Add()
		}
	}

	return syncResult
}

func (self *SElasticip) syncRemoveCloudEip(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return self.SetStatus(userCred, api.EIP_STATUS_UNKNOWN, "sync to delete")
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SElasticip) SyncInstanceWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudEIP) error {
	vm := self.GetAssociateVM()
	vmExtId := ext.GetAssociationExternalId()

	if vm == nil && len(vmExtId) == 0 {
		return nil
	}
	if vm != nil && vm.ExternalId == vmExtId {
		return nil
	}

	if vm != nil { // dissociate
		err := self.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("fail to dissociate vm: %s", err)
			return err
		}
	}

	if len(vmExtId) > 0 {
		newVM, err := GuestManager.FetchByExternalId(vmExtId)
		if err != nil {
			log.Errorf("fail to find vm by external ID %s", vmExtId)
			return err
		}
		err = self.AssociateVM(ctx, userCred, newVM.(*SGuest))
		if err != nil {
			log.Errorf("fail to associate with new vm %s", err)
			return err
		}
	}

	return nil
}

func (self *SElasticip) SyncWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudEIP, projectId string) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {

		// self.Name = ext.GetName()
		self.Bandwidth = ext.GetBandwidth()
		self.IpAddr = ext.GetIpAddr()
		self.Mode = ext.GetMode()
		self.Status = ext.GetStatus()
		self.ExternalId = ext.GetGlobalId()
		// self.ManagerId = ext.GetManagerId()
		self.IsEmulated = ext.IsEmulated()

		self.ChargeType = ext.GetInternetChargeType()

		factory, _ := provider.GetProviderFactory()
		if factory != nil && factory.IsSupportPrepaidResources() {
			self.BillingType = ext.GetBillingType()
			self.ExpiredAt = ext.GetExpiredAt()
		}

		if createAt := ext.GetCreatedAt(); !createAt.IsZero() {
			self.CreatedAt = createAt
		}

		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudEip fail %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, projectId, ext, self.ManagerId)

	return nil
}

func (manager *SElasticipManager) newFromCloudEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, region *SCloudregion, projectId string) (*SElasticip, error) {
	eip := SElasticip{}
	eip.SetModelManager(manager)

	newName, err := db.GenerateName(manager, projectId, extEip.GetName())
	if err != nil {
		return nil, err
	}
	eip.Name = newName
	eip.Status = extEip.GetStatus()
	eip.ExternalId = extEip.GetGlobalId()
	eip.IpAddr = extEip.GetIpAddr()
	eip.Mode = extEip.GetMode()
	eip.IsEmulated = extEip.IsEmulated()
	eip.ManagerId = extEip.GetManagerId()
	eip.CloudregionId = region.Id
	eip.ChargeType = extEip.GetInternetChargeType()

	err = manager.TableSpec().Insert(&eip)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &eip, projectId, extEip, eip.ManagerId)

	db.OpsLog.LogEvent(&eip, db.ACT_CREATE, eip.GetShortDesc(ctx), userCred)

	return &eip, nil
}

func (manager *SElasticipManager) getEipForInstance(instanceType string, instanceId string) (*SElasticip, error) {
	eip := SElasticip{}

	q := manager.Query()
	q = q.Equals("associate_type", instanceType)
	q = q.Equals("associate_id", instanceId)

	err := q.First(&eip)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("getEipForInstance query fail %s", err)
			return nil, err
		} else {
			return nil, nil
		}
	}

	eip.SetModelManager(manager)

	return &eip, nil
}

func (self *SElasticip) IsAssociated() bool {
	if len(self.AssociateId) == 0 {
		return false
	}
	if self.GetAssociateVM() != nil {
		return true
	}
	return false
}

func (self *SElasticip) GetAssociateVM() *SGuest {
	if self.AssociateType == "server" && len(self.AssociateId) > 0 {
		return GuestManager.FetchGuestById(self.AssociateId)
	}
	return nil
}

func (self *SElasticip) Dissociate(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(self.AssociateType) == 0 {
		return nil
	}
	vm := self.GetAssociateVM()
	if vm == nil {
		log.Errorf("dissociate VM not exists???")
	}
	_, err := db.Update(self, func() error {
		self.AssociateId = ""
		self.AssociateType = ""
		return nil
	})
	if err != nil {
		return err
	}
	if vm != nil {
		db.OpsLog.LogDetachEvent(ctx, vm, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, vm.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(vm, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}
	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		self.Delete(ctx, userCred)
	}
	return nil
}

func (self *SElasticip) AssociateVM(ctx context.Context, userCred mcclient.TokenCredential, vm *SGuest) error {
	if vm.PendingDeleted || vm.Deleted {
		return fmt.Errorf("vm is deleted")
	}
	if len(self.AssociateType) > 0 {
		return fmt.Errorf("EIP has been associated!!")
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = "server"
		self.AssociateId = vm.Id
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, vm, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, vm.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(vm, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (manager *SElasticipManager) getEipByExtEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, region *SCloudregion, projectId string) (*SElasticip, error) {
	eipObj, err := manager.FetchByExternalId(extEip.GetGlobalId())
	if err == nil {
		return eipObj.(*SElasticip), nil
	}
	if err != sql.ErrNoRows {
		log.Errorf("FetchByExternalId fail %s", err)
		return nil, err
	}

	return manager.newFromCloudEip(ctx, userCred, extEip, region, projectId)
}

func (manager *SElasticipManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionStr := jsonutils.GetAnyString(data, []string{"region", "region_id"})
	if len(regionStr) == 0 {
		return nil, httperrors.NewMissingParameterError("region_id")
	}
	region, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
		}
	}
	data.Add(jsonutils.NewString(region.GetId()), "cloudregion_id")

	managerStr := jsonutils.GetAnyString(data, []string{"manager", "manager_id"})
	if len(managerStr) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}

	provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("Cloud provider %s not found", managerStr)
		}
	}
	data.Add(jsonutils.NewString(provider.GetId()), "manager_id")

	chargeType := jsonutils.GetAnyString(data, []string{"charge_type"})
	if len(chargeType) == 0 {
		chargeType = api.EIP_CHARGE_TYPE_DEFAULT
	}

	if !utils.IsInStringArray(chargeType, []string{api.EIP_CHARGE_TYPE_BY_BANDWIDTH, api.EIP_CHARGE_TYPE_BY_TRAFFIC}) {
		return nil, httperrors.NewInputParameterError("charge type %s not supported", chargeType)
	}

	data.Add(jsonutils.NewString(chargeType), "charge_type")

	data, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
	if err != nil {
		return nil, err
	}

	//避免参数重名后还有pending.eip残留
	eipPendingUsage := &SQuota{Eip: 1}
	err = QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), eipPendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Out of eip quota: %s", err)
	}

	return data, nil
}

func (self *SElasticip) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	eipPendingUsage := &SQuota{Eip: 1}
	self.startEipAllocateTask(ctx, userCred, nil, eipPendingUsage)
}

func (self *SElasticip) startEipAllocateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipAllocateTask", self, userCred, params, "", "", pendingUsage)
	if err != nil {
		log.Errorf("newtask EipAllocateTask fail %s", err)
		return err
	}
	self.SetStatus(userCred, api.EIP_STATUS_ALLOCATE, "start allocate")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Elasticip delete do nothing")
	return nil
}

func (self *SElasticip) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SElasticip) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartEipDeallocateTask(ctx, userCred, "")
}

func (self *SElasticip) ValidateDeleteCondition(ctx context.Context) error {
	if self.IsAssociated() {
		return fmt.Errorf("eip is associated with instance")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SElasticip) StartEipDeallocateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipDeallocateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("newTask EipDeallocateTask fail %s", err)
		return err
	}
	self.SetStatus(userCred, api.EIP_STATUS_DEALLOCATE, "start to delete")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) AllowPerformAssociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "associate")
}

func (self *SElasticip) PerformAssociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.IsAssociated() {
		return nil, httperrors.NewConflictError("eip has been associated with instance")
	}

	if self.Status != api.EIP_STATUS_READY {
		return nil, httperrors.NewInvalidStatusError("eip cannot associate in status %s", self.Status)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	instanceId := jsonutils.GetAnyString(data, []string{"instance", "instance_id"})
	if len(instanceId) == 0 {
		return nil, httperrors.NewMissingParameterError("instance_id")
	}
	instanceType := jsonutils.GetAnyString(data, []string{"instance_type"})
	if len(instanceType) == 0 {
		instanceType = api.EIP_ASSOCIATE_TYPE_SERVER
	}

	if instanceType != api.EIP_ASSOCIATE_TYPE_SERVER {
		return nil, httperrors.NewInputParameterError("Unsupported %s", instanceType)
	}

	vmObj, err := GuestManager.FetchByIdOrName(userCred, instanceId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("server %s not found", instanceId)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	server := vmObj.(*SGuest)

	lockman.LockObject(ctx, server)
	defer lockman.ReleaseObject(ctx, server)

	if server.PendingDeleted {
		return nil, httperrors.NewInvalidStatusError("cannot associate pending delete server")
	}

	seip, _ := server.GetEip()
	if seip != nil {
		return nil, httperrors.NewInvalidStatusError("instance is already associated with eip")
	}

	if ok, _ := utils.InStringArray(server.Status, []string{api.VM_READY, api.VM_RUNNING}); !ok {
		return nil, httperrors.NewInvalidStatusError("cannot associate server in status %s", server.Status)
	}

	serverRegion := server.getRegion()
	if serverRegion == nil {
		return nil, httperrors.NewInputParameterError("server region is not found???")
	}

	eipRegion := self.GetRegion()
	if eipRegion == nil {
		return nil, httperrors.NewInputParameterError("eip region is not found???")
	}

	if serverRegion.Id != eipRegion.Id {
		return nil, httperrors.NewInputParameterError("eip and server are not in the same region")
	}

	srvHost := server.GetHost()
	if srvHost == nil {
		return nil, httperrors.NewInputParameterError("server host is not found???")
	}

	if srvHost.ManagerId != self.ManagerId {
		return nil, httperrors.NewInputParameterError("server and eip are not managed by the same provider")
	}

	err = self.StartEipAssociateInstanceTask(ctx, userCred, server, "")
	return nil, err
}

func (self *SElasticip) StartEipAssociateInstanceTask(ctx context.Context, userCred mcclient.TokenCredential, server *SGuest, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(server.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(server.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_SERVER), "instance_type")

	return self.StartEipAssociateTask(ctx, userCred, params, parentTaskId)
}

func (self *SElasticip) StartEipAssociateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipAssociateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create EipAssociateTask task fail %s", err)
		return err
	}
	self.SetStatus(userCred, api.EIP_STATUS_ASSOCIATE, "start to associate")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) AllowPerformDissociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "dissociate")
}

func (self *SElasticip) PerformDissociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.AssociateId) == 0 {
		return nil, nil // success
	}

	// associate with an invalid vm
	if !self.IsAssociated() {
		return nil, self.Dissociate(ctx, userCred)
	}

	if self.Status != api.EIP_STATUS_READY {
		return nil, httperrors.NewInvalidStatusError("eip cannot dissociate in status %s", self.Status)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed public eip cannot be dissociated")
	}

	autoDelete := jsonutils.QueryBoolean(data, "auto_delete", false)

	err := self.StartEipDissociateTask(ctx, userCred, autoDelete, "")
	return nil, err
}

func (self *SElasticip) StartEipDissociateTask(ctx context.Context, userCred mcclient.TokenCredential, autoDelete bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	if autoDelete {
		params.Add(jsonutils.JSONTrue, "auto_delete")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "EipDissociateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create EipDissociateTask fail %s", err)
		return nil
	}
	self.SetStatus(userCred, api.EIP_STATUS_DISSOCIATE, "start to dissociate")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, err
	}

	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("fail to find region for eip")
	}

	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SElasticip) GetIEip() (cloudprovider.ICloudEIP, error) {
	iregion, err := self.GetIRegion()
	if err != nil {
		return nil, err
	}
	return iregion.GetIEipById(self.GetExternalId())
}

func (self *SElasticip) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync")
}

func (self *SElasticip) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	/*if self.Status != EIP_STATUS_READY && !strings.HasSuffix(self.Status, "_fail") {
		return nil, httperrors.NewInvalidStatusError("eip cannot syncstatus in status %s", self.Status)
	}*/

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot sync status")
	}

	err := self.StartEipSyncstatusTask(ctx, userCred, "")
	return nil, err
}

func (self *SElasticip) StartEipSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipSyncstatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create EipSyncstatusTask fail %s", err)
		return err
	}
	self.SetStatus(userCred, "sync", "synchronize")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SElasticip) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SElasticip) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))
	vm := self.GetAssociateVM()
	if vm != nil {
		extra.Add(jsonutils.NewString(vm.GetName()), "associate_name")
	}
	return extra
}

func (manager *SElasticipManager) AllocateEipAndAssociateVM(ctx context.Context, userCred mcclient.TokenCredential, vm *SGuest, bw int, chargeType string, eipPendingUsage quotas.IQuota) error {

	host := vm.GetHost()
	region := host.GetRegion()

	if len(chargeType) == 0 {
		chargeType = api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}

	eip := SElasticip{}
	eip.SetModelManager(manager)

	eip.Mode = api.EIP_MODE_STANDALONE_EIP
	// do not implicitly auto dellocate EIP, should be set by user explicitly
	// eip.AutoDellocate = tristate.True
	eip.Bandwidth = bw
	eip.ChargeType = chargeType
	eip.ProjectId = vm.ProjectId
	eip.ManagerId = host.ManagerId
	eip.CloudregionId = region.Id
	eip.Name = fmt.Sprintf("eip-for-%s", vm.GetName())

	err := manager.TableSpec().Insert(&eip)
	if err != nil {
		log.Errorf("create EIP record fail %s", err)
		return err
	}

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(vm.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(vm.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_SERVER), "instance_type")

	vm.SetStatus(userCred, api.VM_ASSOCIATE_EIP, "allocate and associate EIP")

	return eip.startEipAllocateTask(ctx, userCred, params, eipPendingUsage)
}

func (self *SElasticip) AllowPerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-bandwidth")
}

func (self *SElasticip) PerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.EIP_STATUS_READY {
		return nil, httperrors.NewInvalidStatusError("cannot change bandwidth in status %s", self.Status)
	}

	bandwidth, err := data.Int("bandwidth")
	if err != nil || bandwidth <= 0 {
		return nil, httperrors.NewInputParameterError("Invalid bandwidth")
	}

	factory, err := self.GetProviderFactory()
	if err != nil {
		return nil, err
	}

	if err := factory.ValidateChangeBandwidth(self.AssociateId, bandwidth); err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	err = self.StartEipChangeBandwidthTask(ctx, userCred, bandwidth)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

func (self *SElasticip) StartEipChangeBandwidthTask(ctx context.Context, userCred mcclient.TokenCredential, bandwidth int64) error {

	self.SetStatus(userCred, api.EIP_STATUS_CHANGE_BANDWIDTH, "change bandwidth")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(bandwidth), "bandwidth")

	task, err := taskman.TaskManager.NewTask(ctx, "EipChangeBandwidthTask", self, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("create EipChangeBandwidthTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) DoChangeBandwidth(userCred mcclient.TokenCredential, bandwidth int) error {
	changes := jsonutils.NewDict()
	changes.Add(jsonutils.NewInt(int64(self.Bandwidth)), "obw")

	_, err := db.Update(self, func() error {
		self.Bandwidth = bandwidth
		return nil
	})

	self.SetStatus(userCred, api.EIP_STATUS_READY, "finish change bandwidth")

	if err != nil {
		log.Errorf("DoChangeBandwidth update fail %s", err)
		return err
	}

	changes.Add(jsonutils.NewInt(int64(bandwidth)), "nbw")
	db.OpsLog.LogEvent(self, db.ACT_CHANGE_BANDWIDTH, changes, userCred)

	return nil
}

type EipUsage struct {
	PublicIPCount int
	EIPCount      int
	EIPUsedCount  int
}

func (u EipUsage) Total() int {
	return u.PublicIPCount + u.EIPCount
}

func (manager *SElasticipManager) usageQByProvider(q *sqlchemy.SQuery, providers []string) *sqlchemy.SQuery {
	if len(providers) == 0 {
		return q
	}
	cloudproviders := CloudproviderManager.Query().SubQuery()
	subq := cloudproviders.Query(cloudproviders.Field("id")).In("provider", providers).SubQuery()
	q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	return q
}

func (manager *SElasticipManager) usageQByRange(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
	if rangeObj == nil {
		return q
	}

	kw := rangeObj.Keyword()
	// log.Debugf("rangeObj keyword: %s", kw)
	switch kw {
	case "zone":
		zone := rangeObj.(*SZone)
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "wire":
		wire := rangeObj.(*SWire)
		zone := wire.GetZone()
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "host":
		host := rangeObj.(*SHost)
		zone := host.GetZone()
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "cloudprovider":
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), rangeObj.GetId()))
	case "cloudaccount":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id")).Equals("cloudaccount_id", rangeObj.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	case "cloudregion":
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), rangeObj.GetId()))
	}

	return q
}

func (manager *SElasticipManager) usageQ(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, providers []string) *sqlchemy.SQuery {
	q = manager.usageQByRange(q, rangeObj)
	q = manager.usageQByProvider(q, providers)
	return q
}

func (manager *SElasticipManager) TotalCount(projectId string, rangeObj db.IStandaloneModel, providers []string) EipUsage {
	usage := EipUsage{}
	q1 := manager.Query().Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP)
	q1 = manager.usageQ(q1, rangeObj, providers)
	q2 := manager.Query().Equals("mode", api.EIP_MODE_STANDALONE_EIP)
	q2 = manager.usageQ(q2, rangeObj, providers)
	q3 := manager.Query().Equals("mode", api.EIP_MODE_STANDALONE_EIP).IsNotEmpty("associate_id")
	q3 = manager.usageQ(q3, rangeObj, providers)
	if len(projectId) > 0 {
		q1 = q1.Equals("tenant_id", projectId)
		q2 = q2.Equals("tenant_id", projectId)
		q3 = q3.Equals("tenant_id", projectId)
	}
	usage.PublicIPCount, _ = q1.CountWithError()
	usage.EIPCount, _ = q2.CountWithError()
	usage.EIPUsedCount, _ = q3.CountWithError()
	return usage
}

func (self *SElasticip) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SElasticip) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	provider := self.GetCloudprovider()
	if provider != nil {
		if provider.Enabled {
			return nil, httperrors.NewInvalidStatusError("Cannot purge elastic_ip on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (self *SElasticip) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		self.SVirtualResourceBase.DoPendingDelete(ctx, userCred)
		return
	}
	self.Dissociate(ctx, userCred)
}

func (self *SElasticip) getCloudProviderInfo() SCloudProviderInfo {
	region := self.GetRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}
