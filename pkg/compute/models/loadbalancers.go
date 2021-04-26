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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerManager struct {
	SLoadbalancerLogSkipper

	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SVpcResourceBaseManager
	SZoneResourceBaseManager
	SNetworkResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager

	SLoadbalancerClusterResourceBaseManager
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
	SDeletePreventableResourceBase

	// LB might optionally be in a VPC, vpc_id, manager_id, cloudregion_id
	SVpcResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// zone_id
	SZoneResourceBase
	// optional network_id
	SNetworkResourceBase `width:"147" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	SLoadbalancerRateLimiter

	// 备可用区
	Zone1 string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" json:"zone_1"`

	// IP地址
	Address string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"address"`
	// 地址类型
	AddressType string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"address_type"`
	// 网络类型
	NetworkType string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"network_type"`

	// 子网Id
	// NetworkId string `width:"147" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 虚拟私有网络Id
	// VpcId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	SLoadbalancerClusterResourceBase

	// 计费类型
	ChargeType string `list:"user" get:"user" create:"optional" update:"user" json:"charge_type"`

	// 套餐名称
	LoadbalancerSpec string `list:"user" get:"user" list:"user" create:"optional" json:"loadbalancer_spec"`

	// 默认后端服务器组Id
	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" json:"backend_group_id"`

	// LB的其他配置信息
	LBInfo jsonutils.JSONObject `charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional" json:"lb_info"`
}

// 负载均衡实例列表
func (man *SLoadbalancerManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = man.SDeletePreventableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DeletePreventableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDeletePreventableResourceBaseManager.ListItemFilter")
	}
	vpcQuery := api.VpcFilterListInput{
		VpcFilterListInputBase: query.VpcFilterListInputBase,
	}
	q, err = man.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, vpcQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, err = man.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	netQuery := api.NetworkFilterListInput{
		NetworkFilterListBase: query.NetworkFilterListBase,
	}
	q, err = man.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, netQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	ownerId := userCred
	data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		// {Key: "network", ModelKeyword: "network", OwnerId: ownerId},
		{Key: "cluster", ModelKeyword: "loadbalancercluster", OwnerId: ownerId},
	})
	if err != nil {
		return nil, err
	}

	if len(query.Address) > 0 {
		q = q.In("address", query.Address)
	}
	if len(query.AddressType) > 0 {
		q = q.In("address_type", query.AddressType)
	}
	if len(query.NetworkType) > 0 {
		q = q.In("network_type", query.NetworkType)
	}
	if len(query.ChargeType) > 0 {
		q = q.In("charge_type", query.ChargeType)
	}
	if len(query.LoadbalancerSpec) > 0 {
		q = q.In("loadbalancer_spec", query.LoadbalancerSpec)
	}

	return q, nil
}

func (man *SLoadbalancerManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	vpcQuery := api.VpcFilterListInput{
		VpcFilterListInputBase: query.VpcFilterListInputBase,
	}
	q, err = man.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, vpcQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, err = man.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}
	netQuery := api.NetworkFilterListInput{
		NetworkFilterListBase: query.NetworkFilterListBase,
	}
	q, err = man.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, netQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := api.LoadbalancerCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, err
	}

	newData, err := man.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}

	return newData, nil
}

