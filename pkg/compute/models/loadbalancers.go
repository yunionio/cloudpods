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
	"net"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
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
	SBillingResourceBaseManager

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

type SLoadbalancer struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SBillingResourceBase
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

	SLoadbalancerClusterResourceBase

	// 计费类型
	ChargeType string `list:"user" get:"user" create:"optional" update:"user" json:"charge_type"`

	// 套餐名称
	LoadbalancerSpec string `list:"user" get:"user" list:"user" create:"optional" json:"loadbalancer_spec"`

	// 默认后端服务器组Id
	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" json:"backend_group_id"`
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
	q, err = man.SBillingResourceBaseManager.ListItemFilter(ctx, q, userCred, query.BillingResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBillingResourceBaseManager.ListItemFilter")
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
	q, err = validators.ApplyModelFilters(ctx, q, data, []*validators.ModelFilterOptions{
		// {Key: "network", ModelKeyword: "network", OwnerId: ownerId},
		{Key: "cluster", ModelKeyword: "loadbalancercluster", OwnerId: ownerId},
	})
	if err != nil {
		return nil, err
	}

	if len(query.Address) == 1 {
		c1 := sqlchemy.In(q.Field("id"), ElasticipManager.Query("associate_id").Contains("ip_addr", query.Address[0]))
		c2 := sqlchemy.Contains(q.Field("address"), query.Address[0])
		q = q.Filter(sqlchemy.OR(c1, c2))
	} else if len(query.Address) > 1 {
		c1 := sqlchemy.In(q.Field("id"), ElasticipManager.Query("associate_id").In("ip_addr", query.Address))
		c2 := sqlchemy.In(q.Field("address"), query.Address)
		q = q.Filter(sqlchemy.OR(c1, c2))
	}

	// eip filters
	usableLbForEipFilter := query.UsableLoadbalancerForEip
	if len(usableLbForEipFilter) > 0 {
		eipObj, err := ElasticipManager.FetchByIdOrName(ctx, userCred, usableLbForEipFilter)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("eip %s not found", usableLbForEipFilter)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		eip := eipObj.(*SElasticip)

		if len(eip.NetworkId) > 0 {
			// kvm
			sq := LoadbalancernetworkManager.Query("loadbalancer_id").Equals("network_id", eip.NetworkId).SubQuery()
			q = q.NotIn("id", sq)
			if cp := eip.GetCloudprovider(); cp == nil || cp.Provider == api.CLOUD_PROVIDER_ONECLOUD {
				gnq := LoadbalancernetworkManager.Query().SubQuery()
				nq := NetworkManager.Query().SubQuery()
				wq := WireManager.Query().SubQuery()
				vq := VpcManager.Query().SubQuery()
				q.Join(gnq, sqlchemy.Equals(gnq.Field("loadbalancer_id"), q.Field("id")))
				q.Join(nq, sqlchemy.Equals(nq.Field("id"), gnq.Field("network_id")))
				q.Join(wq, sqlchemy.Equals(wq.Field("id"), nq.Field("wire_id")))
				q.Join(vq, sqlchemy.Equals(vq.Field("id"), wq.Field("vpc_id")))
				q.Filter(sqlchemy.NotEquals(vq.Field("id"), api.DEFAULT_VPC_ID))
				// vpc provider thing will be handled ok below
			}
		}

		if eip.ManagerId != "" {
			q = q.Equals("manager_id", eip.ManagerId)
		} else {
			q = q.IsNullOrEmpty("manager_id")
		}
		region, err := eip.GetRegion()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "eip.GetRegion"))
		}
		q = q.Equals("cloudregion_id", region.Id)
	}

	withEip := (query.WithEip != nil && *query.WithEip)
	withoutEip := (query.WithoutEip != nil && *query.WithoutEip) || (query.EipAssociable != nil && *query.EipAssociable)
	if withEip || withoutEip {
		eips := ElasticipManager.Query().SubQuery()
		sq := eips.Query(eips.Field("associate_id")).Equals("associate_type", api.EIP_ASSOCIATE_TYPE_LOADBALANCER)
		sq = sq.IsNotNull("associate_id").IsNotEmpty("associate_id")

		if withEip {
			q = q.In("id", sq)
		} else if withoutEip {
			q = q.NotIn("id", sq)
		}
	}

	if query.EipAssociable != nil {
		sq1 := NetworkManager.Query("id")
		sq2 := WireManager.Query().SubQuery()
		sq3 := VpcManager.Query().SubQuery()
		sq1 = sq1.Join(sq2, sqlchemy.Equals(sq1.Field("wire_id"), sq2.Field("id")))
		sq1 = sq1.Join(sq3, sqlchemy.Equals(sq2.Field("vpc_id"), sq3.Field("id")))
		cond1 := []string{api.VPC_EXTERNAL_ACCESS_MODE_EIP, api.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW}
		if *query.EipAssociable {
			sq1 = sq1.Filter(sqlchemy.In(sq3.Field("external_access_mode"), cond1))
		} else {
			sq1 = sq1.Filter(sqlchemy.NotIn(sq3.Field("external_access_mode"), cond1))
		}
		sq := LoadbalancernetworkManager.Query("loadbalancer_id").In("network_id", sq1)
		q = q.In("id", sq)
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

	if len(query.SecgroupId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, SecurityGroupManager, &query.SecgroupId)
		if err != nil {
			return nil, err
		}
		sq := LoadbalancerSecurityGroupManager.Query("loadbalancer_id").Equals("secgroup_id", query.SecgroupId)
		q = q.In("id", sq.SubQuery())
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

func (man *SLoadbalancerManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	return man.ValidateCreateData(ctx, userCred, ownerId, query, input)
}

func (man *SLoadbalancerManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.LoadbalancerCreateInput,
) (*api.LoadbalancerCreateInput, error) {
	if len(input.NetworkId) > 0 {
		networks := strings.Split(input.NetworkId, ",")
		if len(networks) > 1 {
			input.Networks = networks[1:]
		}
		input.NetworkId = networks[0]
		networkObj, err := validators.ValidateModel(ctx, userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return nil, err
		}
		network := networkObj.(*SNetwork)
		wire, err := network.GetWire()
		if err != nil {
			return nil, err
		}
		if len(wire.ZoneId) > 0 {
			input.ZoneId = wire.ZoneId
		}
		vpc, err := network.GetVpc()
		if err != nil {
			return nil, err
		}
		input.VpcId = vpc.Id
		input.ManagerId = vpc.ManagerId
		input.CloudproviderId = vpc.ManagerId
		input.CloudregionId = vpc.CloudregionId
		for i := range input.Networks {
			netObj, err := validators.ValidateModel(ctx, userCred, NetworkManager, &input.Networks[i])
			if err != nil {
				return nil, err
			}
			network := netObj.(*SNetwork)
			vpc, err := network.GetVpc()
			if err != nil {
				return nil, err
			}
			if vpc.Id != input.VpcId {
				return nil, httperrors.NewInputParameterError("all networks should in the same vpc.")
			}
		}
		if len(input.Address) > 0 {
			addr, err := netutils.NewIPV4Addr(input.Address)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalidate address %s", input.Address)
			}
			if !network.IsAddressInRange(addr) {
				return nil, httperrors.NewInputParameterError("address %s not in network %s", input.Address, network.Name)
			}
		}
	} else if len(input.ZoneId) > 0 {
		zoneObj, err := validators.ValidateModel(ctx, userCred, ZoneManager, &input.ZoneId)
		if err != nil {
			return nil, err
		}
		zone := zoneObj.(*SZone)
		input.CloudregionId = zone.CloudregionId
	}

	if len(input.CloudregionId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudregion_id")
	}

	var cloudprovider *SCloudprovider = nil
	if len(input.CloudproviderId) > 0 {
		managerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = input.CloudproviderId
		cloudprovider = managerObj.(*SCloudprovider)
	}

	if len(input.VpcId) > 0 {
		_vpc, err := validators.ValidateModel(ctx, userCred, VpcManager, &input.VpcId)
		if err != nil {
			return nil, err
		}
		vpc := _vpc.(*SVpc)
		if input.ManagerId != vpc.ManagerId {
			return nil, httperrors.NewInputParameterError("lb manager %s does not match vpc manager %s", input.ManagerId, vpc.ManagerId)
		}
		if input.CloudregionId != vpc.CloudregionId {
			return nil, httperrors.NewInputParameterError("lb region %s does not match vpc region %s", input.CloudregionId, vpc.CloudregionId)
		}
	}

	if len(input.Zone1) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, ZoneManager, &input.Zone1)
		if err != nil {
			return nil, err
		}
	}
	if len(input.AddressType) == 0 {
		input.AddressType = api.LB_ADDR_TYPE_INTRANET
	}

	if len(input.NetworkType) == 0 {
		input.NetworkType = api.LB_NETWORK_TYPE_VPC
	}

	if len(input.EipId) > 0 {
		eipObj, err := validators.ValidateModel(ctx, userCred, ElasticipManager, &input.EipId)
		if err != nil {
			return nil, err
		}
		eip := eipObj.(*SElasticip)
		if eip.CloudregionId != input.CloudregionId {
			return nil, httperrors.NewInputParameterError("lb region %s does not match eip region %s", input.CloudregionId, eip.CloudregionId)
		}
		if eip.Status != api.EIP_STATUS_READY {
			return nil, httperrors.NewInvalidStatusError("eip %s status not ready", eip.Name)
		}
		if len(eip.AssociateType) > 0 {
			return nil, httperrors.NewInvalidStatusError("eip %s alread associate %s", eip.Name, eip.AssociateType)
		}
		if eip.ManagerId != input.ManagerId {
			return nil, httperrors.NewInputParameterError("lb manager %s does not match eip manager %s", input.ManagerId, eip.ManagerId)
		}
	}
	if len(input.Status) == 0 {
		input.Status = api.LB_STATUS_ENABLED
	}

	if len(input.AddressType) == 0 {
		input.AddressType = api.LB_ADDR_TYPE_INTRANET
	}

	if !utils.IsInStringArray(input.AddressType, []string{api.LB_ADDR_TYPE_INTRANET, api.LB_ADDR_TYPE_INTERNET}) {
		return nil, httperrors.NewInputParameterError("invalid address_type %s", input.AddressType)
	}

	if input.AddressType == api.LB_ADDR_TYPE_INTRANET && len(input.NetworkId) == 0 {
		return nil, httperrors.NewMissingParameterError("network_id")
	}

	if len(input.ChargeType) == 0 {
		input.ChargeType = api.LB_CHARGE_TYPE_BY_TRAFFIC
	}

	if !utils.IsInStringArray(input.ChargeType, []string{api.LB_CHARGE_TYPE_BY_BANDWIDTH, api.LB_CHARGE_TYPE_BY_TRAFFIC}) {
		return nil, httperrors.NewInputParameterError("invalid charge_type %s", input.ChargeType)
	}

	if len(input.Duration) > 0 {
		billingCycle, err := billing.ParseBillingCycle(input.Duration)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid duration %s", input.Duration)
		}

		if len(input.BillingType) == 0 {
			input.BillingType = billing_api.BILLING_TYPE_PREPAID
		}
		input.BillingCycle = billingCycle.String()
		input.Duration = billingCycle.String()
	}

	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	region := regionObj.(*SCloudregion)

	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	input, err = region.GetDriver().ValidateCreateLoadbalancerData(ctx, userCred, ownerId, input)
	if err != nil {
		return nil, err
	}

	quotaKeys := fetchRegionalQuotaKeys(rbacscope.ScopeProject, ownerId, region, cloudprovider)
	pendingUsage := SRegionQuota{Loadbalancer: 1}
	if input.EipBw > 0 && len(input.Eip) == 0 {
		pendingUsage.Eip = 1
	}
	pendingUsage.SetKeys(quotaKeys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}

	return input, nil
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
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lb *SLoadbalancer) StartLoadBalancerStopTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerStopTask", lb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lb *SLoadbalancer) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, lb, "LoadbalancerSyncstatusTask", "")
}

