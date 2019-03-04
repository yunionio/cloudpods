package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerManager struct {
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
	SManagedResourceBase

	Address       string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	AddressType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	VpcId         string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	ZoneId        string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" default:"default" create:"optional"`

	ChargeType       string `list:"user" get:"user" create:"optional"`
	LoadbalancerSpec string `list:"user" get:"user" create:"optional"`

	BackendGroupId string               `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" update:"user"`
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
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "network", ModelKeyword: "network", ProjectId: userProjId},
		{Key: "zone", ModelKeyword: "zone", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	networkV := validators.NewModelIdOrNameValidator("network", "network", ownerProjId)
	addressType, _ := data.GetString("address_type")
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", "")
	managerIdV := validators.NewModelIdOrNameValidator("manager_id", "cloudprovider", "")
	if addressType == consts.LB_ADDR_TYPE_INTERNET {
		networkV.Optional(true)
	} else {
		zoneV.Optional(true)
		managerIdV.Optional(true)
	}
	addressV := validators.NewIPv4AddrValidator("address")
	addressTypeV := validators.NewStringChoicesValidator("address_type", consts.LB_ADDR_TYPES)
	{
		keyV := map[string]validators.IValidator{
			"status": validators.NewStringChoicesValidator("status", consts.LB_STATUS_SPEC).Default(consts.LB_STATUS_ENABLED),

			"address":      addressV.Optional(true),
			"address_type": addressTypeV.Default(consts.LB_ADDR_TYPE_INTRANET),
			"network":      networkV,
			"zone":         zoneV,
			"manager_id":   managerIdV,
		}
		for _, v := range keyV {
			if err := v.Validate(data); err != nil {
				return nil, err
			}
		}
	}
	var region *SCloudregion
	if addressTypeV.Value == consts.LB_ADDR_TYPE_INTRANET {
		network := networkV.Model.(*SNetwork)
		if ipAddr := addressV.IP; ipAddr != nil {
			ipS := ipAddr.String()
			ip, err := netutils.NewIPV4Addr(ipS)
			if err != nil {
				return nil, err
			}
			if !network.isAddressInRange(ip) {
				return nil, httperrors.NewInputParameterError("address %s is not in the range of network %s(%s)",
					ipS, network.Name, network.Id)
			}
			if network.isAddressUsed(ipS) {
				return nil, httperrors.NewInputParameterError("address %s is already occupied", ipS)
			}
		}
		if network.getFreeAddressCount() <= 0 {
			return nil, httperrors.NewNotAcceptableError("network %s(%s) has no free addresses",
				network.Name, network.Id)
		}
		wire := network.GetWire()
		if wire == nil {
			return nil, fmt.Errorf("getting wire failed")
		}
		vpc := wire.getVpc()
		if vpc == nil {
			return nil, fmt.Errorf("getting vpc failed")
		}
		data.Set("vpc_id", jsonutils.NewString(vpc.Id))
		if len(vpc.ManagerId) > 0 {
			data.Set("manager_id", jsonutils.NewString(vpc.ManagerId))
		}
		zone := wire.GetZone()
		if zone == nil {
			return nil, fmt.Errorf("getting zone failed")
		}
		data.Set("zone_id", jsonutils.NewString(zone.GetId()))
		region = zone.GetRegion()
		if region == nil {
			return nil, fmt.Errorf("getting region failed")
		}
		data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
		// TODO validate network is of classic type
		data.Set("network_type", jsonutils.NewString(consts.LB_NETWORK_TYPE_CLASSIC))
		data.Set("address_type", jsonutils.NewString(consts.LB_ADDR_TYPE_INTRANET))
	} else {
		zone := zoneV.Model.(*SZone)
		region = zone.GetRegion()
		if region == nil {
			return nil, fmt.Errorf("getting region failed")
		}
		// 公网 lb 实例和vpc、network无关联
		data.Set("vpc_id", jsonutils.NewString(""))
		data.Set("address", jsonutils.NewString(""))
		data.Set("network_id", jsonutils.NewString(""))
		data.Set("cloudregion_id", jsonutils.NewString(region.GetId()))
		data.Set("network_type", jsonutils.NewString(consts.LB_NETWORK_TYPE_VPC))
		data.Set("address_type", jsonutils.NewString(consts.LB_ADDR_TYPE_INTERNET))
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateCreateLoadbalancerData(ctx, userCred, data)
}

func (lb *SLoadbalancer) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lb, "status")
}