func (man *SLoadbalancerManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.LoadbalancerCreateInput,
) (*jsonutils.JSONDict, error) {
	var err error

	var region *SCloudregion
	if len(input.VpcId) > 0 {
		_vpc, err := validators.ValidateModel(userCred, VpcManager, &input.VpcId)
		if err != nil {
			return nil, err
		}
		vpc := _vpc.(*SVpc)
		region, _ = vpc.GetRegion()
	} else if len(input.ZoneId) > 0 {
		var zone *SZone
		zone, input.ZoneResourceInput, err = ValidateZoneResourceInput(userCred, input.ZoneResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateZoneResourceInput")
		}
		region = zone.GetRegion()
	} else if len(input.NetworkId) > 0 {
		if strings.IndexByte(input.NetworkId, ',') >= 0 {
			input.NetworkId = strings.Split(input.NetworkId, ",")[0]
		}
		var network *SNetwork
		network, input.NetworkResourceInput, err = ValidateNetworkResourceInput(userCred, input.NetworkResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateNetworkResourceInput")
		}
		region = network.GetRegion()
	}

	if region == nil {
		return nil, httperrors.NewBadRequestError("cannot find region info")
	}

	input.CloudregionId = region.GetId()

	var cloudprovider *SCloudprovider
	if len(input.CloudproviderId) > 0 {
		cloudprovider, input.CloudproviderResourceInput, err = ValidateCloudproviderResourceInput(userCred, input.CloudproviderResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateCloudproviderResourceInput")
		}
	}

	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	quotaKeys := fetchRegionalQuotaKeys(rbacutils.ScopeProject, ownerId, region, cloudprovider)
	pendingUsage := SRegionQuota{Loadbalancer: 1}
	pendingUsage.SetKeys(quotaKeys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}

	return region.GetDriver().ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input.JSON(input))
}

func (lb *SLoadbalancer) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lb, "status")
}

func (lb *SLoadbalancer) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	if _, err := lb.SVirtualResourceBase.PerformStatus(ctx, userCred, query, input); err != nil {
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
	return nil, StartResourceSyncStatusTask(ctx, userCred, lb, "LoadbalancerSyncstatusTask", "")
}

func (lb *SLoadbalancer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// NOTE lb.Id will only be available after BeforeInsert happens
	// NOTE this means lb.UpdateVersion will be 0, then 1 after creation
	// NOTE need ways to notify error

	pendingUsage := SRegionQuota{Loadbalancer: 1}
	pendingUsage.SetKeys(lb.GetQuotaKeys())
	err := quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	if err != nil {
		log.Errorf("CancelPendingUsage error %s", err)
	}

	lb.SetStatus(userCred, api.LB_CREATING, "")
	if err := lb.StartLoadBalancerCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create loadbalancer error: %v", err)
	}
}

func (lb *SLoadbalancer) GetCloudprovider() *SCloudprovider {
	return lb.SManagedResourceBase.GetCloudprovider()
}

func (lb *SLoadbalancer) GetRegion() *SCloudregion {
	return lb.SCloudregionResourceBase.GetRegion()
}

func (lb *SLoadbalancer) GetCloudproviderId() string {
	return lb.SManagedResourceBase.GetCloudproviderId()
}

func (lb *SLoadbalancer) GetZone() *SZone {
	return lb.SZoneResourceBase.GetZone()
}

