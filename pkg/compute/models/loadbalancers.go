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
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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

	Address       string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	AddressType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	VpcId         string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	ZoneId        string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"required"`

	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"user" update:"user"`
}

func (man *SLoadbalancerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
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
	addressV := validators.NewIPv4AddrValidator("address")
	{
		keyV := map[string]validators.IValidator{
			"status": validators.NewStringChoicesValidator("status", LB_STATUS_SPEC).Default(LB_STATUS_ENABLED),

			"address": addressV.Optional(true),
			"network": networkV,
		}
		for _, v := range keyV {
			if err := v.Validate(data); err != nil {
				return nil, err
			}
		}
	}
	{
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
		if wire := network.GetWire(); wire == nil {
			return nil, fmt.Errorf("getting wire failed")
		} else if zone := wire.GetZone(); zone == nil {
			return nil, fmt.Errorf("getting zone failed")
		} else {
			data.Set("zone_id", jsonutils.NewString(zone.GetId()))
		}
		// TODO validate network is of classic type
		data.Set("network_type", jsonutils.NewString(LB_NETWORK_TYPE_CLASSIC))
		data.Set("address_type", jsonutils.NewString(LB_ADDR_TYPE_INTRANET))
	}
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (lb *SLoadbalancer) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lb, "status")
}

func (lb *SLoadbalancer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	// NOTE lb.Id will only be available after BeforeInsert happens
	// NOTE this means lb.UpdateVersion will be 0, then 1 after creation
	// NOTE need ways to notify error
	LoadbalancerManager.TableSpec().Update(lb, func() error {
		if lb.AddressType == LB_ADDR_TYPE_INTRANET {
			// TODO support use reserved ip address
			// TODO prefer ip address from server_type loadbalancer?
			req := &SLoadbalancerNetworkRequestData{
				loadbalancer: lb,
				networkId:    lb.NetworkId,
				address:      lb.Address,
			}
			// NOTE the small window when agents can see the ephemeral address
			ln, err := LoadbalancernetworkManager.NewLoadbalancerNetwork(ctx, userCred, req)
			if err != nil {
				log.Errorf("allocating loadbalancer network failed: %v, req: %#v", err, req)
				lb.Address = ""
			} else {
				lb.Address = ln.IpAddr
			}
		}
		return nil
	})
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

func (lb *SLoadbalancer) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lb.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (lb *SLoadbalancer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(lb.Address) > 0 {
		// TODO reserve support
		req := &SLoadbalancerNetworkDeleteData{
			loadbalancer: lb,
		}
		err := LoadbalancernetworkManager.DeleteLoadbalancerNetwork(ctx, userCred, req)
		if err != nil {
			return err
		}
		lb.Address = ""
	}
	// TODO How about mark pending delete and return
	return nil
}

func (lb *SLoadbalancer) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lb.SetStatus(userCred, LB_STATUS_DISABLED, "preDelete")
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
			err = removed[i].SetStatus(userCred, LB_STATUS_UNKNOWN, "sync to delete")
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

	lb.ProjectId = userCred.GetProjectId()
	if len(provider.ProjectId) > 0 {
		lb.ProjectId = provider.ProjectId
	}

	if err := man.TableSpec().Insert(&lb); err != nil {
		log.Errorf("newFromCloudRegion fail %s", err)
		return nil, err
	}
	return &lb, nil
}

func (lb *SLoadbalancer) SyncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, extLb cloudprovider.ICloudLoadbalancer, projectId string, projectSync bool) error {
	_, err := LoadbalancerManager.TableSpec().Update(lb, func() error {
		lb.Address = extLb.GetAddress()
		lb.Status = extLb.GetStatus()
		lb.Name = extLb.GetName()

		if projectSync && len(projectId) > 0 {
			lb.ProjectId = projectId
		}

		return nil
	})
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
	q = q.Filter(sqlchemy.OR(sqlchemy.IsEmpty(q.Field("cloudregion_id")), sqlchemy.IsNull(q.Field("cloudregion_id"))))
	if err := db.FetchModelObjects(LoadbalancerManager, q, &lbs); err != nil {
		log.Errorf("fetch all lbs fail %v", err)
		return err
	}
	for i := 0; i < len(lbs); i++ {
		if err := lbs[i].setCloudregionId(); err != nil {
			log.Errorf("fill cloud region info failed error: %v", err)
		}
	}
	return nil
}