func (lb *SLoadbalancer) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, lb, "LoadbalancerSyncstatusTask", parentTaskId)
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

	input := &api.LoadbalancerCreateInput{}
	data.Unmarshal(input)
	lb.SetStatus(ctx, userCred, api.LB_CREATING, "")
	err = lb.StartLoadBalancerCreateTask(ctx, userCred, input)
	if err != nil {
		lb.SetStatus(ctx, userCred, api.LB_CREATE_FAILED, err.Error())
	}
}

func (lb *SLoadbalancer) GetCloudprovider() *SCloudprovider {
	return lb.SManagedResourceBase.GetCloudprovider()
}

func (lb *SLoadbalancer) GetCloudproviderId() string {
	return lb.SManagedResourceBase.GetCloudproviderId()
}

func (lb *SLoadbalancer) GetRegion() (*SCloudregion, error) {
	return lb.SCloudregionResourceBase.GetRegion()
}

func (lb *SLoadbalancer) GetVpc() (*SVpc, error) {
	return lb.SVpcResourceBase.GetVpc()
}

func (lb *SLoadbalancer) GetZone() (*SZone, error) {
	return lb.SZoneResourceBase.GetZone()
}

func (lb *SLoadbalancer) GetNetworks() ([]SNetwork, error) {
	networks := []SNetwork{}
	networkIds := strings.Split(lb.NetworkId, ",")
	err := NetworkManager.Query().In("id", networkIds).All(&networks)
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

func (lb *SLoadbalancer) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := lb.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "lb.GetDriver")
	}
	region, err := lb.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (lb *SLoadbalancer) GetILoadbalancer(ctx context.Context) (cloudprovider.ICloudLoadbalancer, error) {
	if len(lb.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := lb.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetILoadBalancerById(lb.ExternalId)
}

func (lb *SLoadbalancer) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "purge")
	return nil, lb.StartLoadBalancerDeleteTask(ctx, userCred, params, "")
}