func (lb *SLoadbalancer) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if _, err := lb.SVirtualResourceBase.PerformStatus(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	if lb.Status == consts.LB_STATUS_ENABLED {
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
	lb.SetStatus(userCred, consts.LB_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerSyncstatusTask", lb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	// NOTE lb.Id will only be available after BeforeInsert happens
	// NOTE this means lb.UpdateVersion will be 0, then 1 after creation
	// NOTE need ways to notify error

	lb.SetStatus(userCred, consts.LB_CREATING, "")
	if err := lb.StartLoadBalancerCreateTask(ctx, userCred, ""); err != nil {
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

func (lb *SLoadbalancer) GetNetwork() *SNetwork {
	network, err := NetworkManager.FetchById(lb.NetworkId)
	if err != nil {
		return nil
	}
	return network.(*SNetwork)
}

func (lb *SLoadbalancer) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := lb.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovide for lb %s: %s", lb.Name, err)
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
	if lb.AddressType == consts.LB_ADDR_TYPE_INTRANET {
		vpc := lb.GetVpc()
		if vpc == nil {
			return nil, fmt.Errorf("failed to find vpc for lb %s", lb.Name)
		}
		iVpc, err := iRegion.GetIVpcById(vpc.ExternalId)
		if err != nil {
			return nil, err
		}
		params.VpcID = iVpc.GetId()
		network := lb.GetNetwork()
		if network == nil {
			return nil, fmt.Errorf("failed to find network for lb %s", lb.Name)
		}
		iNetwork, err := network.GetINetwork()
		if err != nil {
			return nil, err
		}
		params.NetworkID = iNetwork.GetId()
	}
	return params, nil
}

func (lb *SLoadbalancer) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lb, "purge")
}

func (lb *SLoadbalancer) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lb.StartLoadBalancerDeleteTask(ctx, userCred, parasm, "")
}

func (lb *SLoadbalancer) StartLoadBalancerDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerDeleteTask", lb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) StartLoadBalancerCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCreateTask", lb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lb *SLoadbalancer) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lb.GetOwnerProjectId())
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
	if lb.BackendGroupId == "" {
		return extra
	}
	lbbg, err := LoadbalancerBackendGroupManager.FetchById(lb.BackendGroupId)
	if err != nil {
		log.Errorf("loadbalancer %s(%s): fetch backend group (%s) error: %s",
			lb.Name, lb.Id, lb.BackendGroupId, err)
		return extra
	}
	extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))
	return extra
}

func (lb *SLoadbalancer) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lb.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lb *SLoadbalancer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lb.SetStatus(userCred, consts.LB_STATUS_DELETING, "")
	return lb.StartLoadBalancerDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lb *SLoadbalancer) PendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(lb.NetworkId) > 0 {
		req := &SLoadbalancerNetworkDeleteData{
			loadbalancer: lb,
		}
		LoadbalancernetworkManager.DeleteLoadbalancerNetwork(ctx, userCred, req)
	}
	lb.DoPendingDelete(ctx, userCred)
	lb.PreDeleteSubs(ctx, userCred)
}

func (lb *SLoadbalancer) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	ownerProjId := lb.GetOwnerProjectId()
	lbId := lb.Id
	subMen := []ILoadbalancerSubResourceManager{
		LoadbalancerListenerManager,
		LoadbalancerBackendGroupManager,
	}
	for _, subMan := range subMen {
		func(subMan ILoadbalancerSubResourceManager) {
			lockman.LockClass(ctx, subMan, ownerProjId)
			defer lockman.ReleaseClass(ctx, subMan, ownerProjId)
			q := subMan.Query().Equals("loadbalancer_id", lbId)
			subMan.PreDeleteSubs(ctx, userCred, q)
		}(subMan)
	}
}

