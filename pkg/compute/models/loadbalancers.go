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
	"net"
	"strings"

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

type SLoadbalancerManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var LoadbalancerManager *SLoadbalancerManager

func init() {
	LoadbalancerManager = &SLoadbalancerManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancer{},
			"loadbalancers_tbl",
			"loadbalancer",
			"loadbalancers",
		),
	}
	LoadbalancerManager.SetVirtualObject(LoadbalancerManager)
}

// TODO build errors on pkg/httperrors/errors.go
// NewGetManagerError
// NewMissingArgumentError
// NewInvalidArgumentError
//
// TODO ZoneId or RegionId
// bandwidth
// scheduler
//
// TODO update backendgroupid
type SLoadbalancer struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase
	SZoneResourceBase
	SLoadbalancerRateLimiter

	Address     string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	AddressType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkId   string `width:"147" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	VpcId       string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	ClusterId   string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	ChargeType string `list:"user" get:"user" create:"optional" update:"user"`

	LoadbalancerSpec string `list:"user" get:"user" list:"user" create:"optional"`

	BackendGroupId string               `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user"`
	LBInfo         jsonutils.JSONObject `charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional"`
}

func (man *SLoadbalancerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	ownerId := userCred
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "network", ModelKeyword: "network", OwnerId: ownerId},
		{Key: "zone", ModelKeyword: "zone", OwnerId: ownerId},
		{Key: "manager", ModelKeyword: "cloudprovider", OwnerId: ownerId},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: ownerId},
		{Key: "cluster", ModelKeyword: "loadbalancercluster", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var region *SCloudregion
	if id, _ := data.GetString("vpc"); len(id) > 0 {
		vpc, err := db.FetchByIdOrName(VpcManager, userCred, id)
		if err != nil {
			return nil, fmt.Errorf("getting vpc failed")
		}

		region, _ = vpc.(*SVpc).GetRegion()
	} else if id, _ := data.GetString("zone"); len(id) > 0 {
		zone, err := db.FetchByIdOrName(ZoneManager, userCred, id)
		if err != nil {
			return nil, fmt.Errorf("getting zone failed")
		}

		region = zone.(*SZone).GetRegion()
	} else if id, _ := data.GetString("network"); len(id) > 0 {
		network, err := db.FetchByIdOrName(NetworkManager, userCred, strings.Split(id, ",")[0])
		if err != nil {
			return nil, fmt.Errorf("getting network failed")
		}
		region = network.(*SNetwork).getRegion()
	}

	if region == nil {
		return nil, fmt.Errorf("getting region failed")
	}

	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, "ownerId", ownerId)
	return region.GetDriver().ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (lb *SLoadbalancer) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lb, "status")
}

func (lb *SLoadbalancer) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if _, err := lb.SVirtualResourceBase.PerformStatus(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	if lb.Status == api.LB_STATUS_ENABLED {
		return nil, lb.StartLoadBalancerStartTask(ctx, userCred, "")
	}
	return nil, lb.StartLoadBalancerStopTask(ctx, userCred, "")
}

func (lb *SLoadbalancer) StartLoadBalancerStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerStartTask", lb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) StartLoadBalancerStopTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerStopTask", lb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lb, "syncstatus")
}

func (lb *SLoadbalancer) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lb.StartLoadBalancerSyncstatusTask(ctx, userCred, "")
}

func (lb *SLoadbalancer) StartLoadBalancerSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(lb.Status), "origin_status")
	lb.SetStatus(userCred, api.LB_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerSyncstatusTask", lb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// NOTE lb.Id will only be available after BeforeInsert happens
	// NOTE this means lb.UpdateVersion will be 0, then 1 after creation
	// NOTE need ways to notify error

	lb.SetStatus(userCred, api.LB_CREATING, "")
	if err := lb.StartLoadBalancerCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create loadbalancer error: %v", err)
	}
}

func (lb *SLoadbalancer) GetCloudprovider() *SCloudprovider {
	cloudprovider, err := CloudproviderManager.FetchById(lb.ManagerId)
	if err != nil {
		return nil
	}
	return cloudprovider.(*SCloudprovider)
}

func (lb *SLoadbalancer) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(lb.CloudregionId)
	if err != nil {
		log.Errorf("failed to find region for loadbalancer %s", lb.Name)
		return nil
	}
	return region.(*SCloudregion)
}