func (lb *SLoadbalancer) StartLoadBalancerDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerDeleteTask", lb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lb *SLoadbalancer) StartLoadBalancerCreateTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerCreateInput) error {
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCreateTask", lb, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
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
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}
	if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.LoadbalancerId != lb.Id {
		return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s, not %s",
			backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, lb.Id)
	}
	if clusterV.Model != nil {
		var (
			cluster    = clusterV.Model.(*SLoadbalancerCluster)
			network, _ = lb.GetNetwork()
			wire, _    = network.GetWire()
			zone, _    = wire.GetZone()
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

type SLoadbalancerUsageCount struct {
	Id string
	api.LoadbalancerUsage
}

func (lm *SLoadbalancerManager) query(manager db.IModelManager, field string, lbIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("loadbalancer_id"),
		sqlchemy.COUNT(field),
	).In("loadbalancer_id", lbIds).GroupBy(sq.Field("loadbalancer_id")).SubQuery()
}

func (manager *SLoadbalancerManager) TotalResourceCount(lbIds []string) (map[string]api.LoadbalancerUsage, error) {
	// backendGroup
	lbgSQ := manager.query(LoadbalancerBackendGroupManager, "backend_group_cnt", lbIds, nil)
	// listener
	lisSQ := manager.query(LoadbalancerListenerManager, "listener_cnt", lbIds, nil)

	lb := manager.Query().SubQuery()
	lbQ := lb.Query(
		sqlchemy.SUM("backend_group_count", lbgSQ.Field("backend_group_cnt")),
		sqlchemy.SUM("listener_count", lisSQ.Field("listener_cnt")),
	)

	lbQ.AppendField(lbQ.Field("id"))

	lbQ = lbQ.LeftJoin(lbgSQ, sqlchemy.Equals(lbQ.Field("id"), lbgSQ.Field("loadbalancer_id")))
	lbQ = lbQ.LeftJoin(lisSQ, sqlchemy.Equals(lbQ.Field("id"), lisSQ.Field("loadbalancer_id")))

	lbQ = lbQ.Filter(sqlchemy.In(lbQ.Field("id"), lbIds)).GroupBy(lbQ.Field("id"))

	lbCount := []SLoadbalancerUsageCount{}
	err := lbQ.All(&lbCount)
	if err != nil {
		return nil, errors.Wrapf(err, "lbQ.All")
	}

	result := map[string]api.LoadbalancerUsage{}
	for i := range lbCount {
		result[lbCount[i].Id] = lbCount[i].LoadbalancerUsage
	}

	return result, nil
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

	lbIds := make([]string, len(objs))
	backendGroupIds := make([]string, len(objs))
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
		lb := objs[i].(*SLoadbalancer)
		lbIds[i] = lb.Id
		backendGroupIds[i] = lb.BackendGroupId
	}

	q := ElasticipManager.Query().Equals("associate_type", api.EIP_ASSOCIATE_TYPE_LOADBALANCER).In("associate_id", lbIds)
	eips := []SElasticip{}
	err := db.FetchModelObjects(ElasticipManager, q, &eips)
	if err != nil {
		log.Errorf("Fetch eips error: %v", err)
		return rows
	}
	eipMap := map[string]*SElasticip{}
	for i := range eips {
		eipMap[eips[i].AssociateId] = &eips[i]
	}

	bgMap, err := db.FetchIdNameMap2(LoadbalancerBackendGroupManager, backendGroupIds)
	if err != nil {
		log.Errorf("Fetch LoadbalancerBackendGroup error: %v", err)
		return rows
	}

	secSQ := SecurityGroupManager.Query().SubQuery()
	lsecs := LoadbalancerSecurityGroupManager.Query().SubQuery()
	q = secSQ.Query(
		secSQ.Field("id"),
		secSQ.Field("name"),
		lsecs.Field("loadbalancer_id"),
	).
		Join(lsecs, sqlchemy.Equals(lsecs.Field("secgroup_id"), secSQ.Field("id"))).
		Filter(sqlchemy.In(lsecs.Field("loadbalancer_id"), lbIds))

	secInfo := []struct {
		Id             string
		Name           string
		LoadbalancerId string
	}{}
	err = q.All(&secInfo)
	if err != nil {
		log.Errorf("query secgroup info error: %v", err)
		return rows
	}

	groups := map[string][]api.SimpleSecurityGroup{}
	for _, sec := range secInfo {
		_, ok := groups[sec.LoadbalancerId]
		if !ok {
			groups[sec.LoadbalancerId] = []api.SimpleSecurityGroup{}
		}
		groups[sec.LoadbalancerId] = append(groups[sec.LoadbalancerId], api.SimpleSecurityGroup{
			Id:   sec.Id,
			Name: sec.Name,
		})
	}

	usage, err := man.TotalResourceCount(lbIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}

	for i := range rows {
		eip, ok := eipMap[lbIds[i]]
		if ok {
			rows[i].Eip = eip.IpAddr
			rows[i].EipMode = eip.Mode
			rows[i].EipId = eip.Id
		}
		bg, ok := bgMap[backendGroupIds[i]]
		if ok {
			rows[i].BackendGroup = bg
		}
		rows[i].Secgroups, _ = groups[lbIds[i]]
		rows[i].LoadbalancerUsage, _ = usage[lbIds[i]]
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

func (lb *SLoadbalancer) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if lb.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("loadbalancer is locked, cannot delete")
	}

	return lb.SVirtualResourceBase.ValidateDeleteCondition(ctx, info)
}