func (lb *SLoadbalancer) GetVpc() *SVpc {
	return lb.SVpcResourceBase.GetVpc()
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
		return nil, errors.Wrap(err, "lb.GetDriver")
	}
	region := lb.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to get region for lb %s", lb.Name)
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
	params.Tags, _ = lb.GetAllUserMetadata()

	if len(lb.ZoneId) > 0 {
		zone := lb.GetZone()
		if zone == nil {
			return nil, fmt.Errorf("failed to find zone for lb %s", lb.Name)
		}
		iZone, err := iRegion.GetIZoneById(zone.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "GetIZoneById")
		}
		params.ZoneID = iZone.GetId()
	}

	if len(lb.Zone1) > 0 {
		z1 := ZoneManager.FetchZoneById(lb.Zone1)
		if z1 == nil {
			return nil, fmt.Errorf("failed to find zone 1 for lb %s", lb.Name)
		}
		iZone, err := iRegion.GetIZoneById(z1.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "GetIZoneById")
		}
		params.SlaveZoneID = iZone.GetId()
	}

	if lb.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH {
		params.EgressMbps = lb.EgressMbps
	}

	if lb.AddressType == api.LB_ADDR_TYPE_INTRANET || utils.IsInStringArray(lb.SManagedResourceBase.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_AWS, api.CLOUD_PROVIDER_QCLOUD}) {
		vpc := lb.GetVpc()
		if vpc == nil {
			return nil, fmt.Errorf("failed to find vpc for lb %s", lb.Name)
		}
		iVpc, err := iRegion.GetIVpcById(vpc.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "GetIVpcById")
		}
		params.VpcID = iVpc.GetId()
	}

	if lb.AddressType == api.LB_ADDR_TYPE_INTRANET || utils.IsInStringArray(lb.SManagedResourceBase.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_AWS}) {
		networks, err := lb.GetNetworks()
		if err != nil {
			return nil, fmt.Errorf("failed to find network for lb %s: %s", lb.Name, err)
		}

		for i := range networks {
			iNetwork, err := networks[i].GetINetwork()
			if err != nil {
				return nil, errors.Wrap(err, "GetINetwork")
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
	var (
		ownerId       = lb.GetOwnerId()
		backendGroupV = validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
		clusterV      = validators.NewModelIdOrNameValidator("cluster", "loadbalancercluster", ownerId)
		keyV          = map[string]validators.IValidator{
			"backend_group": backendGroupV,
			"cluster":       clusterV,
		}
	)
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.LoadbalancerId != lb.Id {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s, not %s",
			backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, lb.Id)
	}
	if clusterV.Model != nil {
		var (
			cluster = clusterV.Model.(*SLoadbalancerCluster)
			network = lb.GetNetwork()
			wire    = network.GetWire()
			zone    = wire.GetZone()
		)
		if cluster.ZoneId != zone.Id {
			return nil, httperrors.NewInputParameterError("cluster zone %s does not match network zone %s ",
				cluster.ZoneId, zone.Id)
		}
		if cluster.WireId != "" && cluster.WireId != network.WireId {
			return nil, httperrors.NewInputParameterError("cluster wire affiliation does not match network's: %s != %s",
				cluster.WireId, network.WireId)
		}
	}

	input := apis.VirtualResourceBaseUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lb.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (man *SLoadbalancerManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerDetails {
	rows := make([]api.LoadbalancerDetails, len(objs))

	clusterRows := man.SLoadbalancerClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := man.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := man.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := man.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := man.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zone1Rows := man.FetchZone1ResourceInfos(ctx, userCred, query, objs)
	netRows := man.SNetworkResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerDetails{
			LoadbalancerClusterResourceInfo: clusterRows[i],

			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regRows[i],
			VpcResourceInfoBase:     vpcRows[i].VpcResourceInfoBase,
			ZoneResourceInfoBase:    zoneRows[i].ZoneResourceInfoBase,
			Zone1ResourceInfoBase:   zone1Rows[i],
			NetworkResourceInfoBase: netRows[i].NetworkResourceInfoBase,
		}
		rows[i], _ = objs[i].(*SLoadbalancer).getMoreDetails(rows[i])
	}

	return rows
}

func (lb *SLoadbalancerManager) FetchZone1ResourceInfos(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{}) []api.Zone1ResourceInfoBase {
	rows := make([]api.Zone1ResourceInfoBase, len(objs))
	zoneIds := []string{}
	for i := range objs {
		zone1 := objs[i].(*SLoadbalancer).Zone1
		if len(zone1) > 0 {
			zoneIds = append(zoneIds, zone1)
		}
	}

	zones := make(map[string]SZone)
	err := db.FetchStandaloneObjectsByIds(ZoneManager, zoneIds, &zones)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range objs {
		if zone, ok := zones[objs[i].(*SLoadbalancer).Zone1]; ok {
			rows[i].Zone1Name = zone.GetName()
			rows[i].Zone1ExtId = fetchExternalId(zone.GetExternalId())
		}
	}

	return rows
}

func (lb *SLoadbalancer) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.LoadbalancerDetails, error) {
	return api.LoadbalancerDetails{}, nil
}

func (lb *SLoadbalancer) getMoreDetails(out api.LoadbalancerDetails) (api.LoadbalancerDetails, error) {
	eip, _ := lb.GetEip()
	if eip != nil {
		out.Eip = eip.IpAddr
		out.EipMode = eip.Mode
	}

	if lb.BackendGroupId != "" {
		lbbg, err := LoadbalancerBackendGroupManager.FetchById(lb.BackendGroupId)
		if err != nil {
			log.Errorf("loadbalancer %s(%s): fetch backend group (%s) error: %s",
				lb.Name, lb.Id, lb.BackendGroupId, err)
			return out, err
		}
		out.BackendGroup = lbbg.GetName()
	}

	return out, nil
}

func (lb *SLoadbalancer) ValidateDeleteCondition(ctx context.Context) error {
	err := lb.validatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	if lb.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("loadbalancer is locked, cannot delete")
	}

	return nil
}