func (lb *SLoadbalancer) GetZone() *SZone {
	zone, err := ZoneManager.FetchById(lb.ZoneId)
	if err != nil {
		return nil
	}
	return zone.(*SZone)
}

func (lb *SLoadbalancer) GetVpc() *SVpc {
	vpc, err := VpcManager.FetchById(lb.VpcId)
	if err != nil {
		return nil
	}
	return vpc.(*SVpc)
}

func (lb *SLoadbalancer) GetNetworks() ([]SNetwork, error) {
	networks := []SNetwork{}
	networkIds := strings.Split(lb.NetworkId, ",")
	err := NetworkManager.Query().In("id", networkIds).IsFalse("pending_deleted").All(&networks)
	if err != nil {
		return nil, err
	}

	if len(networks) == 0 {
		return nil, fmt.Errorf("loadbalancer has no releated network found")
	}

	if len(networks) != len(networkIds) {
		return nil, fmt.Errorf("expected %d networks, %d found", len(networkIds), len(networks))
	}

	return networks, nil
}

func (lb *SLoadbalancer) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := lb.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for lb %s: %s", lb.Name, err)
	}
	region := lb.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for lb %s", lb.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (lb *SLoadbalancer) GetCreateLoadbalancerParams(iRegion cloudprovider.ICloudRegion) (*cloudprovider.SLoadbalancer, error) {
	params := &cloudprovider.SLoadbalancer{
		Name:             lb.Name,
		Address:          lb.Address,
		AddressType:      lb.AddressType,
		ChargeType:       lb.ChargeType,
		LoadbalancerSpec: lb.LoadbalancerSpec,
	}
	iRegion, err := lb.GetIRegion()
	if err != nil {
		return nil, err
	}
	if len(lb.ZoneId) > 0 {
		zone := lb.GetZone()
		if zone == nil {
			return nil, fmt.Errorf("failed to find zone for lb %s", lb.Name)
		}
		iZone, err := iRegion.GetIZoneById(zone.ExternalId)
		if err != nil {
			return nil, err
		}
		params.ZoneID = iZone.GetId()
	}
	if lb.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		params.EgressMbps = lb.EgressMbps
	}
	if lb.AddressType == api.LB_ADDR_TYPE_INTRANET || lb.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI || lb.GetProviderName() == api.CLOUD_PROVIDER_AWS {
		vpc := lb.GetVpc()
		if vpc == nil {
			return nil, fmt.Errorf("failed to find vpc for lb %s", lb.Name)
		}
		iVpc, err := iRegion.GetIVpcById(vpc.ExternalId)
		if err != nil {
			return nil, err
		}
		params.VpcID = iVpc.GetId()
		networks, err := lb.GetNetworks()
		if err != nil {
			return nil, fmt.Errorf("failed to find network for lb %s: %s", lb.Name, err)
		}

		for i := range networks {
			iNetwork, err := networks[i].GetINetwork()
			if err != nil {
				return nil, err
			}
			params.NetworkIDs = append(params.NetworkIDs, iNetwork.GetId())
		}

	}
	return params, nil
}

func (lb *SLoadbalancer) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lb, "purge")
}

func (lb *SLoadbalancer) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "purge")
	return nil, lb.StartLoadBalancerDeleteTask(ctx, userCred, params, "")
}

func (lb *SLoadbalancer) StartLoadBalancerDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerDeleteTask", lb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) StartLoadBalancerCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	taskData := jsonutils.NewDict()
	eipId, _ := data.GetString("eip_id")
	if len(eipId) > 0 {
		taskData.Set("eip_id", jsonutils.NewString(eipId)) // for huawei internet elb
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCreateTask", lb, userCred, taskData, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lb.GetOwnerId())
	backendGroupV.Optional(true)
	err := backendGroupV.Validate(data)
	if err != nil {
		return nil, err
	}
	if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.LoadbalancerId != lb.Id {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s, not %s",
			backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, lb.Id)
	}
	return lb.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lb *SLoadbalancer) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lb.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	providerInfo := lb.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if providerInfo != nil {
		extra.Update(providerInfo)
	}

	zoneInfo := lb.SZoneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if zoneInfo != nil {
		extra.Update(zoneInfo)
	} else {
		regionInfo := lb.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
		if regionInfo != nil {
			extra.Update(regionInfo)
		}
	}

	eip, _ := lb.GetEip()
	if eip != nil {
		extra.Set("eip", jsonutils.NewString(eip.IpAddr))
		extra.Set("eip_mode", jsonutils.NewString(eip.Mode))
	}

	if len(lb.VpcId) > 0 {
		if vpc := lb.GetVpc(); vpc != nil {
			extra.Set("vpc", jsonutils.NewString(vpc.Name))
		}
	}

	if lb.BackendGroupId != "" {
		lbbg, err := LoadbalancerBackendGroupManager.FetchById(lb.BackendGroupId)
		if err != nil {
			log.Errorf("loadbalancer %s(%s): fetch backend group (%s) error: %s",
				lb.Name, lb.Id, lb.BackendGroupId, err)
			return extra
		}
		extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))
	}
	return extra
}