func (lb *SLoadbalancer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lb.SetStatus(ctx, userCred, api.LB_STATUS_DELETING, "")
	params := jsonutils.NewDict()
	deleteEip := jsonutils.QueryBoolean(data, "delete_eip", false)
	if deleteEip {
		params.Set("delete_eip", jsonutils.JSONTrue)
	}
	return lb.StartLoadBalancerDeleteTask(ctx, userCred, params, "")
}

func (lb *SLoadbalancer) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	listeners := []SLoadbalancerListener{}
	q := LoadbalancerListenerManager.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(LoadbalancerListenerManager, q, &listeners); err != nil {
		return nil, err
	}
	return listeners, nil
}

func (lb *SLoadbalancer) GetLoadbalancerBackendgroups() ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := LoadbalancerBackendGroupManager.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(LoadbalancerBackendGroupManager, q, &lbbgs); err != nil {
		return nil, err
	}
	return lbbgs, nil
}

func (lb *SLoadbalancer) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(lb.NetworkId) > 0 {
		req := &SLoadbalancerNetworkDeleteData{
			loadbalancer: lb,
		}
		err := LoadbalancernetworkManager.DeleteLoadbalancerNetwork(ctx, userCred, req)
		if err != nil {
			return errors.Wrapf(err, "DeleteLoadbalancerNetwork")
		}
	}
	lbbgs, err := lb.GetLoadbalancerBackendgroups()
	if err != nil {
		return errors.Wrapf(err, "GetLoadbalancerBackendgroups")
	}
	for i := range lbbgs {
		err = lbbgs[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "RealDelete lbbg %s", lbbgs[i].Id)
		}
	}
	listeners, err := lb.GetLoadbalancerListeners()
	if err != nil {
		return errors.Wrapf(err, "GetLoadbalancerListeners")
	}
	for i := range listeners {
		err = listeners[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "RealDelete listener %s", listeners[i].Id)
		}
	}
	return lb.SVirtualResourceBase.Delete(ctx, userCred)
}