func (lb *SLoadbalancer) validatePurgeCondition(ctx context.Context) error {
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
	q := man.Query()
	q = q.Equals("manager_id", provider.Id)
	q = q.Equals("cloudregion_id", region.Id)
	q = q.IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbs); err != nil {
		log.Errorf("failed to get lbs for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return lbs, nil
}

func (man *SLoadbalancerManager) getLoadbalancersByExternalIds(externalIds []string) ([]SLoadbalancer, error) {
	lbs := []SLoadbalancer{}
	q := man.Query()
	q = q.In("external_id", externalIds)
	q = q.IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbs); err != nil {
		log.Errorf("failed to get lbs for region: %#v error: %v", externalIds, err)
		return nil, err
	}
	return lbs, nil
}

func (man *SLoadbalancerManager) getLocalLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbs []cloudprovider.ICloudLoadbalancer) ([]SLoadbalancer, error) {
	// current external ID
	extIds := []string{}
	for i := range lbs {
		extIds = append(extIds, lbs[i].GetGlobalId())
	}

	part1, err := man.getLoadbalancersByRegion(region, provider)
	if err != nil {
		return nil, err
	}

	localLbs := map[string]SLoadbalancer{}
	for i := range part1 {
		localLbs[part1[i].Id] = part1[i]
		if len(part1[i].GetExternalId()) > 0 {
			extIds = append(extIds, part1[i].GetExternalId())
		}
	}

	if len(extIds) > 0 {
		part2, err := man.getLoadbalancersByExternalIds(extIds)
		if err != nil {
			return nil, err
		}

		for i := range part2 {
			localLbs[part2[i].Id] = part2[i]
		}
	}

	ret := make([]SLoadbalancer, 0)
	for id, _ := range localLbs {
		ret = append(ret, localLbs[id])
	}

	return ret, nil
}

func (man *SLoadbalancerManager) SyncLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbs []cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) ([]SLoadbalancer, []cloudprovider.ICloudLoadbalancer, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "loadbalance", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "loadbalance", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	localLbs := []SLoadbalancer{}
	remoteLbs := []cloudprovider.ICloudLoadbalancer{}
	syncResult := compare.SyncResult{}

	dbLbs, err := man.getLocalLoadbalancers(ctx, userCred, provider, region, lbs)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := range dbLbs {
		if taskman.TaskManager.IsInTask(&dbLbs[i]) {
			syncResult.Error(fmt.Errorf("loadbalancer %s(%s)in task", dbLbs[i].Name, dbLbs[i].Id))
			return nil, nil, syncResult
		}
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
		err = commondb[i].SyncWithCloudLoadbalancer(ctx, userCred, commonext[i], syncOwnerId, provider, region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncVirtualResourceMetadata(ctx, userCred, &commondb[i], commonext[i])
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
			syncVirtualResourceMetadata(ctx, userCred, new, added[i])
			localLbs = append(localLbs, *new)
			remoteLbs = append(remoteLbs, added[i])
			syncResult.Add()
		}
	}
	return localLbs, remoteLbs, syncResult
}

func getExtLbNetworkIds(extLb cloudprovider.ICloudLoadbalancer, managerId string) []string {
	extNetworkIds := extLb.GetNetworkIds()
	lbNetworkIds := []string{}
	for _, networkId := range extNetworkIds {
		network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), managerId))
		})
		if err == nil && network != nil {
			lbNetworkIds = append(lbNetworkIds, network.GetId())
		}
	}

	return lbNetworkIds
}