func (lb *SLoadbalancer) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lb.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lb *SLoadbalancer) ValidateDeleteCondition(ctx context.Context) error {
	region := lb.GetRegion()
	if region != nil {
		if err := region.GetDriver().ValidateDeleteLoadbalancerCondition(ctx, lb); err != nil {
			return err
		}
	}

	return lb.SModelBase.ValidateDeleteCondition(ctx)
}

func (lb *SLoadbalancer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lb.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lb.StartLoadBalancerDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lb *SLoadbalancer) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	listeners := []SLoadbalancerListener{}
	q := LoadbalancerListenerManager.Query().Equals("loadbalancer_id", lb.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(LoadbalancerListenerManager, q, &listeners); err != nil {
		return nil, err
	}
	return listeners, nil
}

func (lb *SLoadbalancer) GetLoadbalancerBackendgroups() ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := LoadbalancerBackendGroupManager.Query().Equals("loadbalancer_id", lb.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(LoadbalancerBackendGroupManager, q, &lbbgs); err != nil {
		return nil, err
	}
	return lbbgs, nil
}

func (lb *SLoadbalancer) LBPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(lb.NetworkId) > 0 {
		req := &SLoadbalancerNetworkDeleteData{
			loadbalancer: lb,
		}
		err := LoadbalancernetworkManager.DeleteLoadbalancerNetwork(ctx, userCred, req)
		if err != nil {
			log.Errorf("failed detaching network of loadbalancer %s(%s): %v", lb.Name, lb.Id, err)
		}
	}
	lb.pendingDeleteSubs(ctx, userCred)
	lb.DoPendingDelete(ctx, userCred)
}

func (lb *SLoadbalancer) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	ownerId := lb.GetOwnerId()
	lbId := lb.Id
	subMen := []ILoadbalancerSubResourceManager{
		LoadbalancerListenerManager,
		LoadbalancerBackendGroupManager,
	}
	for _, subMan := range subMen {
		func(subMan ILoadbalancerSubResourceManager) {
			lockman.LockClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
			defer lockman.ReleaseClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
			q := subMan.Query().IsFalse("pending_deleted").Equals("loadbalancer_id", lbId)
			subMan.pendingDeleteSubs(ctx, userCred, q)
		}(subMan)
	}
}