func (lb *SLoadbalancer) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerManager) SyncLoadbalancers(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	region *SCloudregion,
	lbs []cloudprovider.ICloudLoadbalancer,
	xor bool,
) ([]SLoadbalancer, []cloudprovider.ICloudLoadbalancer, compare.SyncResult) {
	lockman.LockRawObject(ctx, man.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, man.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))

	localLbs := []SLoadbalancer{}
	remoteLbs := []cloudprovider.ICloudLoadbalancer{}
	syncResult := compare.SyncResult{}

	dbLbs, err := region.GetManagedLoadbalancers(provider.Id)
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
	if !xor {
		for i := 0; i < len(commondb); i++ {
			err = commondb[i].syncWithCloudLoadbalancer(ctx, userCred, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			localLbs = append(localLbs, commondb[i])
			remoteLbs = append(remoteLbs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		lb, err := region.newFromCloudLoadbalancer(ctx, userCred, provider, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localLbs = append(localLbs, *lb)
		remoteLbs = append(remoteLbs, added[i])
		syncResult.Add()
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

func (region *SCloudregion) newFromCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudLoadbalancer) (*SLoadbalancer, error) {
	lb := SLoadbalancer{}
	lb.SetModelManager(LoadbalancerManager, &lb)

	lb.ManagerId = provider.Id
	lb.CloudregionId = region.Id
	lb.Address = ext.GetAddress()
	lb.AddressType = ext.GetAddressType()
	lb.NetworkType = ext.GetNetworkType()

	lb.Status = ext.GetStatus()
	lb.LoadbalancerSpec = ext.GetLoadbalancerSpec()
	lb.ChargeType = ext.GetChargeType()
	lb.EgressMbps = ext.GetEgressMbps()
	lb.ExternalId = ext.GetGlobalId()
	lbNetworkIds := getExtLbNetworkIds(ext, lb.ManagerId)
	lb.NetworkId = strings.Join(lbNetworkIds, ",")

	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		lb.CreatedAt = createdAt
	}

	// vpc
	if vpcId := ext.GetVpcId(); len(vpcId) > 0 {
		if vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		}); err == nil && vpc != nil {
			lb.VpcId = vpc.GetId()
		}
	}
	zones, err := region.GetZones()
	if err != nil {
		return nil, errors.Wrapf(err, "GetZones")
	}

	if zoneId := ext.GetZoneId(); len(zoneId) > 0 {
		for i := range zones {
			if strings.HasSuffix(zones[i].ExternalId, zoneId) {
				lb.ZoneId = zones[i].Id
				break
			}
		}
	}

	if zoneId := ext.GetZone1Id(); len(zoneId) > 0 {
		for i := range zones {
			if strings.HasSuffix(zones[i].ExternalId, zoneId) {
				lb.Zone1 = zones[i].Id
				break
			}
		}
	}

	syncOwnerId := provider.GetOwnerId()

	err = func() error {
		lockman.LockRawObject(ctx, LoadbalancerManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, LoadbalancerManager.Keyword(), "name")

		var err error
		lb.Name, err = db.GenerateName(ctx, LoadbalancerManager, syncOwnerId, ext.GetName())
		if err != nil {
			return err
		}

		return LoadbalancerManager.TableSpec().Insert(ctx, &lb)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, &lb, ext, false)
	SyncCloudProject(ctx, userCred, &lb, syncOwnerId, ext, provider)

	db.OpsLog.LogEvent(&lb, db.ACT_CREATE, lb.GetShortDesc(ctx), userCred)

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &lb,
		Action: notifyclient.ActionSyncCreate,
	})

	lb.syncLoadbalancerNetwork(ctx, userCred, lbNetworkIds)
	return &lb, nil
}