func (man *SLoadbalancerManager) newFromCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extLb cloudprovider.ICloudLoadbalancer, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancer, error) {
	lb := SLoadbalancer{}
	lb.SetModelManager(man, &lb)

	lb.ManagerId = provider.Id
	lb.CloudregionId = region.Id
	lb.Address = extLb.GetAddress()
	lb.AddressType = extLb.GetAddressType()
	lb.NetworkType = extLb.GetNetworkType()

	lb.Status = extLb.GetStatus()
	lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
	lb.ChargeType = extLb.GetChargeType()
	lb.EgressMbps = extLb.GetEgressMbps()
	lb.ExternalId = extLb.GetGlobalId()
	lbNetworkIds := getExtLbNetworkIds(extLb, lb.ManagerId)
	lb.NetworkId = strings.Join(lbNetworkIds, ",")

	// classic vpc
	if extLb.GetNetworkType() == api.LB_NETWORK_TYPE_CLASSIC {
		if vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, provider, region); err == nil && vpc != nil {
			lb.VpcId = vpc.GetId()
		}
	}

	// vpc
	if vpcId := extLb.GetVpcId(); len(vpcId) > 0 {
		if vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		}); err == nil && vpc != nil {
			lb.VpcId = vpc.GetId()
		}
	}
	if zoneId := extLb.GetZoneId(); len(zoneId) > 0 {
		if zone, err := db.FetchByExternalId(ZoneManager, zoneId); err == nil && zone != nil {
			lb.ZoneId = zone.GetId()
		}
	}

	if zoneId := extLb.GetZone1Id(); len(zoneId) > 0 {
		if zone, err := db.FetchByExternalId(ZoneManager, zoneId); err == nil && zone != nil {
			lb.Zone1 = zone.GetId()
		}
	}

	if extLb.GetSysTags() != nil {
		lb.LBInfo = jsonutils.Marshal(extLb.GetSysTags())
	}

	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		var err error
		lb.Name, err = db.GenerateName(ctx, man, syncOwnerId, extLb.GetName())
		if err != nil {
			return err
		}

		return man.TableSpec().Insert(ctx, &lb)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, &lb, syncOwnerId, extLb, provider.Id)

	db.OpsLog.LogEvent(&lb, db.ACT_CREATE, lb.GetShortDesc(ctx), userCred)

	lb.syncLoadbalancerNetwork(ctx, userCred, lbNetworkIds)
	return &lb, nil
}

func (lb *SLoadbalancer) syncRemoveCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	err := lb.validatePurgeCondition(ctx)
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
	return ElasticipManager.getEip(api.EIP_ASSOCIATE_TYPE_LOADBALANCER, self.Id, "")
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

func (lb *SLoadbalancer) SyncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, extLb cloudprovider.ICloudLoadbalancer, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	diff, err := db.Update(lb, func() error {
		lb.Address = extLb.GetAddress()
		lb.Status = extLb.GetStatus()
		// lb.Name = extLb.GetName()
		lb.LoadbalancerSpec = extLb.GetLoadbalancerSpec()
		lb.EgressMbps = extLb.GetEgressMbps()
		lb.ChargeType = extLb.GetChargeType()
		lb.ManagerId = provider.Id
		lbNetworkIds := getExtLbNetworkIds(extLb, lb.ManagerId)
		lb.NetworkId = strings.Join(lbNetworkIds, ",")
		if extLb.GetSysTags() != nil {
			lb.LBInfo = jsonutils.Marshal(extLb.GetSysTags())
		}
		syncVirtualResourceMetadata(ctx, userCred, lb, extLb)

		// classic vpc
		if extLb.GetNetworkType() == api.LB_NETWORK_TYPE_CLASSIC {
			if vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, provider, region); err == nil && vpc != nil {
				lb.VpcId = vpc.GetId()
			}
		}

		// vpc
		if vpcId := extLb.GetVpcId(); len(vpcId) > 0 {
			if vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			}); err == nil && vpc != nil {
				lb.VpcId = vpc.GetId()
			}
		}

		extZoneId := extLb.GetZoneId()
		if len(extZoneId) > 0 {
			if zone, err := db.FetchByExternalId(ZoneManager, extZoneId); err == nil && zone != nil {
				lb.ZoneId = zone.GetId()
			}
		}

		if len(lb.Zone1) == 0 {
			extZoneId := extLb.GetZone1Id()
			if len(extZoneId) > 0 {
				if zone, err := db.FetchByExternalId(ZoneManager, extZoneId); err == nil && zone != nil {
					lb.Zone1 = zone.GetId()
				}
			}
		}

		return nil
	})

	db.OpsLog.LogSyncUpdate(lb, diff, userCred)

	networkIds := getExtLbNetworkIds(extLb, lb.ManagerId)
	SyncCloudProject(userCred, lb, syncOwnerId, extLb, provider.Id)
	lb.syncLoadbalancerNetwork(ctx, userCred, networkIds)

	return err
}

