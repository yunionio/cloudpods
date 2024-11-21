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
	"sort"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SVpcManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SGlobalVpcResourceBaseManager
}

var VpcManager *SVpcManager

func init() {
	VpcManager = &SVpcManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SVpc{},
			"vpcs_tbl",
			"vpc",
			"vpcs",
		),
	}
	VpcManager.SetVirtualObject(VpcManager)
}

type SVpc struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`

	SGlobalVpcResourceBase `width:"36" charset:"ascii" list:"user" json:"globalvpc_id"`

	// 是否是默认VPC
	// example: true
	IsDefault bool `default:"false" list:"domain" create:"domain_optional"`

	// CIDR地址段
	// example: 192.168.222.0/24
	CidrBlock string `charset:"ascii" nullable:"true" list:"domain" create:"domain_optional" update:"domain"`

	// CIDR for IPv6
	CidrBlock6 string `charset:"ascii" nullable:"true" list:"domain" create:"domain_optional" update:"domain"`

	// Vpc外网访问模式
	ExternalAccessMode string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	// Can it be connected directly
	Direct bool `default:"false" list:"user" update:"user"`
}

func (manager *SVpcManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
		{GlobalVpcManager},
	}
}

func (svpc *SVpc) GetCloudRegionId() string {
	if len(svpc.CloudregionId) == 0 {
		return api.DEFAULT_REGION_ID
	} else {
		return svpc.CloudregionId
	}
}

func (svpc *SVpc) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		svpc.Id = idstr
	}
	return svpc.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (svpc *SVpc) getNatgatewayQuery() *sqlchemy.SQuery {
	return NatGatewayManager.Query().Equals("vpc_id", svpc.Id)
}

func (svpc *SVpc) GetNatgatewayCount() (int, error) {
	return svpc.getNatgatewayQuery().CountWithError()
}

func (svpc *SVpc) GetDnsZones() ([]SDnsZone, error) {
	sq := DnsZoneVpcManager.Query("dns_zone_id").Equals("vpc_id", svpc.Id)
	q := DnsZoneManager.Query().In("id", sq.SubQuery())
	dnsZones := []SDnsZone{}
	err := db.FetchModelObjects(DnsZoneManager, q, &dnsZones)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return dnsZones, nil
}

func (svpc *SVpc) GetInterVpcNetworks() ([]SInterVpcNetwork, error) {
	sq := InterVpcNetworkVpcManager.Query("inter_vpc_network_id").Equals("vpc_id", svpc.Id)
	q := InterVpcNetworkManager.Query().In("id", sq.SubQuery())
	vpcNetworks := []SInterVpcNetwork{}
	err := db.FetchModelObjects(InterVpcNetworkManager, q, &vpcNetworks)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return vpcNetworks, nil
}

func (svpc *SVpc) GetDnsZoneCount() (int, error) {
	sq := DnsZoneVpcManager.Query("dns_zone_id").Equals("vpc_id", svpc.Id)
	q := DnsZoneManager.Query().In("id", sq.SubQuery())
	return q.CountWithError()
}

func (svpc *SVpc) GetNatgateways() ([]SNatGateway, error) {
	nats := []SNatGateway{}
	err := db.FetchModelObjects(NatGatewayManager, svpc.getNatgatewayQuery(), &nats)
	if err != nil {
		return nil, err
	}
	return nats, nil
}

func (svpc *SVpc) getRequesterVpcPeeringConnectionQuery() *sqlchemy.SQuery {
	return VpcPeeringConnectionManager.Query().Equals("vpc_id", svpc.Id)
}

func (svpc *SVpc) getAccepterVpcPeeringConnectionQuery() *sqlchemy.SQuery {
	return VpcPeeringConnectionManager.Query().Equals("peer_vpc_id", svpc.Id)
}

func (svpc *SVpc) GetRequesterVpcPeeringConnections() ([]SVpcPeeringConnection, error) {
	vpcPC := []SVpcPeeringConnection{}
	err := db.FetchModelObjects(VpcPeeringConnectionManager, svpc.getRequesterVpcPeeringConnectionQuery(), &vpcPC)
	if err != nil {
		return nil, err
	}
	return vpcPC, nil
}

func (svpc *SVpc) GetAccepterVpcPeeringConnections() ([]SVpcPeeringConnection, error) {
	vpcPC := []SVpcPeeringConnection{}
	err := db.FetchModelObjects(VpcPeeringConnectionManager, svpc.getAccepterVpcPeeringConnectionQuery(), &vpcPC)
	if err != nil {
		return nil, err
	}
	return vpcPC, nil
}
func (svpc *SVpc) GetVpcPeeringConnectionCount() (int, error) {
	q := svpc.getRequesterVpcPeeringConnectionQuery()
	requesterPeerCount, err := q.CountWithError()
	if err != nil {
		return 0, err
	}
	q = svpc.getRequesterVpcPeeringConnectionQuery()
	accepterPeerCount, err := q.CountWithError()
	if err != nil {
		return 0, err
	}
	return requesterPeerCount + accepterPeerCount, nil
}

func (svpc *SVpc) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcUpdateInput) (api.VpcUpdateInput, error) {
	var err error

	if input.ExternalAccessMode != "" {
		if !utils.IsInArray(input.ExternalAccessMode, api.VPC_EXTERNAL_ACCESS_MODES) {
			return input, httperrors.NewInputParameterError("invalid external_access_mode %q, want %s",
				input.ExternalAccessMode, api.VPC_EXTERNAL_ACCESS_MODES)
		}
	}

	if len(input.CidrBlock) > 0 {
		input.CidrBlock, err = validateCidrBlock(input.CidrBlock)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid cidr_block %s", err)
		}
	}
	if len(input.CidrBlock6) > 0 {
		input.CidrBlock6, err = validateCidrBlock6(input.CidrBlock6)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid ipv6 cidr_block %s", err)
		}
	}

	if _, err = svpc.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput); err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (svpc *SVpc) ValidateDeleteCondition(ctx context.Context, info *api.VpcDetails) error {
	if svpc.Id == api.DEFAULT_VPC_ID {
		return httperrors.NewProtectedResourceError("not allow to delete default vpc")
	}

	if gotypes.IsNil(info) {
		info = &api.VpcDetails{}
		usage, err := VpcManager.TotalResourceCount([]string{svpc.Id})
		if err != nil {
			return err
		}
		info.VpcUsage, _ = usage[svpc.Id]
	}

	if info.NetworkCount > 0 {
		return httperrors.NewNotEmptyError("VPC not empty, please delete network first")
	}
	if info.NatgatewayCount > 0 {
		return httperrors.NewNotEmptyError("VPC not empty, please delete nat gateway first")
	}
	if info.RequestVpcPeerCount > 0 {
		return httperrors.NewNotEmptyError("VPC not empty, please delete vpc peering first")
	}

	return svpc.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (svpc *SVpc) getWireQuery() *sqlchemy.SQuery {
	wires := WireManager.Query()
	if svpc.Id == api.DEFAULT_VPC_ID {
		return wires.Filter(sqlchemy.OR(sqlchemy.IsNull(wires.Field("vpc_id")),
			sqlchemy.IsEmpty(wires.Field("vpc_id")),
			sqlchemy.Equals(wires.Field("vpc_id"), svpc.Id)))
	} else {
		return wires.Equals("vpc_id", svpc.Id)
	}
}

func (svpc *SVpc) GetWireCount() (int, error) {
	q := svpc.getWireQuery()
	return q.CountWithError()
}

func (svpc *SVpc) GetWires() ([]SWire, error) {
	wires := make([]SWire, 0)
	q := svpc.getWireQuery()
	err := db.FetchModelObjects(WireManager, q, &wires)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return wires, nil
}

func (manager *SVpcManager) getVpcExternalIdForClassicNetwork(regionId, cloudproviderId string) string {
	return fmt.Sprintf("%s-%s", regionId, cloudproviderId)
}

func (manager *SVpcManager) GetOrCreateVpcForClassicNetwork(ctx context.Context, cloudprovider *SCloudprovider, region *SCloudregion) (*SVpc, error) {
	externalId := manager.getVpcExternalIdForClassicNetwork(region.Id, cloudprovider.Id)
	_vpc, err := db.FetchByExternalIdAndManagerId(manager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", cloudprovider.Id)
	})
	if err == nil {
		return _vpc.(*SVpc), nil
	}
	if errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrapf(err, "db.FetchByExternalId %s", externalId)
	}
	vpc := &SVpc{}
	vpc.IsDefault = false
	vpc.CloudregionId = region.Id
	vpc.SetModelManager(manager, vpc)
	vpc.Name = api.CLASSIC_VPC_NAME
	vpc.IsEmulated = true
	vpc.SetEnabled(false)
	vpc.Status = api.VPC_STATUS_UNAVAILABLE
	vpc.ExternalId = externalId
	vpc.ManagerId = cloudprovider.Id
	err = manager.TableSpec().Insert(ctx, vpc)
	if err != nil {
		return nil, errors.Wrap(err, "Insert vpc for classic network")
	}
	return vpc, nil
}

func (svpc *SVpc) getNetworkQuery() *sqlchemy.SQuery {
	q := NetworkManager.Query()
	wireQ := svpc.getWireQuery().SubQuery()
	q = q.In("wire_id", wireQ.Query(wireQ.Field("id")).SubQuery())
	return q
}

func (svpc *SVpc) GetNetworks() ([]SNetwork, error) {
	q := svpc.getNetworkQuery()
	nets := make([]SNetwork, 0, 5)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return nets, nil
}

func (svpc *SVpc) GetNetworksByProvider(provider string) ([]SNetwork, error) {
	q := NetworkManager.Query()
	wireQ := svpc.getWireQuery()
	if provider == api.CLOUD_PROVIDER_ONECLOUD {
		wireQ = wireQ.IsNullOrEmpty("manager_id")
	} else {
		account := CloudaccountManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		subq := providers.Query(providers.Field("id"))
		subq = subq.Join(account, sqlchemy.Equals(
			account.Field("id"), providers.Field("cloudaccount_id"),
		))
		subq = subq.Filter(sqlchemy.Equals(account.Field("provider"), provider))
		wireQ = wireQ.Filter(sqlchemy.In(wireQ.Field("manager_id"), subq.SubQuery()))
	}

	wireSubQ := wireQ.SubQuery()
	q = q.In("wire_id", wireSubQ.Query(wireSubQ.Field("id")).SubQuery())

	nets := make([]SNetwork, 0, 5)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return nets, nil
}

func (svpc *SVpc) GetNetworkByExtId(extId string) (*SNetwork, error) {
	network, err := db.FetchByExternalIdAndManagerId(NetworkManager, extId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		wireQ := svpc.getWireQuery().SubQuery()
		q = q.In("wire_id", wireQ.Query(wireQ.Field("id")).SubQuery())
		return q
	})
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworkByExtId(%s)", extId)
	}
	return network.(*SNetwork), nil
}

func (svpc *SVpc) GetNetworkCount() (int, error) {
	q := svpc.getNetworkQuery()
	return q.CountWithError()
}

func (svpc *SVpc) GetRouteTableQuery() *sqlchemy.SQuery {
	return RouteTableManager.Query().Equals("vpc_id", svpc.Id)
}

func (svpc *SVpc) GetRouteTables() []SRouteTable {
	q := svpc.GetRouteTableQuery()
	routes := []SRouteTable{}
	db.FetchModelObjects(RouteTableManager, q, &routes)
	return routes
}

func (svpc *SVpc) GetRouteTableCount() (int, error) {
	return svpc.GetRouteTableQuery().CountWithError()
}

/*func (svpc *SVpc) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := svpc.GetRegion()
	provider := svpc.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}*/

func (svpc *SVpc) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(svpc.CloudregionId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionManager.FetchById")
	}
	return region.(*SCloudregion), nil
}

func (svpc *SVpc) getZoneByExternalId(externalId string) (*SZone, error) {
	region, err := svpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "getZoneByExternalId.GetRegion")
	}
	zones := []SZone{}
	q := ZoneManager.Query().Equals("cloudregion_id", region.Id).Equals("external_id", externalId)
	err = db.FetchModelObjects(ZoneManager, q, &zones)
	if err != nil {
		return nil, errors.Wrapf(err, "getZoneByExternalId.FetchModelObjects")
	}
	if len(zones) == 1 {
		return &zones[0], nil
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("failed to found zone by externalId %s in cloudregion %s(%s)", externalId, region.Name, region.Id)
	}
	return nil, fmt.Errorf("found %d duplicate zones by externalId %s in cloudregion %s(%s)", len(zones), externalId, region.Name, region.Id)
}

type SVpcUsageCount struct {
	Id string
	api.VpcUsage
}

func (nm *SVpcManager) query(manager db.IModelManager, field string, netIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("vpc_id"),
		sqlchemy.COUNT(field),
	).In("vpc_id", netIds).GroupBy(sq.Field("vpc_id")).SubQuery()
}

func (manager *SVpcManager) TotalResourceCount(vpcIds []string) (map[string]api.VpcUsage, error) {
	// wire
	wireSQ := manager.query(WireManager, "wire_cnt", vpcIds, nil)
	// network
	networkSQ := manager.query(NetworkManager, "network_cnt", vpcIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id").Label("network_id"),
			sq.Field("wire_id").Label("wire_id"),
			wires.Field("vpc_id").Label("vpc_id"),
		).LeftJoin(wires, sqlchemy.Equals(sq.Field("wire_id"), wires.Field("id")))
	})

	// routetable
	rtbSQ := manager.query(RouteTableManager, "routetable_cnt", vpcIds, nil)

	// nat
	natSQ := manager.query(NatGatewayManager, "natgateway_cnt", vpcIds, nil)

	// dns
	dnsSQ := manager.query(DnsZoneManager, "dns_zone_cnt", vpcIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		dv := DnsZoneVpcManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			dv.Field("vpc_id").Label("vpc_id"),
		).LeftJoin(dv, sqlchemy.Equals(sq.Field("id"), dv.Field("dns_zone_id")))
	})

	// dns
	rvpSQ := manager.query(VpcPeeringConnectionManager, "request_vpc_peer_cnt", vpcIds, nil)
	avpSQ := manager.query(VpcPeeringConnectionManager, "accept_vpc_peer_cnt", vpcIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			sq.Field("peer_vpc_id").Label("vpc_id"),
		)
	})

	vpcs := manager.Query().SubQuery()
	vpcQ := vpcs.Query(
		sqlchemy.SUM("wire_count", wireSQ.Field("wire_cnt")),
		sqlchemy.SUM("network_count", networkSQ.Field("network_cnt")),
		sqlchemy.SUM("routetable_count", rtbSQ.Field("routetable_cnt")),
		sqlchemy.SUM("natgateway_count", natSQ.Field("natgateway_cnt")),
		sqlchemy.SUM("dns_zone_count", dnsSQ.Field("dns_zone_cnt")),
		sqlchemy.SUM("request_vpc_peer_count", rvpSQ.Field("request_vpc_peer_cnt")),
		sqlchemy.SUM("accept_vpc_peer_count", avpSQ.Field("accept_vpc_peer_cnt")),
	)

	vpcQ.AppendField(vpcQ.Field("id"))

	vpcQ = vpcQ.LeftJoin(wireSQ, sqlchemy.Equals(vpcQ.Field("id"), wireSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(networkSQ, sqlchemy.Equals(vpcQ.Field("id"), networkSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(rtbSQ, sqlchemy.Equals(vpcQ.Field("id"), rtbSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(natSQ, sqlchemy.Equals(vpcQ.Field("id"), natSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(dnsSQ, sqlchemy.Equals(vpcQ.Field("id"), dnsSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(rvpSQ, sqlchemy.Equals(vpcQ.Field("id"), rvpSQ.Field("vpc_id")))
	vpcQ = vpcQ.LeftJoin(avpSQ, sqlchemy.Equals(vpcQ.Field("id"), avpSQ.Field("vpc_id")))

	vpcQ = vpcQ.Filter(sqlchemy.In(vpcQ.Field("id"), vpcIds)).GroupBy(vpcQ.Field("id"))

	vpcCount := []SVpcUsageCount{}
	err := vpcQ.All(&vpcCount)
	if err != nil {
		return nil, errors.Wrapf(err, "vpcQ.All")
	}

	result := map[string]api.VpcUsage{}
	for i := range vpcCount {
		result[vpcCount[i].Id] = vpcCount[i].VpcUsage
	}

	return result, nil
}

func (manager *SVpcManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.VpcDetails {
	rows := make([]api.VpcDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	globalVpcRows := manager.SGlobalVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.VpcDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			CloudregionResourceInfo:                regionRows[i],
			GlobalVpcResourceInfo:                  globalVpcRows[i],
		}
		vpc := objs[i].(*SVpc)
		vpcIds[i] = vpc.Id
	}
	usage, err := manager.TotalResourceCount(vpcIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].VpcUsage, _ = usage[vpcIds[i]]
	}
	return rows
}

func (svpc *SVpc) setDefault(def bool) error {
	var err error
	if svpc.IsDefault != def {
		_, err = db.Update(svpc, func() error {
			svpc.IsDefault = def
			return nil
		})
	}
	return err
}

func (manager *SVpcManager) SyncVPCs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	region *SCloudregion,
	vpcs []cloudprovider.ICloudVpc,
	xor bool,
) ([]SVpc, []cloudprovider.ICloudVpc, compare.SyncResult) {
	lockman.LockRawObject(ctx, manager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))

	localVPCs := make([]SVpc, 0)
	remoteVPCs := make([]cloudprovider.ICloudVpc, 0)
	syncResult := compare.SyncResult{}

	dbVPCs, err := region.GetCloudproviderVpcs(provider.Id)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := range dbVPCs {
		if taskman.TaskManager.IsInTask(&dbVPCs[i]) {
			syncResult.Error(fmt.Errorf("object in task"))
			return nil, nil, syncResult
		}
	}

	removed := make([]SVpc, 0)
	commondb := make([]SVpc, 0)
	commonext := make([]cloudprovider.ICloudVpc, 0)
	added := make([]cloudprovider.ICloudVpc, 0)

	err = compare.CompareSets(dbVPCs, vpcs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudVpc(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].SyncWithCloudVpc(ctx, userCred, commonext[i], provider)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
		}
		localVPCs = append(localVPCs, commondb[i])
		remoteVPCs = append(remoteVPCs, commonext[i])
		syncResult.Update()
	}
	for i := 0; i < len(added); i += 1 {
		newVpc, err := manager.newFromCloudVpc(ctx, userCred, added[i], provider, region)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localVPCs = append(localVPCs, *newVpc)
		remoteVPCs = append(remoteVPCs, added[i])
		syncResult.Add()
	}

	return localVPCs, remoteVPCs, syncResult
}

func (svpc *SVpc) syncRemoveCloudVpc(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, svpc)
	defer lockman.ReleaseObject(ctx, svpc)

	if VpcManager.getVpcExternalIdForClassicNetwork(svpc.CloudregionId, svpc.ManagerId) == svpc.ExternalId { //为经典网络虚拟的vpc
		return nil
	}

	err := svpc.purge(ctx, userCred)
	if err == nil {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    svpc,
			Action: notifyclient.ActionSyncDelete,
		})
	}
	return err
}

func (svpc *SVpc) SyncWithCloudVpc(ctx context.Context, userCred mcclient.TokenCredential, extVPC cloudprovider.ICloudVpc, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, svpc, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(svpc, extVPC.GetName())
			if len(newName) > 0 {
				svpc.Name = newName
			}
		}
		svpc.Status = extVPC.GetStatus()
		svpc.CidrBlock = extVPC.GetCidrBlock()
		svpc.CidrBlock6 = extVPC.GetCidrBlock6()
		svpc.IsDefault = extVPC.GetIsDefault()
		svpc.ExternalId = extVPC.GetGlobalId()

		svpc.IsEmulated = extVPC.IsEmulated()
		svpc.ExternalAccessMode = extVPC.GetExternalAccessMode()

		if len(svpc.Description) == 0 {
			svpc.Description = extVPC.GetDescription()
		}
		if createdAt := extVPC.GetCreatedAt(); !createdAt.IsZero() {
			svpc.CreatedAt = createdAt
		}

		if gId := extVPC.GetGlobalVpcId(); len(gId) > 0 {
			gVpc, err := db.FetchByExternalIdAndManagerId(GlobalVpcManager, gId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", svpc.ManagerId)
			})
			if err != nil {
				log.Errorf("FetchGlobalVpc %s error: %v", gId, err)
			} else {
				svpc.GlobalvpcId = gVpc.GetId()
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	//syncMetadata(ctx, userCred, svpc, extVPC)

	if provider != nil {
		SyncCloudDomain(userCred, svpc, provider.GetOwnerId())
		svpc.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogSyncUpdate(svpc, diff, userCred)
	return nil
}

func (manager *SVpcManager) newFromCloudVpc(ctx context.Context, userCred mcclient.TokenCredential, extVPC cloudprovider.ICloudVpc, provider *SCloudprovider, region *SCloudregion) (*SVpc, error) {
	vpc := SVpc{}
	vpc.SetModelManager(manager, &vpc)

	vpc.Status = extVPC.GetStatus()
	vpc.Description = extVPC.GetDescription()
	vpc.ExternalId = extVPC.GetGlobalId()
	vpc.IsDefault = extVPC.GetIsDefault()
	vpc.CidrBlock = extVPC.GetCidrBlock()
	vpc.ExternalAccessMode = extVPC.GetExternalAccessMode()
	vpc.CloudregionId = region.Id
	vpc.ManagerId = provider.Id
	if createdAt := extVPC.GetCreatedAt(); !createdAt.IsZero() {
		vpc.CreatedAt = createdAt
	}
	if gId := extVPC.GetGlobalVpcId(); len(gId) > 0 {
		gVpc, err := db.FetchByExternalIdAndManagerId(GlobalVpcManager, gId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		})
		if err != nil {
			log.Errorf("FetchGlobalVpc %s error: %v", gId, err)
		} else {
			vpc.GlobalvpcId = gVpc.GetId()
		}
	}

	vpc.IsEmulated = extVPC.IsEmulated()

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, extVPC.GetName())
		if err != nil {
			return err
		}
		vpc.Name = newName

		return manager.TableSpec().Insert(ctx, &vpc)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, &vpc, extVPC, false)
	SyncCloudDomain(userCred, &vpc, provider.GetOwnerId())

	if provider != nil {
		vpc.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogEvent(&vpc, db.ACT_CREATE, vpc.GetShortDesc(ctx), userCred)

	return &vpc, nil
}

func (svpc *SVpc) markAllNetworksUnknown(ctx context.Context, userCred mcclient.TokenCredential) error {
	wires, _ := svpc.GetWires()
	if wires == nil || len(wires) == 0 {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wires[i].markNetworkUnknown(ctx, userCred)
	}
	return nil
}

func (svpc *SVpc) SyncRouteTables(ctx context.Context, userCred mcclient.TokenCredential) error {
	ivpc, err := svpc.GetIVpc(ctx)
	if err != nil {
		return errors.Wrap(err, "svpc.GetIVpc()")
	}
	routeTables, err := ivpc.GetIRouteTables()
	if err != nil {
		return errors.Wrapf(err, "GetIRouteTables for vpc %s failed", ivpc.GetId())
	}
	_, _, result := RouteTableManager.SyncRouteTables(ctx, userCred, svpc, routeTables, svpc.GetCloudprovider(), false)
	if result.IsError() {
		return errors.Wrapf(result.AllError(), "RouteTableManager.SyncRouteTables(%s,%s)", jsonutils.Marshal(svpc).String(), jsonutils.Marshal(routeTables).String())
	}
	return nil
}

func (manager *SVpcManager) InitializeData() error {
	if vpcObj, err := manager.FetchById(api.DEFAULT_VPC_ID); err != nil {
		if err == sql.ErrNoRows {
			defVpc := SVpc{}
			defVpc.SetModelManager(VpcManager, &defVpc)

			defVpc.Id = api.DEFAULT_VPC_ID
			defVpc.Name = "Default"
			defVpc.CloudregionId = api.DEFAULT_REGION_ID
			defVpc.Description = "Default VPC"
			defVpc.Status = api.VPC_STATUS_AVAILABLE
			defVpc.IsDefault = true
			defVpc.IsPublic = true
			defVpc.PublicScope = string(rbacscope.ScopeSystem)
			err = manager.TableSpec().Insert(context.TODO(), &defVpc)
			if err != nil {
				log.Errorf("Insert default vpc fail: %s", err)
			}
			return err
		} else {
			return err
		}
	} else {
		vpc := vpcObj.(*SVpc)
		if vpc.Status != api.VPC_STATUS_AVAILABLE || (vpc.PublicScope == string(rbacscope.ScopeSystem) && !vpc.IsPublic) {
			_, err = db.Update(vpc, func() error {
				vpc.Status = api.VPC_STATUS_AVAILABLE
				vpc.IsPublic = true
				return nil
			})
			return err
		}
	}

	if defaultMode := options.Options.DefaultVpcExternalAccessMode; defaultMode == "" {
		options.Options.DefaultVpcExternalAccessMode = api.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW
	} else if !utils.IsInStringArray(defaultMode, api.VPC_EXTERNAL_ACCESS_MODES) {
		return errors.Errorf("invalid DefaultVpcExternalAccessMode, got %s, want %s", defaultMode, api.VPC_EXTERNAL_ACCESS_MODES)
	}
	{
		// initialize default external_access_mode for onecloud vpc
		var vpcs []SVpc
		q := manager.Query().
			IsNullOrEmpty("manager_id").
			IsNullOrEmpty("external_id").
			IsNullOrEmpty("external_access_mode")
		if err := db.FetchModelObjects(manager, q, &vpcs); err != nil {
			return errors.Wrap(err, "fetch onecloud vpc with external_access_mode not set")
		}
		for i := range vpcs {
			vpc := &vpcs[i]
			if _, err := db.Update(vpc, func() error {
				vpc.ExternalAccessMode = options.Options.DefaultVpcExternalAccessMode
				return nil
			}); err != nil {
				return errors.Wrap(err, "db set default external_access_mode")
			}
		}
	}

	{
		// initialize default external access mode for public cloud
		var vpcs []SVpc
		q := manager.Query().
			IsNotEmpty("manager_id").
			IsNotEmpty("external_id").
			IsNullOrEmpty("external_access_mode")
		if err := db.FetchModelObjects(manager, q, &vpcs); err != nil {
			return errors.Wrap(err, "fetch public cloud vpc with external_access_mode not set")
		}
		for i := range vpcs {
			vpc := &vpcs[i]
			if _, err := db.Update(vpc, func() error {
				vpc.ExternalAccessMode = api.VPC_EXTERNAL_ACCESS_MODE_EIP
				return nil
			}); err != nil {
				return errors.Wrap(err, "db set default external_access_mode")
			}
		}
	}

	{
		vpcs := []SVpc{}
		q := manager.Query().IsTrue("is_emulated").IsNotEmpty("external_id").NotEquals("name", "-")
		err := db.FetchModelObjects(manager, q, &vpcs)
		if err != nil {
			return errors.Wrapf(err, "db.FetchModelObjects")
		}
		for i := range vpcs {
			if vpcs[i].ExternalId == manager.getVpcExternalIdForClassicNetwork(vpcs[i].CloudregionId, vpcs[i].ManagerId) {
				_, err = db.Update(&vpcs[i], func() error {
					vpcs[i].Name = "-"
					return nil
				})
				if err != nil {
					return errors.Wrapf(err, "db.Update class vpc name")
				}
			}
		}
	}

	return nil
}

func validateCidrBlock(blocks string) (string, error) {
	var errs []error
	cidrStrs := strings.Split(blocks, ",")
	cidrList4 := make([]string, 0)
	for _, cidrStr := range cidrStrs {
		cidr4, err := netutils.NewIPV4Prefix(cidrStr)
		if err != nil {
			errs = append(errs, errors.Wrap(err, cidrStr))
		} else {
			cidrList4 = append(cidrList4, cidr4.String())
		}
	}
	if len(errs) > 0 {
		return "", errors.NewAggregate(errs)
	}
	sort.Strings(cidrList4)
	return strings.Join(cidrList4, ","), nil
}

func validateCidrBlock6(block6 string) (string, error) {
	var errs []error
	cidrStrs := strings.Split(block6, ",")
	cidrList6 := make([]string, 0)
	for _, cidrStr := range cidrStrs {
		cidr6, err := netutils.NewIPV6Prefix(cidrStr)
		if err != nil {
			errs = append(errs, errors.Wrap(err, cidrStr))
		} else {
			cidrList6 = append(cidrList6, cidr6.String())
		}
	}
	if len(errs) > 0 {
		return "", errors.NewAggregate(errs)
	}
	sort.Strings(cidrList6)
	return strings.Join(cidrList6, ","), nil
}

func (manager *SVpcManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.VpcCreateInput,
) (api.VpcCreateInput, error) {
	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return input, err
	}
	region := regionObj.(*SCloudregion)
	if region.isManaged() {
		_, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return input, err
		}
		if input.ExternalAccessMode == "" {
			input.ExternalAccessMode = api.VPC_EXTERNAL_ACCESS_MODE_EIP
		}
	} else {
		input.Status = api.VPC_STATUS_AVAILABLE
		if input.ExternalAccessMode == "" {
			input.ExternalAccessMode = options.Options.DefaultVpcExternalAccessMode
		}
	}

	// check external access mode
	if !utils.IsInStringArray(input.ExternalAccessMode, api.VPC_EXTERNAL_ACCESS_MODES) {
		return input, httperrors.NewInputParameterError("invalid external_access_mode %q, want %s",
			input.Status, api.VPC_EXTERNAL_ACCESS_MODES)
	}

	if len(input.CidrBlock) > 0 {
		input.CidrBlock, err = validateCidrBlock(input.CidrBlock)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid cidr_block %s", err)
		}
	}
	if len(input.CidrBlock6) > 0 {
		input.CidrBlock6, err = validateCidrBlock6(input.CidrBlock6)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid ipv6 cidr_block %s", err)
		}
	}

	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}

	input, err = region.GetDriver().ValidateCreateVpcData(ctx, userCred, input)
	if err != nil {
		return input, err
	}

	if region.GetDriver().IsVpcCreateNeedInputCidr() && len(input.CidrBlock) == 0 {
		return input, httperrors.NewMissingParameterError("cidr")
	}

	keys := GetVpcQuotaKeysFromCreateInput(ownerId, input)
	quota := &SInfrasQuota{Vpc: 1}
	quota.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "quotas.CheckSetPendingQuota")
	}

	return input, nil
}

func (svpc *SVpc) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.VpcCreateInput{}
	data.Unmarshal(&input)
	pendingUsage := &SInfrasQuota{Vpc: 1}
	keys := GetVpcQuotaKeysFromCreateInput(ownerId, input)
	pendingUsage.SetKeys(keys)
	quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, true)

	defer func() {
		svpc.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	}()

	if len(svpc.ManagerId) == 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    svpc,
			Action: notifyclient.ActionCreate,
		})
		return
	}
	task, err := taskman.TaskManager.NewTask(ctx, "VpcCreateTask", svpc, userCred, nil, "", "", nil)
	if err != nil {
		svpc.SetStatus(ctx, userCred, api.VPC_STATUS_FAILED, errors.Wrapf(err, "NewTask").Error())
		return
	}
	task.ScheduleRun(nil)
}

func (svpc *SVpc) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := svpc.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "GetRegion")
	}
	provider, err := svpc.GetDriver(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (svpc *SVpc) GetIVpc(ctx context.Context) (cloudprovider.ICloudVpc, error) {
	if len(svpc.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := svpc.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "vpc.GetDriver")
	}
	var iregion cloudprovider.ICloudRegion
	if provider.GetFactory().IsOnPremise() {
		iregion, err = provider.GetOnPremiseIRegion()
	} else {
		region, err := svpc.GetRegion()
		if err != nil {
			return nil, err
		}
		iregion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "find iregion")
	}
	ivpc, err := iregion.GetIVpcById(svpc.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIVpcById")
	}
	return ivpc, nil
}

func (svpc *SVpc) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SVpc delete do nothing")
	svpc.SetStatus(ctx, userCred, api.VPC_STATUS_START_DELETE, "")
	return nil
}

func (svpc *SVpc) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if svpc.Id != api.DEFAULT_VPC_ID {
		return svpc.StartDeleteVpcTask(ctx, userCred)
	} else {
		return svpc.RealDelete(ctx, userCred)
	}
}

func (svpc *SVpc) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(svpc, db.ACT_DELOCATE, svpc.GetShortDesc(ctx), userCred)
	svpc.SetStatus(ctx, userCred, api.VPC_STATUS_DELETED, "real delete")

	return svpc.purge(ctx, userCred)
}

func (svpc *SVpc) StartDeleteVpcTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "VpcDeleteTask", svpc, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start vpcdeleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (svpc *SVpc) getPrefix() []netutils.IPV4Prefix {
	if len(svpc.CidrBlock) > 0 {
		ret := []netutils.IPV4Prefix{}
		blocks := strings.Split(svpc.CidrBlock, ",")
		for _, block := range blocks {
			prefix, _ := netutils.NewIPV4Prefix(block)
			ret = append(ret, prefix)
		}
		return ret
	}
	return []netutils.IPV4Prefix{{}}
}

func (svpc *SVpc) getPrefix6() []netutils.IPV6Prefix {
	if len(svpc.CidrBlock6) > 0 {
		ret := []netutils.IPV6Prefix{}
		blocks := strings.Split(svpc.CidrBlock6, ",")
		for _, block := range blocks {
			prefix, _ := netutils.NewIPV6Prefix(block)
			ret = append(ret, prefix)
		}
		return ret
	}
	return []netutils.IPV6Prefix{{}}
}

func (svpc *SVpc) getIPRanges() []netutils.IPV4AddrRange {
	ret := []netutils.IPV4AddrRange{}
	prefs := svpc.getPrefix()
	for _, pref := range prefs {
		ret = append(ret, pref.ToIPRange())
	}
	return ret
}

func (svpc *SVpc) getIP6Ranges() []netutils.IPV6AddrRange {
	ret := []netutils.IPV6AddrRange{}
	prefs := svpc.getPrefix6()
	for _, pref := range prefs {
		ret = append(ret, pref.ToIPRange())
	}
	return ret
}

func (svpc *SVpc) containsIPV4Range(a netutils.IPV4AddrRange) bool {
	ranges := svpc.getIPRanges()
	for i := range ranges {
		if ranges[i].ContainsRange(a) {
			return true
		}
	}
	return false
}

func (svpc *SVpc) containsIPV6Range(a netutils.IPV6AddrRange) bool {
	ranges := svpc.getIP6Ranges()
	for i := range ranges {
		if ranges[i].ContainsRange(a) {
			return true
		}
	}
	return false
}

func (svpc *SVpc) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := svpc.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return nil, err
	}
	provider := svpc.GetCloudprovider()
	if provider != nil {
		if provider.GetEnabled() {
			return nil, httperrors.NewInvalidStatusError("Cannot purge vpc on enabled cloud provider")
		}
	}
	err = svpc.RealDelete(ctx, userCred)
	return nil, err
}

// 列出VPC
func (manager *SVpcManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SGlobalVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.GlobalVpcResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGlobalVpcResourceBaseManager.ListItemFilter")
	}

	if len(query.ExternalAccessMode) > 0 {
		q = q.Equals("external_access_mode", query.ExternalAccessMode)
	}

	if len(query.DnsZoneId) > 0 {
		dnsZone, err := DnsZoneManager.FetchByIdOrName(ctx, userCred, query.DnsZoneId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("dns_zone", query.DnsZoneId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		sq := DnsZoneVpcManager.Query("vpc_id").Equals("dns_zone_id", dnsZone.GetId())
		q = q.In("id", sq.SubQuery())
	}

	if len(query.UsableForInterVpcNetworkId) > 0 {
		_interVpc, err := validators.ValidateModel(ctx, userCred, InterVpcNetworkManager, &query.UsableForInterVpcNetworkId)
		if err != nil {
			return nil, err
		}
		interVpc := _interVpc.(*SInterVpcNetwork)
		sq := InterVpcNetworkVpcManager.Query("vpc_id").Equals("inter_vpc_network_id", interVpc.GetId())
		q = q.NotIn("id", sq.SubQuery())
		account := interVpc.GetCloudaccount()
		if account == nil {
			return nil, httperrors.NewNotSupportedError("not supported for inter vpc network %s", interVpc.Name)
		}
		vpcs := VpcManager.Query().SubQuery()
		managers := CloudproviderManager.Query().SubQuery()
		accounts := CloudaccountManager.Query().SubQuery()
		accUrl := sqlchemy.Equals(accounts.Field("access_url"), account.AccessUrl)
		if len(account.AccessUrl) == 0 {
			accUrl = sqlchemy.IsNullOrEmpty(accounts.Field("access_url"))
		}
		vpcSQ := vpcs.Query(vpcs.Field("id")).Join(managers, sqlchemy.Equals(vpcs.Field("manager_id"), managers.Field("id"))).Join(accounts, sqlchemy.Equals(managers.Field("cloudaccount_id"), accounts.Field("id"))).Filter(
			sqlchemy.AND(
				sqlchemy.Equals(accounts.Field("provider"), account.Provider),
				accUrl,
			),
		)
		q = q.In("id", vpcSQ.SubQuery())
	}

	if len(query.InterVpcNetworkId) > 0 {
		vpcNetwork, err := InterVpcNetworkManager.FetchByIdOrName(ctx, userCred, query.InterVpcNetworkId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("inter_vpc_network", query.InterVpcNetworkId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		sq := InterVpcNetworkVpcManager.Query("vpc_id").Equals("inter_vpc_network_id", vpcNetwork.GetId())
		q = q.In("id", sq.SubQuery())
	}

	usable := (query.Usable != nil && *query.Usable)
	vpcUsable := (query.UsableVpc != nil && *query.UsableVpc)
	if vpcUsable || usable {
		regions := CloudregionManager.Query().SubQuery()
		providerSQ := usableCloudProviders().SubQuery()
		q = q.Join(regions, sqlchemy.Equals(q.Field("cloudregion_id"), regions.Field("id"))).Filter(
			sqlchemy.AND(
				sqlchemy.Equals(regions.Field("status"), api.CLOUD_REGION_STATUS_INSERVER),
				sqlchemy.OR(
					sqlchemy.In(q.Field("manager_id"), providerSQ),
					sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
				),
			),
		)

		if vpcUsable || usable {
			q = q.Equals("status", api.VPC_STATUS_AVAILABLE)
		}

		if usable {
			wires := WireManager.Query().SubQuery()
			networks := NetworkManager.Query().SubQuery()

			sq := wires.Query(wires.Field("vpc_id")).Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).Filter(
				sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE),
			)

			q = q.In("id", sq.SubQuery())
		}
	}

	if len(query.ZoneId) > 0 {
		zoneObj, err := validators.ValidateModel(ctx, userCred, ZoneManager, &query.ZoneId)
		if err != nil {
			return nil, err
		}
		region, err := zoneObj.(*SZone).GetRegion()
		if err != nil {
			return nil, errors.Wrapf(err, "get zone %s region", zoneObj.GetName())
		}
		q = q.Equals("cloudregion_id", region.Id)
		wires := WireManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		sq := wires.Query(wires.Field("vpc_id")).Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).Filter(
			sqlchemy.OR(
				sqlchemy.Equals(wires.Field("zone_id"), query.ZoneId),
				sqlchemy.IsNullOrEmpty(wires.Field("zone_id")),
			),
		)
		q = q.In("id", sq.SubQuery())

	}

	if query.IsDefault != nil {
		if *query.IsDefault {
			q = q.IsTrue("is_default")
		} else {
			q = q.IsFalse("is_default")
		}
	}
	if len(query.CidrBlock) > 0 {
		q = q.In("cidr_block", query.CidrBlock)
	}

	if len(query.CidrBlock6) > 0 {
		q = q.In("cidr_block6", query.CidrBlock6)
	}

	return q, nil
}

func (manager *SVpcManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "vpc":
		q = q.AppendField(q.Field("name").Label("vpc")).Distinct()
		return q, nil
	default:
		var err error
		q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SGlobalVpcResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SVpcManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SGlobalVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.GlobalVpcResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGlobalVpcResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByNetworkCount}) {
		var (
			wireq = WireManager.Query().SubQuery()
			netq  = NetworkManager.Query().SubQuery()
			vpcq  = VpcManager.Query().SubQuery()
		)
		countq := vpcq.Query(
			vpcq.Field("id"),
			sqlchemy.COUNT("network_count", netq.Field("id")),
		)
		countq = countq.Join(wireq, sqlchemy.OR(
			sqlchemy.Equals(countq.Field("id"), wireq.Field("vpc_id")),
			sqlchemy.AND(
				sqlchemy.Equals(countq.Field("id"), api.DEFAULT_VPC_ID),
				sqlchemy.IsNullOrEmpty(wireq.Field("vpc_id")),
			),
		))
		countq = countq.Join(netq, sqlchemy.Equals(
			netq.Field("wire_id"), wireq.Field("id"),
		))
		countq = countq.GroupBy(vpcq.Field("id"))
		countSubq := countq.SubQuery()
		q = q.LeftJoin(countSubq, sqlchemy.Equals(
			q.Field("id"), countSubq.Field("id"),
		))
		q.AppendField(q.QueryFields()...)
		q = q.AppendField(countSubq.Field("network_count"))
		q = db.OrderByFields(q,
			[]string{
				query.OrderByNetworkCount,
			},
			[]sqlchemy.IQueryField{
				countSubq.Field("network_count"),
			},
		)
	}
	if db.NeedOrderQuery([]string{query.OrderByWireCount}) {
		wireQ := WireManager.Query()
		wireQ = wireQ.AppendField(wireQ.Field("vpc_id"), sqlchemy.COUNT("wire_count"))
		wireQ = wireQ.GroupBy(wireQ.Field("vpc_id"))
		wireSQ := wireQ.SubQuery()
		q = q.LeftJoin(wireSQ, sqlchemy.Equals(wireSQ.Field("vpc_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(wireSQ.Field("wire_count"))
		q = db.OrderByFields(q, []string{query.OrderByWireCount}, []sqlchemy.IQueryField{q.Field("wire_count")})
	}
	return q, nil
}

func (svpc *SVpc) SyncRemoteWires(ctx context.Context, userCred mcclient.TokenCredential) error {
	ivpc, err := svpc.GetIVpc(ctx)
	if err != nil {
		return errors.Wrap(err, "GetIVpc")
	}

	provider := CloudproviderManager.FetchCloudproviderById(svpc.ManagerId)
	syncVpcWires(ctx, userCred, nil, provider, svpc, ivpc, nil, &SSyncRange{})

	hosts := HostManager.GetHostsByManagerAndRegion(provider.Id, svpc.CloudregionId)
	for i := 0; i < len(hosts); i += 1 {
		ihost, err := hosts[i].GetIHost(ctx)
		if err != nil {
			return err
		}
		syncHostNics(ctx, userCred, nil, provider, &hosts[i], ihost)
	}
	return nil
}

// 同步VPC状态
func (vpc *SVpc) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcSyncstatusInput) (jsonutils.JSONObject, error) {
	return vpc.PerformSync(ctx, userCred, query, input)
}

func (vpc *SVpc) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcSyncstatusInput) (jsonutils.JSONObject, error) {
	if vpc.IsManaged() {
		return nil, StartResourceSyncStatusTask(ctx, userCred, vpc, "VpcSyncstatusTask", "")
	}
	return nil, httperrors.NewUnsupportOperationError("on-premise vpc cannot sync status")
}

func (svpc *SVpc) initWire(ctx context.Context, zone *SZone, externalId string) (*SWire, error) {
	wire := &SWire{
		Bandwidth: 10000,
		Mtu:       options.Options.OvnUnderlayMtu,
	}
	wire.VpcId = svpc.Id
	wire.ZoneId = zone.Id
	wire.IsEmulated = true
	wire.ExternalId = externalId
	wire.Name = fmt.Sprintf("vpc-%s", svpc.Name)

	wire.DomainId = svpc.DomainId
	wire.IsPublic = svpc.IsPublic
	wire.PublicScope = svpc.PublicScope

	wire.ManagerId = svpc.ManagerId

	wire.SetModelManager(WireManager, wire)
	err := WireManager.TableSpec().Insert(ctx, wire)
	if err != nil {
		return nil, err
	}
	return wire, nil
}

func GetVpcQuotaKeysFromCreateInput(owner mcclient.IIdentityProvider, input api.VpcCreateInput) quotas.SDomainRegionalCloudResourceKeys {
	ownerId := &db.SOwnerId{DomainId: owner.GetProjectDomainId()}
	var region *SCloudregion
	if len(input.CloudregionId) > 0 {
		region = CloudregionManager.FetchRegionById(input.CloudregionId)
	}
	var provider *SCloudprovider
	if len(input.CloudproviderId) > 0 {
		provider = CloudproviderManager.FetchCloudproviderById(input.CloudproviderId)
	}
	regionKeys := fetchRegionalQuotaKeys(rbacscope.ScopeDomain, ownerId, region, provider)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = regionKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = regionKeys.SRegionalBaseKeys
	keys.SCloudResourceBaseKeys = regionKeys.SCloudResourceBaseKeys
	return keys
}

func (vpc *SVpc) GetQuotaKeys() quotas.SDomainRegionalCloudResourceKeys {
	region, _ := vpc.GetRegion()
	manager := vpc.GetCloudprovider()
	ownerId := vpc.GetOwnerId()
	regionKeys := fetchRegionalQuotaKeys(rbacscope.ScopeDomain, ownerId, region, manager)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = regionKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = regionKeys.SRegionalBaseKeys
	keys.SCloudResourceBaseKeys = regionKeys.SCloudResourceBaseKeys
	return keys
}

func (vpc *SVpc) GetUsages() []db.IUsage {
	if vpc.Deleted {
		return nil
	}
	usage := SInfrasQuota{Vpc: 1}
	keys := vpc.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (manager *SVpcManager) totalCount(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	providers []string,
	brands []string,
	cloudEnv string,
) int {
	q := VpcManager.Query()

	if scope != rbacscope.ScopeSystem && ownerId != nil {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)

	cnt, _ := q.CountWithError()

	return cnt
}

func (vpc *SVpc) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{
		vpc.SManagedResourceBase.GetChangeOwnerCandidateDomainIds(),
	}
	globalVpc, _ := vpc.GetGlobalVpc()
	if globalVpc != nil {
		candidates = append(candidates, db.ISharableChangeOwnerCandidateDomainIds(globalVpc))
	}
	log.Debugf("Candidate: %s", candidates)
	return db.ISharableMergeChangeOwnerCandidateDomainIds(vpc, candidates...)
}

func (vpc *SVpc) GetChangeOwnerRequiredDomainIds() []string {
	requires := stringutils2.SSortedStrings{}
	wires, _ := vpc.GetWires()
	for i := range wires {
		requires = stringutils2.Append(requires, wires[i].DomainId)
	}
	return requires
}

func (vpc *SVpc) GetRequiredSharedDomainIds() []string {
	wires, _ := vpc.GetWires()
	if len(wires) == 0 {
		return vpc.SEnabledStatusInfrasResourceBase.GetRequiredSharedDomainIds()
	}
	requires := make([][]string, len(wires))
	for i := range wires {
		requires[i] = db.ISharableChangeOwnerCandidateDomainIds(&wires[i])
	}
	return db.ISharableMergeShareRequireDomainIds(requires...)
}

func (manager *SVpcManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SGlobalVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGlobalVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGlobalVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.Contains("wire_count") {
		wires := WireManager.Query("vpc_id").SubQuery()
		subq := wires.Query(sqlchemy.COUNT("wire_count"), wires.Field("vpc_id")).GroupBy(wires.Field("vpc_id")).SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("id"), subq.Field("vpc_id")))
		q = q.AppendField(subq.Field("wire_count"))
	}

	if keys.Contains("network_count") {
		wires := WireManager.Query("id", "vpc_id").SubQuery()
		networks := NetworkManager.Query("wire_id").SubQuery()
		subq := networks.Query(sqlchemy.COUNT("network_count"), wires.Field("vpc_id"))
		subq = subq.LeftJoin(wires, sqlchemy.Equals(networks.Field("wire_id"), wires.Field("id")))
		subq = subq.GroupBy(wires.Field("vpc_id"))
		subqQ := subq.SubQuery()
		q = q.LeftJoin(subqQ, sqlchemy.Equals(q.Field("id"), subqQ.Field("vpc_id")))
		q = q.AppendField(subqQ.Field("network_count"))
	}

	return q, nil
}

func (vpc *SVpc) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	if vpc.Id == api.DEFAULT_VPC_ID && rbacscope.String2ScopeDefault(input.Scope, rbacscope.ScopeSystem) != rbacscope.ScopeSystem {
		return nil, httperrors.NewForbiddenError("For default vpc, only system level sharing can be set")
	}
	_, err := vpc.SEnabledStatusInfrasResourceBase.PerformPublic(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformPublic")
	}
	// perform public for all emulated wires
	wires, _ := vpc.GetWires()
	for i := range wires {
		if wires[i].IsEmulated {
			_, err := wires[i].PerformPublic(ctx, userCred, query, input)
			if err != nil {
				return nil, errors.Wrap(err, "wire.PerformPublic")
			}
		}
	}
	nats, err := vpc.GetNatgateways()
	if err != nil {
		return nil, errors.Wrapf(err, "vpc.GetNatgateways")
	}
	for i := range nats {
		_, err = nats[i].PerformPublic(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrapf(err, "nat.PerformPublic")
		}
	}
	return nil, nil
}

func (vpc *SVpc) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	if vpc.Id == "default" {
		return nil, httperrors.NewForbiddenError("Prohibit making default vpc private")
	}
	nats, err := vpc.GetNatgateways()
	if err != nil {
		return nil, errors.Wrapf(err, "vpc.GetNatgateways")
	}
	for i := range nats {
		_, err = nats[i].PerformPrivate(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrapf(err, "nat.PerformPrivate")
		}
	}
	// perform private for all emulated wires
	emptyNets := true
	wires, _ := vpc.GetWires()
	for i := range wires {
		if wires[i].DomainId == vpc.DomainId {
			nets, _ := wires[i].getNetworks(ctx, nil, nil, rbacscope.ScopeNone)
			for j := range nets {
				if nets[j].DomainId != vpc.DomainId {
					emptyNets = false
					break
				}
			}
			if !emptyNets {
				break
			}
		} else {
			emptyNets = false
			break
		}
	}
	if emptyNets {
		for i := range wires {
			nets, _ := wires[i].getNetworks(ctx, nil, nil, rbacscope.ScopeNone)
			netfail := false
			for j := range nets {
				if nets[j].IsPublic && nets[j].GetPublicScope().HigherEqual(rbacscope.ScopeDomain) {
					var err error
					if consts.GetNonDefaultDomainProjects() {
						netinput := apis.PerformPublicProjectInput{}
						netinput.Scope = string(rbacscope.ScopeDomain)
						_, err = nets[j].PerformPublic(ctx, userCred, nil, netinput)
					} else {
						_, err = nets[j].PerformPrivate(ctx, userCred, nil, input)
					}
					if err != nil {
						log.Errorf("nets[j].PerformPublic fail %s", err)
						netfail = true
						break
					}
				}
			}
			if netfail {
				break
			}
			_, err := wires[i].PerformPrivate(ctx, userCred, query, input)
			if err != nil {
				log.Errorf("wires[i].PerformPrivate fail %s", err)
				break
			}
		}
	}
	return vpc.SEnabledStatusInfrasResourceBase.PerformPrivate(ctx, userCred, query, input)
}

func (vpc *SVpc) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	_, err := vpc.SEnabledStatusInfrasResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformChangeOwner")
	}
	wires, _ := vpc.GetWires()
	for i := range wires {
		if wires[i].IsEmulated {
			_, err := wires[i].PerformChangeOwner(ctx, userCred, query, input)
			if err != nil {
				return nil, errors.Wrap(err, "wires[i].PerformChangeOwner")
			}
		}
	}

	return nil, nil
}

func (svpc *SVpc) GetVpcPeeringConnections() ([]SVpcPeeringConnection, error) {
	q := VpcPeeringConnectionManager.Query().Equals("vpc_id", svpc.Id)
	peers := []SVpcPeeringConnection{}
	err := db.FetchModelObjects(VpcPeeringConnectionManager, q, &peers)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return peers, nil
}

func (svpc *SVpc) GetVpcPeeringConnectionByExtId(extId string) (*SVpcPeeringConnection, error) {
	peer, err := db.FetchByExternalIdAndManagerId(VpcPeeringConnectionManager, extId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("vpc_id", svpc.Id)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "FetchByExternalIdAndManagerId %s", extId)
	}
	return peer.(*SVpcPeeringConnection), nil
}

func (svpc *SVpc) GetAccepterVpcPeeringConnectionByExtId(extId string) (*SVpcPeeringConnection, error) {
	peer, err := db.FetchByExternalIdAndManagerId(VpcPeeringConnectionManager, extId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("peer_vpc_id", svpc.Id)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "FetchByExternalIdAndManagerId %s", extId)
	}
	return peer.(*SVpcPeeringConnection), nil
}

func (svpc *SVpc) BackSycVpcPeeringConnectionsVpc(exts []cloudprovider.ICloudVpcPeeringConnection) compare.SyncResult {
	result := compare.SyncResult{}
	for i := range exts {
		Peering, err := svpc.GetAccepterVpcPeeringConnectionByExtId(exts[i].GetId())
		if err != nil {
			if errors.Cause(err) != errors.ErrNotFound {
				result.Error(err)
			}
			break
		}
		if len(Peering.PeerVpcId) == 0 {
			_, err := db.Update(Peering, func() error {
				Peering.PeerVpcId = svpc.GetId()
				return nil
			})
			if err != nil {
				result.Error(err)
				break
			}
		}
	}
	return result

}

func (svpc *SVpc) SyncVpcPeeringConnections(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	exts []cloudprovider.ICloudVpcPeeringConnection,
	xor bool,
) compare.SyncResult {
	result := compare.SyncResult{}

	dbPeers, err := svpc.GetVpcPeeringConnections()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetVpcPeeringConnections"))
		return result
	}

	provider := svpc.GetCloudprovider()

	removed := make([]SVpcPeeringConnection, 0)
	commondb := make([]SVpcPeeringConnection, 0)
	commonext := make([]cloudprovider.ICloudVpcPeeringConnection, 0)
	added := make([]cloudprovider.ICloudVpcPeeringConnection, 0)

	err = compare.CompareSets(dbPeers, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		if len(removed[i].ExternalId) > 0 {
			err = removed[i].syncRemove(ctx, userCred)
			if err != nil {
				result.DeleteError(err)
				continue
			}
			result.Delete()
		}
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudPeerConnection(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		_, err := svpc.newFromCloudPeerConnection(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(errors.Wrapf(err, "newFromCloudPeerConnection"))
			continue
		}
		result.Add()
	}

	return result
}

func (svpc *SVpc) newFromCloudPeerConnection(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudVpcPeeringConnection, provider *SCloudprovider) (*SVpcPeeringConnection, error) {
	peer := &SVpcPeeringConnection{}
	peer.SetModelManager(VpcPeeringConnectionManager, peer)
	peer.ExternalId = ext.GetGlobalId()
	peer.Status = ext.GetStatus()
	peer.VpcId = svpc.Id
	peer.Enabled = tristate.True
	peer.ExtPeerVpcId = ext.GetPeerVpcId()
	manager := svpc.GetCloudprovider()
	peerVpc, _ := db.FetchByExternalIdAndManagerId(VpcManager, peer.ExtPeerVpcId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		managerQ := CloudproviderManager.Query("id").Equals("provider", manager.Provider)
		return q.In("manager_id", managerQ.SubQuery())
	})
	if peerVpc != nil {
		peer.PeerVpcId = peerVpc.GetId()
	}
	peer.ExtPeerAccountId = ext.GetPeerAccountId()
	var err = func() error {
		lockman.LockClass(ctx, VpcPeeringConnectionManager, "name")
		defer lockman.ReleaseClass(ctx, VpcPeeringConnectionManager, "name")

		var err error
		peer.Name, err = db.GenerateName(ctx, VpcPeeringConnectionManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return VpcPeeringConnectionManager.TableSpec().Insert(ctx, peer)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	if provider != nil {
		SyncCloudDomain(userCred, peer, provider.GetOwnerId())
		peer.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogEvent(peer, db.ACT_CREATE, peer.GetShortDesc(ctx), userCred)
	return peer, nil
}

func (svpc *SVpc) IsSupportAssociateEip() bool {
	if utils.IsInStringArray(svpc.ExternalAccessMode, []string{api.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW, api.VPC_EXTERNAL_ACCESS_MODE_EIP}) {
		return true
	}

	return false
}

func (svpc *SVpc) GetDetailsTopology(ctx context.Context, userCred mcclient.TokenCredential, input *api.VpcTopologyInput) (*api.VpcTopologyOutput, error) {
	ret := &api.VpcTopologyOutput{
		Name:   svpc.Name,
		Status: svpc.Status,
		Wires:  []api.WireTopologyOutput{},
	}
	wires, err := svpc.GetWires()
	if err != nil {
		return ret, errors.Wrapf(err, "GetWires")
	}
	for i := range wires {
		wire := api.WireTopologyOutput{
			Name:      wires[i].Name,
			Status:    wires[i].Status,
			Bandwidth: wires[i].Bandwidth,
			Networks:  []api.NetworkTopologyOutput{},
			Hosts:     []api.HostTopologyOutput{},
		}
		if len(wires[i].ZoneId) > 0 {
			zone, _ := wires[i].GetZone()
			if zone != nil {
				wire.Zone = zone.Name
			}
		}
		hosts, err := wires[i].GetHosts()
		if err != nil {
			return nil, errors.Wrapf(err, "GetHosts for wire %s", wires[i].Id)
		}
		for i := range hosts {
			hns := hosts[i].GetBaremetalnetworks()
			hss := hosts[i]._getAttachedStorages(tristate.None, tristate.None, nil)
			host := api.HostTopologyOutput{
				Name:       hosts[i].Name,
				Id:         hosts[i].Id,
				Status:     hosts[i].Status,
				HostStatus: hosts[i].HostStatus,
				HostType:   hosts[i].HostType,
				Networks:   []api.HostnetworkTopologyOutput{},
				Storages:   []api.StorageShortDesc{},
				Schedtags:  GetSchedtagsDetailsToResourceV2(&hosts[i], ctx),
			}
			for j := range hns {
				host.Networks = append(host.Networks, api.HostnetworkTopologyOutput{
					IpAddr:  hns[j].IpAddr,
					MacAddr: hns[j].MacAddr,
				})
			}
			for j := range hss {
				host.Storages = append(host.Storages, api.StorageShortDesc{
					Name:        hss[j].Name,
					Id:          hss[j].Id,
					Status:      hss[j].Status,
					Enabled:     hss[j].Enabled.Bool(),
					StorageType: hss[j].StorageType,
					CapacityMb:  hss[j].Capacity,
				})
			}
			wire.Hosts = append(wire.Hosts, host)
		}
		networks, err := wires[i].GetNetworks(ctx, nil, nil, rbacscope.ScopeSystem)
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetworks")
		}
		for j := range networks {
			network := api.NetworkTopologyOutput{
				Name:         networks[j].Name,
				Status:       networks[j].Status,
				GuestIpStart: networks[j].GuestIpStart,
				GuestIpEnd:   networks[j].GuestIpEnd,
				GuestIpMask:  networks[j].GuestIpMask,
				ServerType:   networks[j].ServerType,
				VlanId:       networks[j].VlanId,
				// Address:      []api.SNetworkUsedAddress{},
			}

			network.GetNetworkAddressesOutput, err = networks[j].fetchAddressDetails(ctx, userCred, userCred, rbacscope.ScopeSystem)
			if err != nil {
				return nil, errors.Wrapf(err, "fetchAddressDetails")
			}

			wire.Networks = append(wire.Networks, network)
		}
		ret.Wires = append(ret.Wires, wire)
	}
	return ret, nil
}

func (self *SVpc) CheckSecurityGroupConsistent(secgroup *SSecurityGroup) error {
	if secgroup.Status != api.SECGROUP_STATUS_READY {
		return httperrors.NewInvalidStatusError("security group %s status is not ready", secgroup.Name)
	}
	if len(self.ExternalId) > 0 && len(secgroup.ExternalId) == 0 {
		return httperrors.NewInvalidStatusError("The security group %s does not have an external id", secgroup.Name)
	}
	if len(secgroup.VpcId) > 0 {
		if secgroup.VpcId != self.Id {
			return httperrors.NewInvalidStatusError("The security group does not belong to the vpc")
		}
	} else if len(secgroup.GlobalvpcId) > 0 {
		if secgroup.GlobalvpcId != self.GlobalvpcId {
			return httperrors.NewInvalidStatusError("The security group and vpc are in different global vpc")
		}
	} else if len(secgroup.CloudregionId) > 0 {
		if secgroup.CloudregionId != self.CloudregionId {
			return httperrors.NewInvalidStatusError("The security group and vpc are in different areas")
		}
	}
	return nil
}

func (self *SVpc) GetSecurityGroups() ([]SSecurityGroup, error) {
	q := SecurityGroupManager.Query().Equals("vpc_id", self.Id)
	ret := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SVpc) GetDefaultSecurityGroup(ownerId mcclient.IIdentityProvider, filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) (*SSecurityGroup, error) {
	q := SecurityGroupManager.Query().Equals("status", api.SECGROUP_STATUS_READY).Like("name", "%"+"default"+"%")

	q = filter(q)
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("public_scope"), "system"),
				sqlchemy.Equals(q.Field("is_public"), true),
			),
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()),
				sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
			),
		),
	)

	ret := &SSecurityGroup{}
	ret.SetModelManager(SecurityGroupManager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (manager *SVpcManager) FetchVpcById(id string) *SVpc {
	obj, err := manager.FetchById(id)
	if err != nil {
		log.Errorf("region %s %s", id, err)
		return nil
	}
	return obj.(*SVpc)
}

func (manager *SVpcManager) FetchDefaultVpc() *SVpc {
	return manager.FetchVpcById(api.DEFAULT_VPC_ID)
}