func (lb *SLoadbalancer) syncRemoveCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	err := lb.SDeletePreventableResourceBase.DeletePreventionOff(lb, userCred)
	if err != nil {
		return err
	}
	err = lb.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		return lb.SetStatus(ctx, userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	}
	err = lb.DeleteEip(ctx, userCred, false)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lb,
		Action: notifyclient.ActionSyncDelete,
	})
	return lb.RealDelete(ctx, userCred)
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

func (self *SLoadbalancer) DeleteEip(ctx context.Context, userCred mcclient.TokenCredential, autoDelete bool) error {
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
			return errors.Wrap(err, "RealDelete")
		}
	} else {
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("Dissociate eip on delete server fail %s", err)
			return errors.Wrap(err, "Dissociate")
		}
		if autoDelete {
			err = eip.RealDelete(ctx, userCred)
			if err != nil {
				log.Errorf("Delete eip on delete server fail %s", err)
				return errors.Wrap(err, "RealDelete")
			}
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
		region, _ := self.GetRegion()
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, region, provider.GetOwnerId())
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
				region, _ := self.GetRegion()
				neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, region, provider.GetOwnerId())
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

func (lb *SLoadbalancer) GetSecurityGroups() ([]SSecurityGroup, error) {
	q := SecurityGroupManager.Query()
	sq := LoadbalancerSecurityGroupManager.Query("secgroup_id").Equals("loadbalancer_id", lb.Id)
	q = q.In("id", sq.SubQuery())
	ret := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (lb *SLoadbalancer) removeSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, groupIds []string) {
	params := []interface{}{time.Now(), lb.Id}
	placeholder := []string{}
	for _, id := range groupIds {
		params = append(params, id)
		placeholder = append(placeholder, "?")
	}
	sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set deleted = 1, deleted_at = ? where loadbalancer_id = ? and secgroup_id in (%s)",
			LoadbalancerSecurityGroupManager.TableSpec().Name(), strings.Join(placeholder, ","),
		), params...,
	)
}

func (lb *SLoadbalancer) addSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, groupIds []string) {
	for _, groupId := range groupIds {
		lbsec := &SLoadbalancerSecurityGroup{}
		lbsec.LoadbalancerId = lb.Id
		lbsec.SecgroupId = groupId
		lbsec.SetModelManager(LoadbalancerSecurityGroupManager, lbsec)
		LoadbalancerSecurityGroupManager.TableSpec().Insert(ctx, lbsec)
	}
}