/*func (lb *SLoadbalancer) setCloudregionId() error {
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
}*/

func (man *SLoadbalancerManager) InitializeData() error {
	/*lbs := []SLoadbalancer{}
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
	}*/
	return nil
}

func (manager *SLoadbalancerManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerManager) FetchByExternalId(providerId string, extId string) (*SLoadbalancer, error) {
	ret := []SLoadbalancer{}
	vpcs := VpcManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
	q = q.Filter(sqlchemy.Equals(vpcs.Field("manager_id"), providerId))
	q = q.Equals("external_id", extId)
	q = q.IsFalse("pending_deleted")
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
	q := manager.Query().IsFalse("pending_deleted").IsNotEmpty("backend_group_id")
	err := q.All(&lbs)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerManager.GetLbDefaultBackendGroupIds")
	}

	ret := []string{}
	for i := range lbs {
		ret = append(ret, lbs[i].BackendGroupId)
	}

	return ret, nil
}

func (man *SLoadbalancerManager) getLoadbalancer(lbId string) (*SLoadbalancer, error) {
	obj, err := man.FetchById(lbId)
	if err != nil {
		return nil, errors.Wrapf(err, "get loadbalancer %s", lbId)
	}
	lb := obj.(*SLoadbalancer)
	if lb.PendingDeleted {
		return nil, errors.Wrap(errors.ErrNotFound, "pending deleted")
	}
	return lb, nil
}

func (man *SLoadbalancerManager) TotalCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
) (int, error) {
	q := man.Query()
	q = scopeOwnerIdFilter(q, scope, ownerId)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, nil, q.Field("zone_id"), q.Field("manager_id"), nil, nil)
	return q.CountWithError()
}

func (lb *SLoadbalancer) GetQuotaKeys() quotas.IQuotaKeys {
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		lb.GetOwnerId(),
		lb.GetRegion(),
		lb.GetCloudprovider(),
	)
}

func (lb *SLoadbalancer) GetUsages() []db.IUsage {
	if lb.PendingDeleted || lb.Deleted {
		return nil
	}
	usage := SRegionQuota{Loadbalancer: 1}
	keys := lb.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (manager *SLoadbalancerManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.Contains("zone") {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"zone"}))
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.Contains("vpc") {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"vpc"}))
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny("network", "wire") {
		q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"network", "wire"}))
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (self *SLoadbalancer) GetChangeOwnerCandidateDomainIds() []string {
	return self.SManagedResourceBase.GetChangeOwnerCandidateDomainIds()
}

func (self *SLoadbalancer) AllowPerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "remote-update")
}

func (self *SLoadbalancer) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LoadbalancerRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SLoadbalancer) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	if replaceTags {
		data.Add(jsonutils.JSONTrue, "replace_tags")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return errors.Wrap(err, "Start LoadbalancerRemoteUpdateTask")
	} else {
		self.SetStatus(userCred, api.LB_UPDATE_TAGS, "StartRemoteUpdateTask")
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SLoadbalancer) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}