func (lb *SLoadbalancer) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerManager) getLoadbalancersByRegion(region *SCloudregion, provider *SCloudprovider) ([]SLoadbalancer, error) {
	lbs := []SLoadbalancer{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbs); err != nil {
		log.Errorf("failed to get lbs for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return lbs, nil
}

func (man *SLoadbalancerManager) SyncLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbs []cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) ([]SLoadbalancer, []cloudprovider.ICloudLoadbalancer, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

	localLbs := []SLoadbalancer{}
	remoteLbs := []cloudprovider.ICloudLoadbalancer{}
	syncResult := compare.SyncResult{}

	dbLbs, err := man.getLoadbalancersByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancer{}
	commondb := []SLoadbalancer{}
	commonext := []cloudprovider.ICloudLoadbalancer{}
	added := []cloudprovider.ICloudLoadbalancer{}

	err = compare.CompareSets(dbLbs, lbs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancer(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancer(ctx, userCred, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localLbs = append(localLbs, commondb[i])
			remoteLbs = append(remoteLbs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancer(ctx, userCred, provider, added[i], region, syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localLbs = append(localLbs, *new)
			remoteLbs = append(remoteLbs, added[i])
			syncResult.Add()
		}
	}
	return localLbs, remoteLbs, syncResult
}

func (man *SLoadbalancerManager) newFromCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extLb cloudprovider.ICloudLoadbalancer, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancer, error) {
	lb := SLoadbalancer{}
	lb.SetModelManager(man, &lb)

	lb.ManagerId = provider.Id
	lb.CloudregionId = region.Id
	lb.Address = extLb.GetAddress()
	lb.AddressType = extLb.GetAddressType()
	lb.NetworkType = extLb.GetNetworkType()

	newName, err := db.GenerateName(man, syncOwnerId, extLb.GetName())
	if err != nil {
		return nil, err
	}
	lb.Name = newName
	lb.Status = extLb.GetStatus()
	lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
	lb.ChargeType = extLb.GetChargeType()
	lb.EgressMbps = extLb.GetEgressMbps()
	lb.ExternalId = extLb.GetGlobalId()
	networkIds := extLb.GetNetworkIds()
	lbNetworkIds := []string{}
	for _, networkId := range networkIds {
		if network, err := db.FetchByExternalId(NetworkManager, networkId); err == nil && network != nil {
			lbNetworkIds = append(lbNetworkIds, network.GetId())
		}
	}
	lb.NetworkId = strings.Join(lbNetworkIds, ",")

	if vpcId := extLb.GetVpcId(); len(vpcId) > 0 {
		if vpc, err := db.FetchByExternalId(VpcManager, vpcId); err == nil && vpc != nil {
			lb.VpcId = vpc.GetId()
		}
	}
	if zoneId := extLb.GetZoneId(); len(zoneId) > 0 {
		if zone, err := db.FetchByExternalId(ZoneManager, zoneId); err == nil && zone != nil {
			lb.ZoneId = zone.GetId()
		}
	}

	if extLb.GetMetadata() != nil {
		lb.LBInfo = extLb.GetMetadata()
	}

	if err := man.TableSpec().Insert(&lb); err != nil {
		log.Errorf("newFromCloudRegion fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &lb, syncOwnerId, extLb, lb.ManagerId)

	db.OpsLog.LogEvent(&lb, db.ACT_CREATE, lb.GetShortDesc(ctx), userCred)

	lb.syncLoadbalancerNetwork(ctx, userCred, networkIds)
	return &lb, nil
}

func (lb *SLoadbalancer) syncRemoveCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	err := lb.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return lb.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lb.LBPendingDelete(ctx, userCred)
		return nil
	}
}

func (lb *SLoadbalancer) syncLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential, networkIds []string) {
	if len(lb.NetworkId) > 0 {
		ip := ""
		if net.ParseIP(lb.Address) != nil {
			ip = lb.Address
		}

		for i := range networkIds {
			lbNetReq := &SLoadbalancerNetworkRequestData{
				Loadbalancer: lb,
				NetworkId:    networkIds[i],
				Address:      ip,
			}
			err := LoadbalancernetworkManager.syncLoadbalancerNetwork(ctx, userCred, lbNetReq)
			if err != nil {
				log.Errorf("failed to create loadbalancer network: %v", err)
			}
		}
	}
}

func (self *SLoadbalancer) DeleteEip(ctx context.Context, userCred mcclient.TokenCredential) error {
	eip, err := self.GetEip()
	if err != nil {
		log.Errorf("Delete eip fail for get Eip %s", err)
		return err
	}
	if eip == nil {
		return nil
	}
	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		err = eip.RealDelete(ctx, userCred)
		if err != nil {
			log.Errorf("Delete eip on delete server fail %s", err)
			return err
		}
	} else {
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("Dissociate eip on delete server fail %s", err)
			return err
		}
	}
	return nil
}

func (self *SLoadbalancer) GetEip() (*SElasticip, error) {
	return ElasticipManager.getEipForInstance(api.EIP_ASSOCIATE_TYPE_LOADBALANCER, self.Id)
}