func (lb *SLoadbalancer) SyncSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, groupIds []string) compare.SyncResult {
	result := compare.SyncResult{}

	dbSecs, err := lb.GetSecurityGroups()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetSecurityGroups"))
		return result
	}

	remote := []SSecurityGroup{}
	{
		q := SecurityGroupManager.Query().In("external_id", groupIds).Equals("manager_id", lb.ManagerId)
		err := db.FetchModelObjects(SecurityGroupManager, q, &remote)
		if err != nil {
			result.Error(errors.Wrapf(err, "FetchModelObjects"))
			return result
		}
	}

	removed := []string{}
	common := []string{}
	for _, sec := range dbSecs {
		if !utils.IsInStringArray(sec.ExternalId, groupIds) {
			removed = append(removed, sec.Id)
			continue
		}
		common = append(common, sec.Id)
	}

	added := []string{}
	for _, sec := range remote {
		if !utils.IsInStringArray(sec.Id, common) && !utils.IsInStringArray(sec.Id, added) {
			added = append(added, sec.Id)
		}
	}

	lb.removeSecurityGroups(ctx, userCred, removed)
	lb.addSecurityGroups(ctx, userCred, added)

	result.AddCnt = len(added)
	result.UpdateCnt = len(common)
	result.DelCnt = len(removed)

	return result
}

func (lb *SLoadbalancer) SyncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancer, provider *SCloudprovider) error {
	err := lb.syncWithCloudLoadbalancer(ctx, userCred, ext)
	if err != nil {
		return errors.Wrapf(err, "syncWithCloudLoadbalancer")
	}
	syncLbPeripherals(ctx, userCred, provider, lb, ext)
	return nil
}

func (lb *SLoadbalancer) syncWithCloudLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancer) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	diff, err := db.Update(lb, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(lb, ext.GetName())
			if len(newName) > 0 {
				lb.Name = newName
			}
		}

		lb.Address = ext.GetAddress()
		lb.AddressType = ext.GetAddressType()
		lb.Status = ext.GetStatus()
		lb.LoadbalancerSpec = ext.GetLoadbalancerSpec()
		lb.EgressMbps = ext.GetEgressMbps()
		lb.ChargeType = ext.GetChargeType()
		lbNetworkIds := getExtLbNetworkIds(ext, lb.ManagerId)
		lb.NetworkId = strings.Join(lbNetworkIds, ",")
		if len(lb.VpcId) == 0 {
			if vpcId := ext.GetVpcId(); len(vpcId) > 0 {
				vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, vpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					return q.Equals("manager_id", lb.ManagerId)
				})
				if err != nil {
					log.Errorf("fetch vpc %s error: %v", vpcId, err)
				} else {
					lb.VpcId = vpc.GetId()
				}
			}
		}

		if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
			lb.CreatedAt = createdAt
		}

		return nil
	})

	db.OpsLog.LogSyncUpdate(lb, diff, userCred)

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lb,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	networkIds := getExtLbNetworkIds(ext, lb.ManagerId)
	if account := lb.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, lb, ext, account.ReadOnly)
	}
	provider := lb.GetCloudprovider()
	SyncCloudProject(ctx, userCred, lb, provider.GetOwnerId(), ext, provider)
	lb.syncLoadbalancerNetwork(ctx, userCred, networkIds)

	return err
}

func (manager *SLoadbalancerManager) FetchByExternalId(providerId string, extId string) (*SLoadbalancer, error) {
	ret := []SLoadbalancer{}
	vpcs := VpcManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
	q = q.Filter(sqlchemy.Equals(vpcs.Field("manager_id"), providerId))
	q = q.Equals("external_id", extId)
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
	q := manager.Query().IsNotEmpty("backend_group_id")
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

func (man *SLoadbalancerManager) TotalCount(
	ctx context.Context,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (int, error) {
	q := man.Query()
	q = db.ObjectIdQueryWithPolicyResult(ctx, q, man, policyResult)
	q = scopeOwnerIdFilter(q, scope, ownerId)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, nil, q.Field("zone_id"), q.Field("manager_id"), nil, nil)
	return q.CountWithError()
}

func (lb *SLoadbalancer) GetQuotaKeys() quotas.IQuotaKeys {
	region, _ := lb.GetRegion()
	return fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		lb.GetOwnerId(),
		region,
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
		self.SetStatus(ctx, userCred, api.LB_UPDATE_TAGS, "StartRemoteUpdateTask")
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SLoadbalancer) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := self.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

func (lb *SLoadbalancer) IsEipAssociable() error {
	if !utils.IsInStringArray(lb.Status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "cannot associate eip in status %s", lb.Status)
	}

	err := ValidateAssociateEip(lb)
	if err != nil {
		return errors.Wrap(err, "ValidateAssociateEip")
	}

	eip, err := lb.GetEip()
	if err != nil {
		return errors.Wrap(err, "GetElasticIp")
	}
	if eip != nil {
		return httperrors.NewInvalidStatusError("already associate with eip")
	}
	return nil
}

