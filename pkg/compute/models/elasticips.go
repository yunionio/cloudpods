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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	ElasticipManager.SetVirtualObject(ElasticipManager)
	ElasticipManager.TableSpec().AddIndex(true, "associate_id", "associate_type")
}

type SElasticip struct {
	db.SVirtualResourceBase

	db.SExternalizedResourceBase

	SManagedResourceBase
	SBillingResourceBase

	NetworkId string `width:"36" charset:"ascii" nullable:"true" get:"user" list:"user" create:"optional"`
	Mode      string `width:"32" charset:"ascii" list:"user"`

	IpAddr string `width:"17" charset:"ascii" list:"user" create:"optional"`

	AssociateType string `width:"32" charset:"ascii" list:"user"`
	AssociateId   string `width:"256" charset:"ascii" list:"user"`

	Bandwidth int `list:"user" create:"optional" default:"0"`

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

	associateType, _ := query.GetString("usable_eip_for_associate_type")
	associateId, _ := query.GetString("usable_eip_for_associate_id")
	if len(associateType) > 0 && len(associateId) > 0 {
		switch associateType {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			serverObj, err := GuestManager.FetchByIdOrName(userCred, associateId)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("server %s not found", regionFilter)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			guest := serverObj.(*SGuest)
			if utils.IsInStringArray(guest.Hypervisor, api.PRIVATE_CLOUD_HYPERVISORS) {
				zone := guest.getZone()
				networks := NetworkManager.Query().SubQuery()
				wires := WireManager.Query().SubQuery()

				sq := networks.Query(networks.Field("id")).Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).
					Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id)).SubQuery()
				q = q.Filter(sqlchemy.In(q.Field("network_id"), sq))
			} else {
				region := guest.getRegion()
				q = q.Equals("cloudregion_id", region.Id)
			}
			managerId := guest.GetHost().ManagerId
			q = q.Equals("manager_id", managerId)
		default:
			return nil, httperrors.NewInputParameterError("Not support associate type %s, only support %s", associateType, api.EIP_ASSOCIATE_VALID_TYPES)
		}
	}

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

func (self *SElasticip) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return network.(*SNetwork), nil
}

func (self *SElasticip) GetZone() *SZone {
	if len(self.NetworkId) == 0 {
		return nil
	}
	network, err := self.GetNetwork()
	if err != nil {
		return nil
	}
	return network.getZone()
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

func (manager *SElasticipManager) SyncEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, eips []cloudprovider.ICloudEIP, syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {
	// ownerProjId := projectId

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

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
		err = commondb[i].SyncWithCloudEip(ctx, userCred, provider, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudEip(ctx, userCred, added[i], provider, region, syncOwnerId)
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

	return self.RealDelete(ctx, userCred)
}

func (self *SElasticip) SyncInstanceWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudEIP) error {
	resource := self.GetAssociateResource()
	vmExtId := ext.GetAssociationExternalId()

	if resource == nil && len(vmExtId) == 0 {
		return nil
	}
	if resource != nil && resource.(db.IExternalizedModel).GetExternalId() == vmExtId {
		return nil
	}

	if resource != nil { // dissociate
		err := self.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("fail to dissociate vm: %s", err)
			return err
		}
	}

	if len(vmExtId) > 0 {
		var manager db.IModelManager
		switch ext.GetAssociationType() {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			manager = GuestManager
		case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
			manager = NatGatewayManager
		case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
			manager = LoadbalancerManager
		default:
			return errors.Error("unsupported association type")
		}

		extRes, err := db.FetchByExternalId(manager, vmExtId)
		if err != nil {
			log.Errorf("fail to find vm by external ID %s", vmExtId)
			return err
		}
		switch newRes := extRes.(type) {
		case *SGuest:
			err = self.AssociateVM(ctx, userCred, newRes)
		case *SLoadbalancer:
			err = self.AssociateLoadbalancer(ctx, userCred, newRes)
		case *SNatGateway:
			err = self.AssociateNatGateway(ctx, userCred, newRes)
		default:
			return errors.Error("unsupported association type")
		}
		if err != nil {
			log.Errorf("fail to associate with new vm %s", err)
			return err
		}
	}

	return nil
}