func (lb *SLoadbalancer) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerManager) getLoadbalancersByRegion(region *SCloudregion, provider *SCloudprovider) ([]SLoadbalancer, error) {
	lbs := []SLoadbalancer{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id)
	if err := db.FetchModelObjects(man, q, &lbs); err != nil {
		log.Errorf("failed to get lbs for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return lbs, nil
}

func (man *SLoadbalancerManager) SyncLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbs []cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) ([]SLoadbalancer, []cloudprovider.ICloudLoadbalancer, compare.SyncResult) {
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
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, consts.LB_STATUS_UNKNOWN, "sync to delete")
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancer(ctx, userCred, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localLbs = append(localLbs, commondb[i])
			remoteLbs = append(remoteLbs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancer(ctx, userCred, provider, added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localLbs = append(localLbs, *new)
			remoteLbs = append(remoteLbs, added[i])
			syncResult.Add()
		}
	}
	return localLbs, remoteLbs, syncResult
}

func (man *SLoadbalancerManager) newFromCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extLb cloudprovider.ICloudLoadbalancer, region *SCloudregion) (*SLoadbalancer, error) {
	lb := SLoadbalancer{}
	lb.SetModelManager(man)

	lb.ManagerId = provider.Id
	lb.CloudregionId = region.Id
	lb.Address = extLb.GetAddress()
	lb.AddressType = extLb.GetAddressType()
	lb.NetworkType = extLb.GetNetworkType()
	lb.Name = extLb.GetName()
	lb.Status = extLb.GetStatus()
	lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
	lb.ChargeType = extLb.GetChargeType()
	lb.ExternalId = extLb.GetGlobalId()
	if networkId := extLb.GetNetworkId(); len(networkId) > 0 {
		if network, err := NetworkManager.FetchByExternalId(networkId); err == nil && network != nil {
			lb.NetworkId = network.GetId()
		}
	}
	if vpcId := extLb.GetVpcId(); len(vpcId) > 0 {
		if vpc, err := VpcManager.FetchByExternalId(vpcId); err == nil && vpc != nil {
			lb.VpcId = vpc.GetId()
		}
	}
	if zoneId := extLb.GetZoneId(); len(zoneId) > 0 {
		if zone, err := ZoneManager.FetchByExternalId(zoneId); err == nil && zone != nil {
			lb.ZoneId = zone.GetId()
		}
	}

	lb.ProjectSource = db.PROJECT_SOURCE_CLOUD
	lb.ProjectId = userCred.GetProjectId()
	if len(provider.ProjectId) > 0 {
		lb.ProjectId = provider.ProjectId
	}

	if extProjectId := extLb.GetProjectId(); len(extProjectId) > 0 {
		externalProject, err := ExternalProjectManager.GetProject(extProjectId, lb.ManagerId)
		if err != nil {
			log.Errorf(err.Error())
		} else {
			lb.ProjectId = externalProject.ProjectId
		}
	}

	if extLb.GetMetadata() != nil {
		lb.LBInfo = extLb.GetMetadata()
	}

	if err := man.TableSpec().Insert(&lb); err != nil {
		log.Errorf("newFromCloudRegion fail %s", err)
		return nil, err
	}
	lb.syncLoadbalancerNetwork(ctx, userCred)
	return &lb, nil
}

func (lb *SLoadbalancer) syncLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(lb.NetworkId) > 0 {
		lbNetReq := &SLoadbalancerNetworkRequestData{
			Loadbalancer: lb,
			NetworkId:    lb.NetworkId,
			Address:      lb.Address,
		}
		err := LoadbalancernetworkManager.SyncLoadbalancerNetwork(ctx, userCred, lbNetReq)
		if err != nil {
			log.Errorf("failed to create loadbalancer network: %v", err)
		}
	}
}

func (lb *SLoadbalancer) SyncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, extLb cloudprovider.ICloudLoadbalancer, projectId string, projectSync bool) error {
	_, err := lb.GetModelManager().TableSpec().Update(lb, func() error {
		lb.Address = extLb.GetAddress()
		lb.Status = extLb.GetStatus()
		lb.Name = extLb.GetName()
		lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
		lb.ChargeType = extLb.GetChargeType()

		if extLb.GetMetadata() != nil {
			lb.LBInfo = extLb.GetMetadata()
		}

		if projectSync && lb.ProjectSource != db.PROJECT_SOURCE_LOCAL {
			if extProjectId := extLb.GetProjectId(); len(extProjectId) > 0 {
				extProject, err := ExternalProjectManager.GetProject(extProjectId, lb.ManagerId)
				if err != nil {
					log.Errorf(err.Error())
				} else {
					lb.ProjectId = extProject.ProjectId
				}
			}
		}
		return nil
	})
	lb.syncLoadbalancerNetwork(ctx, userCred)
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
	_, err := lb.GetModelManager().TableSpec().Update(lb, func() error {
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