// 绑定弹性公网IP, 仅支持kvm
func (lb *SLoadbalancer) PerformAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LoadbalancerAssociateEipInput) (jsonutils.JSONObject, error) {
	if lb.IsManaged() {
		return nil, httperrors.NewUnsupportOperationError("not support managed lb")
	}
	err := lb.IsEipAssociable()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	eipStr := input.EipId
	if len(eipStr) == 0 {
		return nil, httperrors.NewMissingParameterError("eip_id")
	}
	eipObj, err := ElasticipManager.FetchByIdOrName(ctx, userCred, eipStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("eip %s not found", eipStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	eip := eipObj.(*SElasticip)
	eipRegion, err := eip.GetRegion()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "eip.GetRegion"))
	}
	instRegion, _ := lb.GetRegion()

	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	if eip.IsAssociated() {
		return nil, httperrors.NewConflictError("eip has been associated")
	}

	if eipRegion.Id != instRegion.Id {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different region")
	}

	if len(eip.NetworkId) > 0 {
		nets, err := lb.GetNetworks()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetNetworks"))
		}
		for _, net := range nets {
			if net.Id == eip.NetworkId {
				return nil, httperrors.NewInputParameterError("cannot associate eip with same network")
			}
		}
	}

	eipZone, _ := eip.GetZone()
	if eipZone != nil {
		insZone, _ := lb.GetZone()
		if eipZone.Id != insZone.Id {
			return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different zone")
		}
	}

	if lb.ManagerId != eip.ManagerId {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different provider")
	}

	err = eip.AssociateLoadbalancer(ctx, userCred, lb)
	if err != nil {
		return nil, errors.Wrap(err, "AssociateLoadbalancer")
	}

	_, err = db.Update(lb, func() error {
		lb.Address = eip.IpAddr
		lb.AddressType = api.LB_ADDR_TYPE_INTERNET
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "set loadbalancer address")
	}

	return nil, nil
}

// 解绑弹性公网IP，仅支持kvm
func (lb *SLoadbalancer) PerformDissociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LoadbalancerDissociateEipInput) (jsonutils.JSONObject, error) {
	if lb.IsManaged() {
		return nil, httperrors.NewUnsupportOperationError("not support managed lb")
	}

	eip, err := lb.GetEip()
	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if eip == nil {
		return nil, httperrors.NewInvalidStatusError("No eip to dissociate")
	}

	err = db.IsObjectRbacAllowed(ctx, eip, userCred, policy.PolicyActionGet)
	if err != nil {
		return nil, errors.Wrap(err, "eip is not accessible")
	}

	lbnet, err := LoadbalancernetworkManager.FetchFirstByLbId(ctx, lb.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "LoadbalancernetworkManager.FetchFirstByLbId(%s)", lb.Id)
	}
	if _, err := db.Update(lb, func() error {
		lb.Address = lbnet.IpAddr
		lb.AddressType = api.LB_ADDR_TYPE_INTRANET
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "db.Update")
	}

	autoDelete := (input.AudoDelete != nil && *input.AudoDelete)
	err = lb.DeleteEip(ctx, userCred, autoDelete)
	if err != nil {
		return nil, errors.Wrap(err, "DeleteEip")
	}

	return nil, nil
}

func (lb *SLoadbalancer) PerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LoadbalancerCreateEipInput) (jsonutils.JSONObject, error) {
	var (
		region, _    = lb.GetRegion()
		regionDriver = region.GetDriver()

		bw            = input.Bandwidth
		chargeType    = input.ChargeType
		bgpType       = input.BgpType
		autoDellocate = (input.AutoDellocate != nil && *input.AutoDellocate)
	)

	err := lb.IsEipAssociable()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if chargeType == "" {
		chargeType = regionDriver.GetEipDefaultChargeType()
	}

	if chargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		if bw == 0 {
			return nil, httperrors.NewMissingParameterError("bandwidth")
		}
	}

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys := lb.GetQuotaKeys()
	eipPendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, eipPendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Out of eip quota: %s", err)
	}

	eip, err := ElasticipManager.NewEipForVMOnHost(ctx, userCred, &NewEipForVMOnHostArgs{
		Bandwidth:     int(bw),
		BgpType:       bgpType,
		ChargeType:    chargeType,
		AutoDellocate: autoDellocate,

		Loadbalancer: lb,
		PendingUsage: eipPendingUsage,
	})
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, false)
		return nil, httperrors.NewGeneralError(err)
	}

	opts := api.ElasticipAssociateInput{
		InstanceId:         lb.Id,
		InstanceExternalId: lb.ExternalId,
		InstanceType:       api.EIP_ASSOCIATE_TYPE_LOADBALANCER,
	}

	err = eip.AllocateAndAssociateInstance(ctx, userCred, lb, opts, "")
	if err != nil {
		return nil, errors.Wrap(err, "AllocateAndAssociateInstance")
	}

	return nil, nil
}

func (man *SLoadbalancerManager) InitializeData() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set deleted = true where pending_deleted = true",
			man.TableSpec().Name(),
		),
	)
	return err
}