func (self *SLoadbalancer) SyncLoadbalancerEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extEip cloudprovider.ICloudEIP) compare.SyncResult {
	result := compare.SyncResult{}

	eip, err := self.GetEip()
	if err != nil {
		result.Error(fmt.Errorf("getEip error %s", err))
		return result
	}

	if eip == nil && extEip == nil {
		// do nothing
	} else if eip == nil && extEip != nil {
		// add
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, self.GetRegion(), provider.GetOwnerId())
		if err != nil {
			log.Errorf("getEipByExtEip error %v", err)
			result.AddError(err)
		} else {
			err = neip.AssociateLoadbalancer(ctx, userCred, self)
			if err != nil {
				log.Errorf("AssociateVM error %v", err)
				result.AddError(err)
			} else {
				result.Add()
			}
		}
	} else if eip != nil && extEip == nil {
		// remove
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	} else {
		// sync
		if eip.IpAddr != extEip.GetIpAddr() {
			// remove then add
			err = eip.Dissociate(ctx, userCred)
			if err != nil {
				// fail to remove
				result.DeleteError(err)
			} else {
				result.Delete()
				neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, self.GetRegion(), provider.GetOwnerId())
				if err != nil {
					result.AddError(err)
				} else {
					err = neip.AssociateLoadbalancer(ctx, userCred, self)
					if err != nil {
						result.AddError(err)
					} else {
						result.Add()
					}
				}
			}
		} else {
			// do nothing
			err := eip.SyncWithCloudEip(ctx, userCred, provider, extEip, provider.GetOwnerId())
			if err != nil {
				result.UpdateError(err)
			} else {
				result.Update()
			}
		}
	}

	return result
}

func (lb *SLoadbalancer) SyncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, extLb cloudprovider.ICloudLoadbalancer, syncOwnerId mcclient.IIdentityProvider) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	diff, err := db.Update(lb, func() error {
		lb.Address = extLb.GetAddress()
		lb.Status = extLb.GetStatus()
		// lb.Name = extLb.GetName()
		lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
		lb.EgressMbps = extLb.GetEgressMbps()
		lb.ChargeType = extLb.GetChargeType()

		if extLb.GetMetadata() != nil {
			lb.LBInfo = extLb.GetMetadata()
		}

		return nil
	})

	db.OpsLog.LogSyncUpdate(lb, diff, userCred)

	SyncCloudProject(userCred, lb, syncOwnerId, extLb, lb.ManagerId)
	lb.syncLoadbalancerNetwork(ctx, userCred, extLb.GetNetworkIds())

	return err
}

func (lb *SLoadbalancer) setCloudregionId() error {
	zone := ZoneManager.FetchZoneById(lb.ZoneId)
	if zone == nil {
		return fmt.Errorf("failed to find zone %s", lb.ZoneId)
	}
	region := zone.GetRegion()
	if region == nil {
		return fmt.Errorf("failed to find region for zone: %s", lb.ZoneId)
	}
	_, err := db.Update(lb, func() error {
		lb.CloudregionId = region.Id
		return nil
	})
	return err
}

func (man *SLoadbalancerManager) InitializeData() error {
	lbs := []SLoadbalancer{}
	q := LoadbalancerManager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(LoadbalancerManager, q, &lbs); err != nil {
		log.Errorf("failed fetching lbs with empty cloudregion_id error: %v", err)
		return err
	}
	for i := 0; i < len(lbs); i++ {
		if err := lbs[i].setCloudregionId(); err != nil {
			log.Errorf("failed setting lb %s(%s) cloud region error: %v", lbs[i].Name, lbs[i].Id, err)
		}
	}
	return nil
}

func (manager *SLoadbalancerManager) GetResourceCount() ([]db.SProjectResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateProjectResourceCount(virts)
}

func (manager *SLoadbalancerManager) FetchByExternalId(providerId string, extId string) (*SLoadbalancer, error) {
	ret := []SLoadbalancer{}
	q := manager.Query().Equals("manager_id", providerId).Equals("external_id", extId)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, err
	}

	if len(ret) == 1 {
		return &ret[0], nil
	} else {
		return nil, fmt.Errorf("loadbalancerManager.FetchByExternalId provider %s external id %s %d found", providerId, extId, len(ret))
	}
}

func (manager *SLoadbalancerManager) GetLbDefaultBackendGroupIds() ([]string, error) {
	lbs := []SLoadbalancer{}
	err := manager.Query().IsFalse("pending_deleted").IsNotEmpty("backend_group_id").All(&lbs)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerManager.GetLbDefaultBackendGroupIds")
	}

	ret := []string{}
	for i := range lbs {
		ret = append(ret, lbs[i].BackendGroupId)
	}

	return ret, nil
}