func (self *SElasticip) SyncWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudEIP, syncOwnerId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {

		// self.Name = ext.GetName()
		self.Bandwidth = ext.GetBandwidth()
		self.IpAddr = ext.GetIpAddr()
		self.Mode = ext.GetMode()
		self.Status = ext.GetStatus()
		self.ExternalId = ext.GetGlobalId()
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

	err = self.SyncInstanceWithCloudEip(ctx, userCred, ext)
	if err != nil {
		return errors.Wrap(err, "fail to sync associated instance of EIP")
	}
	SyncCloudProject(userCred, self, syncOwnerId, ext, self.ManagerId)

	return nil
}

func (manager *SElasticipManager) newFromCloudEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, provider *SCloudprovider, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SElasticip, error) {
	eip := SElasticip{}
	eip.SetModelManager(manager, &eip)

	newName, err := db.GenerateName(manager, syncOwnerId, extEip.GetName())
	if err != nil {
		return nil, err
	}
	eip.Name = newName
	eip.Status = extEip.GetStatus()
	eip.ExternalId = extEip.GetGlobalId()
	eip.IpAddr = extEip.GetIpAddr()
	eip.Mode = extEip.GetMode()
	eip.IsEmulated = extEip.IsEmulated()
	eip.ManagerId = provider.Id
	eip.CloudregionId = region.Id
	eip.ChargeType = extEip.GetInternetChargeType()
	if networkId := extEip.GetINetworkId(); len(networkId) > 0 {
		network, err := db.FetchByExternalId(NetworkManager, networkId)
		if err != nil {
			msg := fmt.Sprintf("failed to found network by externalId %s error: %v", networkId, err)
			log.Errorf(msg)
			return nil, errors.Error(msg)
		}
		eip.NetworkId = network.GetId()
	}

	err = manager.TableSpec().Insert(&eip)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &eip, syncOwnerId, extEip, eip.ManagerId)

	err = eip.SyncInstanceWithCloudEip(ctx, userCred, extEip)
	if err != nil {
		return nil, errors.Wrap(err, "fail to sync associated instance of EIP")
	}

	db.OpsLog.LogEvent(&eip, db.ACT_CREATE, eip.GetShortDesc(ctx), userCred)

	return &eip, nil
}

func (manager *SElasticipManager) getEipForInstance(instanceType string, instanceId string) (*SElasticip, error) {
	return manager.getEip(instanceType, instanceId, api.EIP_MODE_STANDALONE_EIP)
}

func (manager *SElasticipManager) getEip(instanceType string, instanceId string, eipMode string) (*SElasticip, error) {
	eip := SElasticip{}

	q := manager.Query()
	q = q.Equals("associate_type", instanceType)
	q = q.Equals("associate_id", instanceId)
	if len(eipMode) > 0 {
		q = q.Equals("mode", eipMode)
	}

	err := q.First(&eip)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("getEipForInstance query fail %s", err)
			return nil, err
		} else {
			return nil, nil
		}
	}

	eip.SetModelManager(manager, &eip)

	return &eip, nil
}

func (self *SElasticip) IsAssociated() bool {
	if len(self.AssociateId) == 0 {
		return false
	}
	if self.GetAssociateVM() != nil {
		return true
	}
	if self.GetAssociateLoadbalancer() != nil {
		return true
	}
	if self.GetAssociateNatGateway() != nil {
		return true
	}
	return false
}

func (self *SElasticip) GetAssociateVM() *SGuest {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_SERVER && len(self.AssociateId) > 0 {
		return GuestManager.FetchGuestById(self.AssociateId)
	}
	return nil
}

func (self *SElasticip) GetAssociateLoadbalancer() *SLoadbalancer {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_LOADBALANCER && len(self.AssociateId) > 0 {
		_lb, err := LoadbalancerManager.FetchById(self.AssociateId)
		if err != nil {
			return nil
		}
		lb := _lb.(*SLoadbalancer)
		if lb.PendingDeleted {
			return nil
		}
		return lb
	}
	return nil
}

func (self *SElasticip) GetAssociateNatGateway() *SNatGateway {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY && len(self.AssociateId) > 0 {
		natGateway, err := NatGatewayManager.FetchById(self.AssociateId)
		if err != nil {
			return nil
		}
		return natGateway.(*SNatGateway)
	}
	return nil
}

func (self *SElasticip) GetAssociateResource() db.IModel {
	if vm := self.GetAssociateVM(); vm != nil {
		return vm
	}
	if lb := self.GetAssociateLoadbalancer(); lb != nil {
		return lb
	}
	if nat := self.GetAssociateNatGateway(); nat != nil {
		return nat
	}
	return nil
}

func (self *SElasticip) Dissociate(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(self.AssociateType) == 0 {
		return nil
	}
	var vm *SGuest
	var nat *SNatGateway
	var lb *SLoadbalancer
	switch self.AssociateType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		vm = self.GetAssociateVM()
		if vm == nil {
			log.Errorf("dissociate VM not exists???")
		}
	case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
		nat = self.GetAssociateNatGateway()
		if nat == nil {
			log.Errorf("dissociate Nat gateway not exists???")
		}
	case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
		lb = self.GetAssociateLoadbalancer()
		if lb == nil {
			log.Errorf("dissociate loadbalancer not exists???")
		}
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

	if nat != nil {
		db.OpsLog.LogDetachEvent(ctx, nat, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, nat.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(nat, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if lb != nil {
		db.OpsLog.LogDetachEvent(ctx, lb, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, lb.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(lb, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		self.RealDelete(ctx, userCred)
	}
	return nil
}

func (self *SElasticip) AssociateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer) error {
	if lb.PendingDeleted {
		return fmt.Errorf("loadbalancer is deleted")
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == api.EIP_ASSOCIATE_TYPE_LOADBALANCER && self.AssociateId == lb.Id {
			return nil
		} else {
			return fmt.Errorf("EIP has been associated!!")
		}
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = api.EIP_ASSOCIATE_TYPE_LOADBALANCER
		self.AssociateId = lb.Id
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, lb, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, lb.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(lb, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (self *SElasticip) AssociateVM(ctx context.Context, userCred mcclient.TokenCredential, vm *SGuest) error {
	if vm.PendingDeleted || vm.Deleted {
		return fmt.Errorf("vm is deleted")
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == api.EIP_ASSOCIATE_TYPE_SERVER && self.AssociateId == vm.Id {
			return nil
		} else {
			return fmt.Errorf("EIP has been associated!!")
		}
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = api.EIP_ASSOCIATE_TYPE_SERVER
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

func (self *SElasticip) AssociateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, nat *SNatGateway) error {
	if nat.Deleted {
		return fmt.Errorf("nat gateway is deleted")
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY && self.AssociateId == nat.Id {
			return nil
		} else {
			return fmt.Errorf("Eip has been associated!!")
		}
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
		self.AssociateId = nat.Id
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, nat, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, nat.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(nat, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (manager *SElasticipManager) getEipByExtEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, provider *SCloudprovider, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SElasticip, error) {
	eipObj, err := db.FetchByExternalId(manager, extEip.GetGlobalId())
	if err == nil {
		return eipObj.(*SElasticip), nil
	}
	if err != sql.ErrNoRows {
		log.Errorf("FetchByExternalId fail %s", err)
		return nil, err
	}

	return manager.newFromCloudEip(ctx, userCred, extEip, provider, region, syncOwnerId)
}

func (manager *SElasticipManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionStr := jsonutils.GetAnyString(data, []string{"region", "region_id"})
	if len(regionStr) == 0 {
		return nil, httperrors.NewMissingParameterError("region_id")
	}
	_region, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
		}
	}
	region := _region.(*SCloudregion)
	data.Add(jsonutils.NewString(region.GetId()), "cloudregion_id")

	managerStr := jsonutils.GetAnyString(data, []string{"manager", "manager_id"})
	if len(managerStr) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}

	providerObj, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("Cloud provider %s not found", managerStr)
		}
	}
	provider := providerObj.(*SCloudprovider)
	data.Add(jsonutils.NewString(provider.Id), "manager_id")

	chargeType := jsonutils.GetAnyString(data, []string{"charge_type"})
	if len(chargeType) == 0 {
		chargeType = api.EIP_CHARGE_TYPE_DEFAULT
	}

	if !utils.IsInStringArray(chargeType, []string{api.EIP_CHARGE_TYPE_BY_BANDWIDTH, api.EIP_CHARGE_TYPE_BY_TRAFFIC}) {
		return nil, httperrors.NewInputParameterError("charge type %s not supported", chargeType)
	}

	data.Add(jsonutils.NewString(chargeType), "charge_type")

	input := apis.VirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	//避免参数重名后还有pending.eip残留
	eipPendingUsage := &SRegionQuota{Eip: 1}
	quotaKeys := fetchRegionalQuotaKeys(rbacutils.ScopeProject, ownerId, region, provider)
	eipPendingUsage.SetKeys(quotaKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, eipPendingUsage)
	if err != nil {
		return nil, err
	}

	return region.GetDriver().ValidateCreateEipData(ctx, userCred, data)
}

func (eip *SElasticip) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	region := eip.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid region")
	}
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		eip.GetOwnerId(),
		region,
		eip.GetCloudprovider(),
	), nil
}

func (self *SElasticip) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys, err := self.GetQuotaKeys()
	if err != nil {
		log.Errorf("GetQuotaKeys fail %s", err)
	} else {
		eipPendingUsage.SetKeys(keys)
		err := quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage)
		if err != nil {
			log.Errorf("SElasticip CancelPendingUsage error: %s", err)
		}
	}

	self.startEipAllocateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SElasticip) startEipAllocateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipAllocateTask", self, userCred, params, parentTaskId, "", nil)
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
		return fmt.Errorf("eip is associated with resources")
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

	eipZone := self.GetZone()
	if eipZone != nil {
		serverZone := server.getZone()
		if serverZone.Id != eipZone.Id {
			return nil, httperrors.NewInputParameterError("eip and server are not in the same zone")
		}
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

	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY {
		model, err := NatGatewayManager.FetchById(self.AssociateId)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch natgateway %s", self.AssociateId)
		}
		natgateway := model.(*SNatGateway)
		sCount, err := natgateway.GetSTableSize(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("ip", self.IpAddr)
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fail to get stable size of natgateway %s", self.AssociateId)
		}
		if sCount > 0 {
			return nil, httperrors.NewUnsupportOperationError(
				"the associated natgateway has corresponding snat rules with eip %s, please delete them firstly", self.IpAddr)
		}
		dCount, err := natgateway.GetDTableSize(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("external_ip", self.IpAddr)
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fail to get dtable size of natgateway %s", self.AssociateId)
		}
		if dCount > 0 {
			return nil, httperrors.NewUnsupportOperationError(
				"the associated natgateway has corresponding dnat rules with eip %s, please delete them firstly", self.IpAddr)
		}
	}

	autoDelete := jsonutils.QueryBoolean(data, "auto_delete", false)
	switch self.AssociateType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		guest := self.GetAssociateVM()
		if guest == nil {
			return nil, httperrors.NewInputParameterError("unable to found guest for elasticip %s(%s)", self.Name, self.IpAddr)
		}
		return nil, guest.StartGuestDissociateEipTask(ctx, userCred, self, autoDelete, "")
	default:
		return nil, self.StartEipDissociateTask(ctx, userCred, autoDelete, "")
	}
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
	instance := self.GetAssociateResource()
	if instance != nil {
		extra.Add(jsonutils.NewString(instance.GetName()), "associate_name")
	}
	return extra
}

func (manager *SElasticipManager) NewEipForVMOnHost(ctx context.Context, userCred mcclient.TokenCredential, vm *SGuest, host *SHost, bw int, chargeType string, pendingUsage quotas.IQuota) (*SElasticip, error) {
	region := host.GetRegion()

	if len(chargeType) == 0 {
		chargeType = api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}

	eip := SElasticip{}
	eip.SetModelManager(manager, &eip)

	eip.Mode = api.EIP_MODE_STANDALONE_EIP
	// do not implicitly auto dellocate EIP, should be set by user explicitly
	// eip.AutoDellocate = tristate.True
	eip.Bandwidth = bw
	eip.ChargeType = chargeType
	eip.DomainId = vm.DomainId
	eip.ProjectId = vm.ProjectId
	eip.ProjectSrc = string(db.PROJECT_SOURCE_LOCAL)
	eip.ManagerId = host.ManagerId
	eip.CloudregionId = region.Id
	eip.Name = fmt.Sprintf("eip-for-%s", vm.GetName())

	err := manager.TableSpec().Insert(&eip)
	if err != nil {
		log.Errorf("create EIP record fail %s", err)
		return nil, err
	}

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys := fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		vm.GetOwnerId(),
		region,
		host.GetCloudprovider(),
	)
	eipPendingUsage.SetKeys(keys)
	quotas.CancelPendingUsage(ctx, userCred, pendingUsage, eipPendingUsage)

	return &eip, nil
}

func (eip *SElasticip) AllocateAndAssociateVM(ctx context.Context, userCred mcclient.TokenCredential, vm *SGuest, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(vm.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(vm.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_SERVER), "instance_type")

	vm.SetStatus(userCred, api.VM_ASSOCIATE_EIP, "allocate and associate EIP")

	return eip.startEipAllocateTask(ctx, userCred, params, parentTaskId)
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

func (manager *SElasticipManager) usageQByCloudEnv(q *sqlchemy.SQuery, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	return CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
}

func (manager *SElasticipManager) usageQByRanges(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel) *sqlchemy.SQuery {
	return rangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"))
}

func (manager *SElasticipManager) usageQ(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	q = manager.usageQByRanges(q, rangeObjs)
	q = manager.usageQByCloudEnv(q, providers, brands, cloudEnv)
	return q
}

func (manager *SElasticipManager) TotalCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) EipUsage {
	usage := EipUsage{}
	q1 := manager.Query().Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP)
	q1 = manager.usageQ(q1, rangeObjs, providers, brands, cloudEnv)
	q2 := manager.Query().Equals("mode", api.EIP_MODE_STANDALONE_EIP)
	q2 = manager.usageQ(q2, rangeObjs, providers, brands, cloudEnv)
	q3 := manager.Query().Equals("mode", api.EIP_MODE_STANDALONE_EIP).IsNotEmpty("associate_id")
	q3 = manager.usageQ(q3, rangeObjs, providers, brands, cloudEnv)
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q1 = q1.Equals("domain_id", ownerId.GetProjectDomainId())
		q2 = q2.Equals("domain_id", ownerId.GetProjectDomainId())
		q3 = q3.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q1 = q1.Equals("tenant_id", ownerId.GetProjectId())
		q2 = q2.Equals("tenant_id", ownerId.GetProjectId())
		q3 = q3.Equals("tenant_id", ownerId.GetProjectId())
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

func (eip *SElasticip) GetUsages() []db.IUsage {
	if eip.PendingDeleted || eip.Deleted {
		return nil
	}
	usage := SRegionQuota{Eip: 1}
	keys, err := eip.GetQuotaKeys()
	if err != nil {
		log.Errorf("disk.GetQuotaKeys fail %s", err)
		return nil
	}
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (manager *SElasticipManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "account":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudaccounts := CloudaccountManager.Query("name", "id").Distinct().SubQuery()
		q = q.Join(cloudproviders, sqlchemy.Equals(q.Field("manager_id"), cloudproviders.Field("id")))
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), cloudaccounts.Field("id")))
		q.GroupBy(cloudaccounts.Field("name"))
		q.AppendField(cloudaccounts.Field("name", "account"))
	default:
		return q, httperrors.NewBadRequestError("unsupport field %s", field)
	}
	return q, nil
}
