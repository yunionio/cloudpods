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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWireManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SVpcResourceBaseManager
	SZoneResourceBaseManager
}

var WireManager *SWireManager

func init() {
	WireManager = &SWireManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SWire{},
			"wires_tbl",
			"wire",
			"wires",
		),
	}
	WireManager.SetVirtualObject(WireManager)
}

type SWire struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	SVpcResourceBase  `wdith:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" update:""`
	SZoneResourceBase `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"domain_required" update:""`

	// 带宽大小, 单位Mbps
	// example: 1000
	Bandwidth int `list:"domain" update:"domain" nullable:"false" create:"domain_required" json:"bandwidth"`
	// MTU
	// example: 1500
	Mtu int `list:"domain" update:"domain" nullable:"false" create:"domain_optional" default:"1500" json:"mtu"`
	// swagger:ignore
	ScheduleRank int `list:"domain" update:"domain" json:"schedule_rank"`

	// 可用区Id
	// ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"domain_required"`
	// VPC Id
	// VpcId string `wdith:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
}

func (manager *SWireManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{ZoneManager},
		{VpcManager},
	}
}

func (manager *SWireManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.WireCreateInput,
) (api.WireCreateInput, error) {
	var err error

	if input.Bandwidth < 0 {
		return input, httperrors.NewOutOfRangeError("bandwidth must be greater than 0")
	}

	if input.Mtu < 0 || input.Mtu > 1000000 {
		return input, httperrors.NewOutOfRangeError("mtu must be range of 0~1000000")
	}

	if input.VpcId == "" {
		input.VpcId = api.DEFAULT_VPC_ID
	}

	_vpc, err := validators.ValidateModel(userCred, VpcManager, &input.VpcId)
	if err != nil {
		return input, err
	}
	vpc := _vpc.(*SVpc)

	if len(vpc.ManagerId) > 0 {
		return input, httperrors.NewNotSupportedError("Currently only onpremise classic VPC supports creating wire")
	}

	if len(input.ZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("zone")
	}

	_, input.ZoneResourceInput, err = ValidateZoneResourceInput(userCred, input.ZoneResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateZoneResourceInput")
	}

	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (wire *SWire) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WireUpdateInput) (api.WireUpdateInput, error) {
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	keysV := []validators.IValidator{
		validators.NewNonNegativeValidator("bandwidth"),
		validators.NewRangeValidator("mtu", 1, 1000000).Optional(true),
	}
	for _, v := range keysV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return input, err
		}
	}
	var err error
	input.InfrasResourceBaseUpdateInput, err = wire.SInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.InfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (wire *SWire) ValidateDeleteCondition(ctx context.Context, info *api.WireDetails) error {
	if gotypes.IsNil(info) {
		info = &api.WireDetails{}
		usage, err := WireManager.TotalResourceCount([]string{wire.Id})
		if err != nil {
			return err
		}
		info.WireUsage, _ = usage[wire.Id]
	}
	if info.HostCount > 0 {
		return httperrors.NewNotEmptyError("wire contains hosts")
	}
	if info.Networks > 0 {
		return httperrors.NewNotEmptyError("wire contains networks")
	}
	return wire.SInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SWireManager) getWireExternalIdForClassicNetwork(provider string, vpcId string, zoneId string) string {
	if !utils.IsInStringArray(provider, api.REGIONAL_NETWORK_PROVIDERS) {
		return fmt.Sprintf("%s-%s", vpcId, zoneId)
	}
	return vpcId
}

func (manager *SWireManager) GetOrCreateWireForClassicNetwork(ctx context.Context, vpc *SVpc, zone *SZone) (*SWire, error) {
	cloudprovider := vpc.GetCloudprovider()
	if cloudprovider == nil {
		return nil, fmt.Errorf("failed to found cloudprovider for vpc %s(%s)", vpc.Id, vpc.Id)
	}
	externalId := manager.getWireExternalIdForClassicNetwork(cloudprovider.Provider, vpc.Id, zone.Id)
	name := fmt.Sprintf("emulate for vpc %s classic network", vpc.Id)
	zoneId := zone.Id
	if utils.IsInStringArray(cloudprovider.Provider, api.REGIONAL_NETWORK_PROVIDERS) { //reginal network
		zoneId = ""
	} else {
		name = fmt.Sprintf("emulate for zone %s vpc %s classic network", zone.Name, vpc.Id)
	}
	_wire, err := db.FetchByExternalIdAndManagerId(manager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("manager_id", vpc.ManagerId)
		return q
	})
	if err == nil {
		return _wire.(*SWire), nil
	}
	if errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchByExternalIdAndManagerId")
	}
	wire := &SWire{}
	wire.VpcId = vpc.Id
	wire.ZoneId = zoneId
	wire.SetModelManager(manager, wire)
	wire.ExternalId = externalId
	wire.IsEmulated = true
	wire.Name = name
	wire.ManagerId = vpc.ManagerId
	err = manager.TableSpec().Insert(ctx, wire)
	if err != nil {
		return nil, errors.Wrap(err, "Insert wire for classic network")
	}
	return wire, nil
}

func (wire *SWire) HostCount() (int, error) {
	q := NetInterfaceManager.Query().Equals("wire_id", wire.Id).GroupBy("baremetal_id")
	return q.CountWithError()
}

func (swire *SWire) GetHosts() ([]SHost, error) {
	sq := NetInterfaceManager.Query("baremetal_id").Equals("wire_id", swire.Id)
	q := HostManager.Query().In("id", sq)
	hosts := []SHost{}
	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return hosts, nil
}

func (wire *SWire) NetworkCount() (int, error) {
	q := NetworkManager.Query().Equals("wire_id", wire.Id)
	return q.CountWithError()
}

func (wire *SWire) GetVpcId() string {
	if len(wire.VpcId) == 0 {
		return "default"
	} else {
		return wire.VpcId
	}
}

func (manager *SWireManager) getWiresByVpcAndZone(vpc *SVpc, zone *SZone) ([]SWire, error) {
	wires := make([]SWire, 0)
	q := manager.Query()
	if vpc != nil {
		q = q.Equals("vpc_id", vpc.Id)
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	err := db.FetchModelObjects(manager, q, &wires)
	if err != nil {
		return nil, err
	}
	return wires, nil
}

func (manager *SWireManager) SyncWires(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	vpc *SVpc,
	wires []cloudprovider.ICloudWire,
	provider *SCloudprovider,
	xor bool,
	zone *SZone,
) ([]SWire, []cloudprovider.ICloudWire, compare.SyncResult) {
	lockman.LockRawObject(ctx, manager.Keyword(), vpc.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), vpc.Id)

	localWires := make([]SWire, 0)
	remoteWires := make([]cloudprovider.ICloudWire, 0)
	syncResult := compare.SyncResult{}

	dbWires, err := manager.getWiresByVpcAndZone(vpc, zone)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := range dbWires {
		if taskman.TaskManager.IsInTask(&dbWires[i]) {
			syncResult.Error(fmt.Errorf("object in task"))
			return nil, nil, syncResult
		}
	}

	removed := make([]SWire, 0)
	commondb := make([]SWire, 0)
	commonext := make([]cloudprovider.ICloudWire, 0)
	added := make([]cloudprovider.ICloudWire, 0)

	err = compare.CompareSets(dbWires, wires, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudWire(ctx, userCred)
		if err != nil { // cannot delete
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].syncWithCloudWire(ctx, userCred, commonext[i], vpc, provider, zone)
			if err != nil {
				syncResult.UpdateError(err)
			}
			syncResult.Update()
		}
		localWires = append(localWires, commondb[i])
		remoteWires = append(remoteWires, commonext[i])
	}
	for i := 0; i < len(added); i += 1 {
		wire, err := manager.newFromCloudWire(ctx, userCred, added[i], vpc, provider, zone)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localWires = append(localWires, *wire)
		remoteWires = append(remoteWires, added[i])
		syncResult.Add()
	}

	return localWires, remoteWires, syncResult
}

func (swire *SWire) syncRemoveCloudWire(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, swire)
	defer lockman.ReleaseObject(ctx, swire)

	vpc, _ := swire.GetVpc()
	cloudprovider := vpc.GetCloudprovider()
	if cloudprovider == nil || swire.ExternalId == WireManager.getWireExternalIdForClassicNetwork(cloudprovider.Provider, swire.VpcId, swire.ZoneId) {
		return nil
	}

	err := swire.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		err = swire.markNetworkUnknown(userCred)
	} else {
		err = swire.Delete(ctx, userCred)
	}
	return err
}

func (swire *SWire) syncWithCloudWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire, vpc *SVpc, provider *SCloudprovider, zone *SZone) error {
	diff, err := db.UpdateWithLock(ctx, swire, func() error {
		// swire.Name = extWire.GetName()
		swire.Bandwidth = extWire.GetBandwidth() // 10G

		swire.IsEmulated = extWire.IsEmulated()
		swire.Status = extWire.GetStatus()

		if len(swire.Description) == 0 {
			swire.Description = extWire.GetDescription()
		}

		swire.ManagerId = provider.Id

		if zone != nil {
			swire.ZoneId = zone.Id
		} else {
			vpc, _ := swire.GetVpc()
			if vpc != nil {
				region, err := vpc.GetRegion()
				if err != nil {
					return errors.Wrapf(err, "vpc.GetRegion")
				}
				if utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
					swire.ZoneId = ""
				}
			}
		}

		if swire.IsEmulated {
			swire.DomainId = vpc.DomainId
			// swire.IsPublic = vpc.IsPublic
			// swire.PublicScope = vpc.PublicScope
			// swire.PublicSrc = vpc.PublicSrc
		}

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudWire error %s", err)
	}

	if provider != nil && !swire.IsEmulated {
		SyncCloudDomain(userCred, swire, provider.GetOwnerId())
		swire.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	} else if swire.IsEmulated {
		swire.SaveSharedInfo(apis.TOwnerSource(vpc.PublicSrc), ctx, userCred, vpc.GetSharedInfo())
	}
	syncMetadata(ctx, userCred, swire, extWire, false)

	db.OpsLog.LogSyncUpdate(swire, diff, userCred)
	return err
}

func (swire *SWire) markNetworkUnknown(userCred mcclient.TokenCredential) error {
	nets, err := swire.getNetworks(nil, nil, rbacscope.ScopeNone)
	if err != nil {
		return err
	}
	for i := 0; i < len(nets); i += 1 {
		nets[i].SetStatus(userCred, api.NETWORK_STATUS_UNKNOWN, "wire sync to remove")
	}
	return nil
}

func (manager *SWireManager) newFromCloudWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire, vpc *SVpc, provider *SCloudprovider, zone *SZone) (*SWire, error) {
	wire := SWire{}
	wire.SetModelManager(manager, &wire)

	wire.ExternalId = extWire.GetGlobalId()
	wire.Bandwidth = extWire.GetBandwidth()
	wire.Status = extWire.GetStatus()
	wire.Description = extWire.GetDescription()
	wire.VpcId = vpc.Id
	wire.ManagerId = provider.Id
	var err error
	wire.ZoneId, err = func() (string, error) {
		if zone != nil {
			return zone.Id, nil
		}
		izone := extWire.GetIZone()
		if gotypes.IsNil(izone) {
			return "", nil
		}
		zone, err := vpc.getZoneByExternalId(izone.GetGlobalId())
		if err != nil {
			return "", errors.Wrapf(err, "getZoneByExternalId")
		}
		return zone.Id, nil
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "get zone id")
	}

	wire.IsEmulated = extWire.IsEmulated()

	wire.DomainId = vpc.DomainId
	wire.IsPublic = vpc.IsPublic
	wire.PublicScope = vpc.PublicScope
	wire.PublicSrc = vpc.PublicSrc

	err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, extWire.GetName())
		if err != nil {
			return err
		}
		wire.Name = newName

		return manager.TableSpec().Insert(ctx, &wire)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	if provider != nil && !wire.IsEmulated {
		SyncCloudDomain(userCred, &wire, provider.GetOwnerId())
		wire.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	syncMetadata(ctx, userCred, &wire, extWire, false)
	db.OpsLog.LogEvent(&wire, db.ACT_CREATE, wire.GetShortDesc(ctx), userCred)
	return &wire, nil
}

func filterByScopeOwnerId(q *sqlchemy.SQuery, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, domainResource bool) *sqlchemy.SQuery {
	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		if domainResource {
			q = q.Equals("domain_id", ownerId.GetProjectId())
		} else {
			q = q.Equals("tenant_id", ownerId.GetProjectId())
		}
	}
	return q
}

func fixVmwareProvider(providers []string) (bool, []string) {
	findVmware := false
	findOnecloud := false
	newp := make([]string, 0)
	for _, p := range providers {
		if p == api.CLOUD_PROVIDER_VMWARE {
			findVmware = true
		} else {
			if p == api.CLOUD_PROVIDER_ONECLOUD {
				findOnecloud = true
			}
			newp = append(newp, p)
		}
	}
	if findVmware && !findOnecloud {
		newp = append(newp, api.CLOUD_PROVIDER_ONECLOUD)
	}
	return findVmware, newp
}

func (manager *SWireManager) totalCountQ(
	rangeObjs []db.IStandaloneModel,
	hostTypes []string, hostProviders, hostBrands []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) *sqlchemy.SQuery {
	guestsQ := GuestManager.Query()
	guestsQ = filterByScopeOwnerId(guestsQ, scope, ownerId, false)
	guestsQ = db.ObjectIdQueryWithPolicyResult(guestsQ, GuestManager, policyResult)
	guests := guestsQ.SubQuery()

	// hosts no filter, for guest networks
	hostsQ := HostManager.Query()
	if len(hostTypes) > 0 {
		hostsQ = hostsQ.In("host_type", hostTypes)
	}
	if len(hostProviders) > 0 || len(hostBrands) > 0 || len(cloudEnv) > 0 {
		hostsQ = CloudProviderFilter(hostsQ, hostsQ.Field("manager_id"), providers, brands, cloudEnv)
	}
	if len(rangeObjs) > 0 {
		hostsQ = RangeObjectsFilter(hostsQ, rangeObjs, nil, hostsQ.Field("zone_id"), hostsQ.Field("manager_id"), hostsQ.Field("id"), nil)
	}
	hosts := hostsQ.SubQuery()

	// hosts filter by owner, for host networks
	hostsQ2 := HostManager.Query()
	hostsQ2 = db.ObjectIdQueryWithPolicyResult(hostsQ2, HostManager, policyResult)
	hostsQ2 = filterByScopeOwnerId(hostsQ2, scope, ownerId, true)
	if len(hostTypes) > 0 {
		hostsQ2 = hostsQ2.In("host_type", hostTypes)
	}
	if len(hostProviders) > 0 || len(hostBrands) > 0 || len(cloudEnv) > 0 {
		hostsQ2 = CloudProviderFilter(hostsQ2, hostsQ2.Field("manager_id"), providers, brands, cloudEnv)
	}
	if len(rangeObjs) > 0 {
		hostsQ2 = RangeObjectsFilter(hostsQ2, rangeObjs, nil, hostsQ2.Field("zone_id"), hostsQ2.Field("manager_id"), hostsQ2.Field("id"), nil)
	}
	hosts2 := hostsQ2.SubQuery()

	groupsQ := GroupManager.Query()
	groupsQ = db.ObjectIdQueryWithPolicyResult(groupsQ, GroupManager, policyResult)
	groupsQ = filterByScopeOwnerId(groupsQ, scope, ownerId, false)
	groups := groupsQ.SubQuery()

	lbsQ := LoadbalancerManager.Query()
	lbsQ = db.ObjectIdQueryWithPolicyResult(lbsQ, LoadbalancerManager, policyResult)
	lbsQ = filterByScopeOwnerId(lbsQ, scope, ownerId, false)
	if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		lbsQ = CloudProviderFilter(lbsQ, lbsQ.Field("manager_id"), providers, brands, cloudEnv)
	}
	if len(rangeObjs) > 0 {
		lbsQ = RangeObjectsFilter(lbsQ, rangeObjs, lbsQ.Field("cloudregion_id"), lbsQ.Field("zone_id"), lbsQ.Field("manager_id"), nil, nil)
	}
	lbs := lbsQ.SubQuery()

	dbsQ := DBInstanceManager.Query()
	dbsQ = db.ObjectIdQueryWithPolicyResult(dbsQ, DBInstanceManager, policyResult)
	dbsQ = filterByScopeOwnerId(dbsQ, scope, ownerId, false)
	if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		dbsQ = CloudProviderFilter(dbsQ, dbsQ.Field("manager_id"), providers, brands, cloudEnv)
	}
	if len(rangeObjs) > 0 {
		dbsQ = RangeObjectsFilter(dbsQ, rangeObjs, dbsQ.Field("cloudregion_id"), dbsQ.Field("zone_id"), dbsQ.Field("manager_id"), nil, nil)
	}
	dbs := dbsQ.SubQuery()

	var (
		gNicSQ     *sqlchemy.SSubQuery
		gNicSQ4    *sqlchemy.SSubQuery
		gNicSQ6    *sqlchemy.SSubQuery
		hNicSQ     *sqlchemy.SSubQuery
		grpNicSQ   *sqlchemy.SSubQuery
		lbNicSQ    *sqlchemy.SSubQuery
		eipNicSQ   *sqlchemy.SSubQuery
		netifNicSQ *sqlchemy.SSubQuery
		dbNicSQ    *sqlchemy.SSubQuery
	)
	{
		gNics := GuestnetworkManager.Query().SubQuery()
		gNicQ := gNics.Query(
			gNics.Field("network_id"),
			sqlchemy.COUNT("gnic_count"),
			sqlchemy.SUM("pending_deleted_gnic_count", guests.Field("pending_deleted")),
		)
		gNicQ = gNicQ.Join(guests, sqlchemy.Equals(guests.Field("id"), gNics.Field("guest_id")))
		gNicQ = gNicQ.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
		gNicQ = gNicQ.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		gNicSQ = gNicQ.GroupBy(gNics.Field("network_id")).SubQuery()
	}
	{
		gNics := GuestnetworkManager.Query().IsNotEmpty("ip_addr").SubQuery()
		gNicQ := gNics.Query(
			gNics.Field("network_id"),
			sqlchemy.COUNT("gnic_count4"),
		)
		gNicQ = gNicQ.Join(guests, sqlchemy.Equals(guests.Field("id"), gNics.Field("guest_id")))
		gNicQ = gNicQ.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
		gNicQ = gNicQ.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		gNicSQ4 = gNicQ.GroupBy(gNics.Field("network_id")).SubQuery()
	}
	{
		gNics := GuestnetworkManager.Query().IsNotEmpty("ip6_addr").SubQuery()
		gNicQ := gNics.Query(
			gNics.Field("network_id"),
			sqlchemy.COUNT("gnic_count6"),
		)
		gNicQ = gNicQ.Join(guests, sqlchemy.Equals(guests.Field("id"), gNics.Field("guest_id")))
		gNicQ = gNicQ.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
		gNicQ = gNicQ.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		gNicSQ6 = gNicQ.GroupBy(gNics.Field("network_id")).SubQuery()
	}
	{
		hNics := HostnetworkManager.Query().SubQuery()
		hNicQ := hNics.Query(
			hNics.Field("network_id"),
			sqlchemy.COUNT("hnic_count"),
		)
		hNicQ = hNicQ.Join(hosts2, sqlchemy.Equals(hNics.Field("baremetal_id"), hosts2.Field("id")))
		hNicQ = hNicQ.Filter(sqlchemy.IsTrue(hosts2.Field("enabled")))
		hNicSQ = hNicQ.GroupBy(hNics.Field("network_id")).SubQuery()
	}
	{
		groupNics := GroupnetworkManager.Query().SubQuery()
		grpNicQ := groupNics.Query(
			groupNics.Field("network_id"),
			sqlchemy.COUNT("grpnic_count"),
		)
		grpNicQ = grpNicQ.Join(groups, sqlchemy.Equals(groups.Field("id"), groupNics.Field("group_id")))
		grpNicSQ = grpNicQ.GroupBy(groupNics.Field("network_id")).SubQuery()
	}
	{
		lbNics := LoadbalancernetworkManager.Query().SubQuery()
		lbNicQ := lbNics.Query(
			lbNics.Field("network_id"),
			sqlchemy.COUNT("lbnic_count"),
		)
		lbNicQ = lbNicQ.Join(lbs, sqlchemy.Equals(lbs.Field("id"), lbNics.Field("loadbalancer_id")))
		lbNicQ = lbNicQ.Filter(sqlchemy.IsFalse(lbs.Field("pending_deleted")))
		lbNicSQ = lbNicQ.GroupBy(lbNics.Field("network_id")).SubQuery()
	}
	{
		eipNicsQ := ElasticipManager.Query().IsNotEmpty("network_id")
		eipNicsQ = db.ObjectIdQueryWithPolicyResult(eipNicsQ, ElasticipManager, policyResult)
		eipNics := filterByScopeOwnerId(eipNicsQ, scope, ownerId, false).SubQuery()
		eipNicQ := eipNics.Query(
			eipNics.Field("network_id"),
			sqlchemy.COUNT("eipnic_count"),
		)
		if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
			eipNicQ = CloudProviderFilter(eipNicQ, eipNicQ.Field("manager_id"), providers, brands, cloudEnv)
		}
		if len(rangeObjs) > 0 {
			eipNicQ = RangeObjectsFilter(eipNicQ, rangeObjs, eipNicQ.Field("cloudregion_id"), nil, eipNicQ.Field("manager_id"), nil, nil)
		}
		eipNicSQ = eipNicQ.GroupBy(eipNics.Field("network_id")).SubQuery()
	}
	{
		netifsQ := NetworkInterfaceManager.Query()
		netifsQ = db.ObjectIdQueryWithPolicyResult(netifsQ, NetworkInterfaceManager, policyResult)
		netifsQ = filterByScopeOwnerId(netifsQ, scope, ownerId, true)
		if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
			netifsQ = CloudProviderFilter(netifsQ, netifsQ.Field("manager_id"), providers, brands, cloudEnv)
		}
		if len(rangeObjs) > 0 {
			netifsQ = RangeObjectsFilter(netifsQ, rangeObjs, netifsQ.Field("cloudregion_id"), nil, netifsQ.Field("manager_id"), nil, nil)
		}
		netifs := netifsQ.SubQuery()
		netifNics := NetworkinterfacenetworkManager.Query().SubQuery()
		netifNicQ := netifNics.Query(
			netifNics.Field("network_id"),
			sqlchemy.COUNT("netifnic_count"),
		)
		netifNicQ = netifNicQ.Join(netifs, sqlchemy.Equals(netifNics.Field("networkinterface_id"), netifs.Field("id")))
		netifNicSQ = netifNicQ.GroupBy(netifNics.Field("network_id")).SubQuery()
	}
	{
		dbNics := DBInstanceNetworkManager.Query().SubQuery()
		dbNicQ := dbNics.Query(
			dbNics.Field("network_id"),
			sqlchemy.COUNT("dbnic_count"),
		)
		dbNicQ = dbNicQ.Join(dbs, sqlchemy.Equals(dbs.Field("id"), dbNics.Field("dbinstance_id")))
		dbNicQ = dbNicQ.Filter(sqlchemy.IsFalse(dbs.Field("pending_deleted")))
		dbNicSQ = dbNicQ.GroupBy(dbNics.Field("network_id")).SubQuery()
	}

	networks := NetworkManager.Query().SubQuery()
	netQ := networks.Query(
		sqlchemy.SUM("guest_nic_count", gNicSQ.Field("gnic_count")),
		sqlchemy.SUM("pending_deleted_guest_nic_count", gNicSQ.Field("pending_deleted_gnic_count")),
		sqlchemy.SUM("host_nic_count", hNicSQ.Field("hnic_count")),
		sqlchemy.SUM("guest_nic_count4", gNicSQ4.Field("gnic_count4")),
		sqlchemy.SUM("guest_nic_count6", gNicSQ6.Field("gnic_count6")),
		sqlchemy.SUM("group_nic_count", grpNicSQ.Field("grpnic_count")),
		sqlchemy.SUM("lb_nic_count", lbNicSQ.Field("lbnic_count")),
		sqlchemy.SUM("eip_nic_count", eipNicSQ.Field("eipnic_count")),
		sqlchemy.SUM("netif_nic_count", netifNicSQ.Field("netifnic_count")),
		sqlchemy.SUM("db_nic_count", dbNicSQ.Field("dbnic_count")),
	)
	netQ = netQ.LeftJoin(gNicSQ, sqlchemy.Equals(gNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(gNicSQ4, sqlchemy.Equals(gNicSQ4.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(gNicSQ6, sqlchemy.Equals(gNicSQ6.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(hNicSQ, sqlchemy.Equals(hNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(grpNicSQ, sqlchemy.Equals(grpNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(lbNicSQ, sqlchemy.Equals(lbNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(eipNicSQ, sqlchemy.Equals(eipNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(netifNicSQ, sqlchemy.Equals(netifNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(dbNicSQ, sqlchemy.Equals(dbNicSQ.Field("network_id"), networks.Field("id")))

	return netQ
}

func (manager *SWireManager) totalCountQ2(
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) *sqlchemy.SQuery {
	var revSQ *sqlchemy.SSubQuery
	var revSQ6 *sqlchemy.SSubQuery
	{
		revIps := filterExpiredReservedIp4s(ReservedipManager.Query()).SubQuery()
		revQ := revIps.Query(
			revIps.Field("network_id"),
			sqlchemy.COUNT("rnic_count"),
		)
		revSQ = revQ.GroupBy(revIps.Field("network_id")).SubQuery()
	}
	{
		revIps := filterExpiredReservedIp6s(ReservedipManager.Query()).SubQuery()
		revQ := revIps.Query(
			revIps.Field("network_id"),
			sqlchemy.COUNT("rnic_count6"),
		)
		revSQ6 = revQ.GroupBy(revIps.Field("network_id")).SubQuery()
	}

	ownerNetQ1 := NetworkManager.Query()
	ownerNetQ1 = db.ObjectIdQueryWithPolicyResult(ownerNetQ1, NetworkManager, policyResult)
	ownerNetworks := filterByScopeOwnerId(ownerNetQ1, scope, ownerId, false).SubQuery()
	ownerNetQ := ownerNetworks.Query(
		ownerNetworks.Field("wire_id"),
		sqlchemy.COUNT("id").Label("net_count"),
		sqlchemy.SUM("rev_count", revSQ.Field("rnic_count")),
		sqlchemy.SUM("rev_count6", revSQ6.Field("rnic_count6")),
	)
	ownerNetQ = ownerNetQ.LeftJoin(revSQ, sqlchemy.Equals(revSQ.Field("network_id"), ownerNetworks.Field("id")))
	ownerNetQ = ownerNetQ.LeftJoin(revSQ6, sqlchemy.Equals(revSQ6.Field("network_id"), ownerNetworks.Field("id")))
	ownerNetQ = ownerNetQ.GroupBy(ownerNetworks.Field("wire_id"))
	ownerNetSQ := ownerNetQ.SubQuery()

	wires := WireManager.Query().SubQuery()
	q := wires.Query(
		sqlchemy.SUM("net_count", ownerNetSQ.Field("net_count")),
		sqlchemy.SUM("reserved_count", ownerNetSQ.Field("rev_count")),
		sqlchemy.SUM("reserved_count6", ownerNetSQ.Field("rev_count6")),
	)
	q = q.LeftJoin(ownerNetSQ, sqlchemy.Equals(wires.Field("id"), ownerNetSQ.Field("wire_id")))
	return filterWiresCountQuery(q, hostTypes, providers, brands, cloudEnv, rangeObjs)
}

func (manager *SWireManager) totalCountQ3(
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) *sqlchemy.SQuery {
	wiresQ := WireManager.Query()
	wiresQ = db.ObjectIdQueryWithPolicyResult(wiresQ, WireManager, policyResult)
	wires := filterByScopeOwnerId(WireManager.Query(), scope, ownerId, true).SubQuery()
	q := wires.Query(
		sqlchemy.COUNT("id").Label("wires_count"),
		sqlchemy.SUM("emulated_wires_count", wires.Field("is_emulated")),
	)
	return filterWiresCountQuery(q, hostTypes, providers, brands, cloudEnv, rangeObjs)
}

func filterWiresCountQuery(q *sqlchemy.SQuery, hostTypes, providers, brands []string, cloudEnv string, rangeObjs []db.IStandaloneModel) *sqlchemy.SQuery {
	if len(hostTypes) > 0 {
		hostwires := NetInterfaceManager.Query().SubQuery()
		hosts := HostManager.Query().SubQuery()
		hostWireQ := hostwires.Query(hostwires.Field("wire_id"))
		hostWireQ = hostWireQ.Join(hosts, sqlchemy.Equals(hostWireQ.Field("baremetal_id"), hosts.Field("id")))
		hostWireQ = hostWireQ.Filter(sqlchemy.In(hosts.Field("host_type"), hostTypes))
		hostWireQ = hostWireQ.GroupBy(hostwires.Field("wire_id"))
		hostWireSQ := hostWireQ.SubQuery()

		q = q.Join(hostWireSQ, sqlchemy.Equals(hostWireSQ.Field("wire_id"), q.Field("id")))
	}

	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		vpcs := VpcManager.Query().SubQuery()
		q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
		q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
		q = RangeObjectsFilter(q, rangeObjs, vpcs.Field("cloudregion_id"), q.Field("zone_id"), q.Field("manager_id"), nil, nil)
	}

	return q
}

type WiresCountStat struct {
	WiresCount         int
	EmulatedWiresCount int
	NetCount           int
	GuestNicCount      int
	GuestNicCount4     int
	GuestNicCount6     int
	HostNicCount       int
	ReservedCount      int
	ReservedCount6     int
	GroupNicCount      int
	LbNicCount         int
	EipNicCount        int
	NetifNicCount      int
	DbNicCount         int

	PendingDeletedGuestNicCount int
}

func (wstat WiresCountStat) NicCount() int {
	return wstat.GuestNicCount + wstat.HostNicCount + wstat.ReservedCount + wstat.GroupNicCount + wstat.LbNicCount + wstat.NetifNicCount + wstat.EipNicCount + wstat.DbNicCount
}

func (manager *SWireManager) TotalCount(
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) WiresCountStat {
	vmwareP, hostProviders := fixVmwareProvider(providers)
	vmwareB, hostBrands := fixVmwareProvider(brands)

	if vmwareP || vmwareB {
		if !utils.IsInStringArray(api.HOST_TYPE_ESXI, hostTypes) {
			hostTypes = append(hostTypes, api.HOST_TYPE_ESXI)
		}
	} else {
		if utils.IsInStringArray(api.HOST_TYPE_ESXI, hostTypes) {
			providers = append(providers, api.CLOUD_PROVIDER_VMWARE)
			brands = append(brands, api.CLOUD_PROVIDER_VMWARE)
		}
	}
	if len(hostTypes) > 0 {
		for _, p := range providers {
			if hs, ok := api.CLOUD_PROVIDER_HOST_TYPE_MAP[p]; ok {
				hostTypes = append(hostTypes, hs...)
			}
		}
		for _, p := range brands {
			if hs, ok := api.CLOUD_PROVIDER_HOST_TYPE_MAP[p]; ok {
				hostTypes = append(hostTypes, hs...)
			}
		}
	}
	log.Debugf("providers: %#v hostProviders: %#v brands: %#v hostBrands: %#v hostTypes: %#v", providers, hostProviders, brands, hostBrands, hostTypes)

	stat := WiresCountStat{}
	err := manager.totalCountQ(
		rangeObjs,
		hostTypes, hostProviders, hostBrands,
		providers, brands, cloudEnv,
		scope, ownerId,
		policyResult,
	).First(&stat)
	if err != nil {
		log.Errorf("Wire total count: %v", err)
	}
	err = manager.totalCountQ2(
		rangeObjs,
		hostTypes,
		providers, brands, cloudEnv,
		scope, ownerId,
		policyResult,
	).First(&stat)
	if err != nil {
		log.Errorf("Wire total count 2: %v", err)
	}
	err = manager.totalCountQ3(
		rangeObjs,
		hostTypes,
		providers, brands, cloudEnv,
		scope, ownerId,
		policyResult,
	).First(&stat)
	if err != nil {
		log.Errorf("Wire total count 2: %v", err)
	}
	return stat
}

func (swire *SWire) getNetworkQuery(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	additionalNicIds := NetworkAdditionalWireManager.networkIdQuery(swire.Id)
	q := NetworkManager.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("wire_id"), swire.Id),
		sqlchemy.In(q.Field("id"), additionalNicIds.SubQuery()),
	))
	if ownerId != nil {
		q = NetworkManager.FilterByOwner(q, NetworkManager, userCred, ownerId, scope)
	}
	return q
}

func (swire *SWire) GetNetworks(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) ([]SNetwork, error) {
	return swire.getNetworks(userCred, ownerId, scope)
}

func (swire *SWire) getNetworks(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) ([]SNetwork, error) {
	q := swire.getNetworkQuery(userCred, ownerId, scope)
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (swire *SWire) getGatewayNetworkQuery(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q := swire.getNetworkQuery(userCred, ownerId, scope)
	q = q.IsNotNull("guest_gateway").IsNotEmpty("guest_gateway")
	q = q.Equals("status", api.NETWORK_STATUS_AVAILABLE)
	return q
}

func (swire *SWire) getAutoAllocNetworks(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) ([]SNetwork, error) {
	q := swire.getGatewayNetworkQuery(userCred, ownerId, scope)
	q = q.IsTrue("is_auto_alloc")
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (swire *SWire) getPublicNetworks(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) ([]SNetwork, error) {
	q := swire.getGatewayNetworkQuery(userCred, ownerId, scope)
	q = q.IsTrue("is_public")
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (swire *SWire) getPrivateNetworks(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) ([]SNetwork, error) {
	q := swire.getGatewayNetworkQuery(userCred, ownerId, scope)
	q = q.IsFalse("is_public")
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (swire *SWire) GetCandidatePrivateNetwork(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, isExit bool, serverTypes []string) (*SNetwork, error) {
	nets, err := swire.getPrivateNetworks(userCred, ownerId, scope)
	if err != nil {
		return nil, err
	}
	return ChooseCandidateNetworks(nets, isExit, serverTypes), nil
}

func (swire *SWire) GetCandidateAutoAllocNetwork(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, isExit bool, serverTypes []string) (*SNetwork, error) {
	nets, err := swire.getAutoAllocNetworks(userCred, ownerId, scope)
	if err != nil {
		return nil, err
	}
	return ChooseCandidateNetworks(nets, isExit, serverTypes), nil
}

func (swire *SWire) GetCandidateNetworkForIp(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, ipAddr string) (*SNetwork, error) {
	ip, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return nil, err
	}
	netPrivates, err := swire.getPrivateNetworks(userCred, ownerId, scope)
	if err != nil {
		return nil, err
	}
	for _, net := range netPrivates {
		if net.IsAddressInRange(ip) {
			return &net, nil
		}
	}
	netPublics, err := swire.getPublicNetworks(userCred, ownerId, scope)
	if err != nil {
		return nil, err
	}
	for _, net := range netPublics {
		if net.IsAddressInRange(ip) {
			return &net, nil
		}
	}
	return nil, nil
}

func ChooseNetworkByAddressCount(nets []*SNetwork) (*SNetwork, *SNetwork) {
	return chooseNetworkByAddressCount(nets)
}

func chooseNetworkByAddressCount(nets []*SNetwork) (*SNetwork, *SNetwork) {
	minCnt := 65535
	maxCnt := 0
	var minSel *SNetwork
	var maxSel *SNetwork
	for _, net := range nets {
		cnt, err := net.getFreeAddressCount()
		if err != nil || cnt <= 0 {
			continue
		}
		if minSel == nil || minCnt > cnt {
			minSel = net
			minCnt = cnt
		}
		if maxSel == nil || maxCnt < cnt {
			maxSel = net
			maxCnt = cnt
		}
	}
	return minSel, maxSel
}

func ChooseCandidateNetworks(nets []SNetwork, isExit bool, serverTypes []string) *SNetwork {
	matchingNets := make([]*SNetwork, 0)
	notMatchingNets := make([]*SNetwork, 0)

	for _, s := range serverTypes {
		net := chooseCandidateNetworksByNetworkType(nets, isExit, s)
		if net != nil {
			if utils.IsInStringArray(net.ServerType, serverTypes) {
				matchingNets = append(matchingNets, net)
			} else {
				notMatchingNets = append(notMatchingNets, net)
			}
		}
	}

	if len(matchingNets) >= 1 {
		return matchingNets[0]
	}

	if len(notMatchingNets) >= 1 {
		return notMatchingNets[0]
	}

	return nil
}

func chooseCandidateNetworksByNetworkType(nets []SNetwork, isExit bool, serverType string) *SNetwork {
	matchingNets := make([]*SNetwork, 0)
	notMatchingNets := make([]*SNetwork, 0)

	for i := 0; i < len(nets); i++ {
		net := nets[i]
		if isExit != net.IsExitNetwork() {
			continue
		}
		if serverType == net.ServerType || (len(net.ServerType) == 0 && serverType == api.NETWORK_TYPE_GUEST) {
			matchingNets = append(matchingNets, &net)
		} else {
			notMatchingNets = append(notMatchingNets, &net)
		}
	}
	minSel, maxSel := chooseNetworkByAddressCount(matchingNets)
	if (isExit && minSel == nil) || (!isExit && maxSel == nil) {
		minSel, maxSel = chooseNetworkByAddressCount(notMatchingNets)
	}
	if isExit {
		return minSel
	} else {
		return maxSel
	}
}

func (manager *SWireManager) InitializeData() error {
	{
		err := manager.initVpcId()
		if err != nil {
			return errors.Wrap(err, "initVpcId")
		}
	}
	{
		err := manager.initManagerId()
		if err != nil {
			return errors.Wrap(err, "initManagerId")
		}
	}
	{
		err := manager.cleanNoVpcWires()
		if err != nil {
			return errors.Wrap(err, "cleanNoVpcWires")
		}
	}
	return nil
}

func (manager *SWireManager) initVpcId() error {
	wires, err := manager.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsEmpty(q.Field("vpc_id")),
			sqlchemy.IsEmpty(q.Field("status")),
			sqlchemy.Equals(q.Field("status"), "init"),
			sqlchemy.Equals(q.Field("status"), api.WIRE_STATUS_READY_DEPRECATED),
		))
		return q
	})
	if err != nil {
		return errors.Wrap(err, "FetchWires")
	}
	for i := range wires {
		w := wires[i]
		_, err := db.Update(&w, func() error {
			if len(w.VpcId) == 0 {
				w.VpcId = api.DEFAULT_VPC_ID
			}
			if len(w.Status) == 0 || w.Status == "init" || w.Status == api.WIRE_STATUS_READY_DEPRECATED {
				w.Status = api.WIRE_STATUS_AVAILABLE
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	}
	return nil
}

func (manager *SWireManager) initManagerId() error {
	wires, err := manager.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		vpcs := VpcManager.Query().SubQuery()
		q = q.Join(vpcs, sqlchemy.Equals(q.Field("vpc_id"), vpcs.Field("id")))
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNull(q.Field("manager_id")),
			sqlchemy.NotEquals(q.Field("manager_id"), vpcs.Field("manager_id")),
		))
		q = q.IsNullOrEmpty("manager_id")
		return q
	})
	if err != nil {
		return errors.Wrap(err, "FetchWires")
	}
	for i := range wires {
		w := wires[i]
		vpc, err := w.GetVpc()
		if err != nil {
			return errors.Wrap(err, "GetVpc")
		}
		_, err = db.Update(&w, func() error {
			w.ManagerId = vpc.ManagerId
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	}
	return nil
}

func (manager *SWireManager) cleanNoVpcWires() error {
	wires, err := manager.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		// find wires whose VPC was deleted
		deletedVpcs := VpcManager.RawQuery("id").IsTrue("deleted").SubQuery()
		q = q.Join(deletedVpcs, sqlchemy.Equals(q.Field("vpc_id"), deletedVpcs.Field("id")))
		return q
	})
	if err != nil {
		return errors.Wrap(err, "FetchWires")
	}
	for i := range wires {
		err := wires[i].Delete(context.Background(), auth.AdminCredential())
		if err != nil {
			return errors.Wrapf(err, "Delete wire %s", wires[i].Id)
		}
	}
	return nil
}

func (manager *SWireManager) FetchWires(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]SWire, error) {
	wires := make([]SWire, 0)
	q := manager.Query()
	q = filter(q)
	err := db.FetchModelObjects(manager, q, &wires)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return wires, nil
}

func (wire *SWire) isOneCloudVpcWire() bool {
	return IsOneCloudVpcResource(wire)
}

func (wire *SWire) getEnabledHosts() []SHost {
	hosts := make([]SHost, 0)

	hostQuery := HostManager.Query().SubQuery()
	hostNetifQuery := NetInterfaceManager.Query().SubQuery()

	q := hostQuery.Query()
	q = q.Join(hostNetifQuery, sqlchemy.Equals(hostQuery.Field("id"), hostNetifQuery.Field("baremetal_id")))
	q = q.Filter(sqlchemy.IsTrue(hostQuery.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(hostQuery.Field("host_status"), api.HOST_ONLINE))
	if wire.isOneCloudVpcWire() {
		q = q.Filter(sqlchemy.NOT(sqlchemy.IsNullOrEmpty(hostQuery.Field("ovn_version"))))
	} else {
		q = q.Filter(sqlchemy.Equals(hostNetifQuery.Field("wire_id"), wire.Id))
	}

	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		log.Errorf("getEnabledHosts fail %s", err)
		return nil
	}

	return hosts
}

func (wire *SWire) clearHostSchedDescCache() error {
	hosts := wire.getEnabledHosts()
	if hosts != nil {
		for i := 0; i < len(hosts); i += 1 {
			host := hosts[i]
			if err := host.ClearSchedDescCache(); err != nil {
				return errors.Wrapf(err, "wire %s clear host %s sched cache", wire.GetName(), host.GetName())
			}
		}
	}
	return nil
}

func (swire *SWire) GetIWire(ctx context.Context) (cloudprovider.ICloudWire, error) {
	vpc, err := swire.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	ivpc, err := vpc.GetIVpc(ctx)
	if err != nil {
		return nil, err
	}
	return ivpc.GetIWireById(swire.GetExternalId())
}

func (manager *SWireManager) FetchWireById(wireId string) *SWire {
	wireObj, err := manager.FetchById(wireId)
	if err != nil {
		log.Errorf("FetchWireById fail %s", err)
		return nil
	}
	return wireObj.(*SWire)
}

func (manager *SWireManager) FetchWireByExternalId(managerId, extId string) (*SWire, error) {
	wires, err := manager.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("manager_id", managerId).Equals("external_id", extId)
		return q
	})
	if err != nil {
		return nil, errors.Wrap(err, "FetchWires")
	}
	switch len(wires) {
	case 0:
		return nil, errors.Wrap(sql.ErrNoRows, "not found")
	case 1:
		return &wires[0], nil
	default:
		return nil, errors.Wrapf(httperrors.ErrDuplicateId, "duplicate wires externalId %s", extId)
	}
}

func (manager *SWireManager) GetOnPremiseWireOfIp(ipAddr string) (*SWire, error) {
	net, err := NetworkManager.GetOnPremiseNetworkOfIP(ipAddr, "", tristate.None)
	if err != nil {
		return nil, err
	}
	wire, _ := net.GetWire()
	if wire != nil {
		return wire, nil
	} else {
		return nil, fmt.Errorf("Wire not found")
	}
}

func (w *SWire) PerformMergeNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WireMergeNetworkInput) (jsonutils.JSONObject, error) {
	return nil, w.StartMergeNetwork(ctx, userCred, "")
}

func (sm *SWireManager) FetchByIdsOrNames(idOrNames []string) ([]SWire, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	q := sm.Query()
	if len(idOrNames) == 1 {
		q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("id"), idOrNames[0]), sqlchemy.Equals(q.Field("name"), idOrNames[0])))
	} else {
		q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("id"), stringutils2.RemoveUtf8Strings(idOrNames)), sqlchemy.In(q.Field("name"), idOrNames)))
	}
	ret := make([]SWire, 0, len(idOrNames))
	err := db.FetchModelObjects(sm, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (w *SWire) PerformMergeFrom(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WireMergeFromInput) (ret jsonutils.JSONObject, err error) {
	if len(input.Sources) == 0 {
		return nil, httperrors.NewMissingParameterError("sources")
	}
	defer func() {
		if err != nil {
			logclient.AddActionLogWithContext(ctx, w, logclient.ACT_MERGE, err.Error(), userCred, false)
		}
	}()

	wires, err := WireManager.FetchByIdsOrNames(input.Sources)
	if err != nil {
		return
	}
	wireIdOrNameSet := sets.NewString(input.Sources...)
	for i := range wires {
		id, name := wires[i].GetId(), wires[i].GetName()
		if wireIdOrNameSet.Has(id) {
			wireIdOrNameSet.Delete(id)
			continue
		}
		if wireIdOrNameSet.Has(name) {
			wireIdOrNameSet.Delete(name)
		}
	}
	if wireIdOrNameSet.Len() > 0 {
		return nil, httperrors.NewInputParameterError("invalid wire id or name %v", wireIdOrNameSet.UnsortedList())
	}

	lockman.LockClass(ctx, WireManager, db.GetLockClassKey(WireManager, userCred))
	defer lockman.ReleaseClass(ctx, WireManager, db.GetLockClassKey(WireManager, userCred))

	for _, tw := range wires {
		err = WireManager.handleWireIdChange(ctx, &wireIdChangeArgs{
			oldWire: &tw,
			newWire: w,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "unable to merge wire %s to %s", tw.GetId(), w.GetId())
		}
		if err = tw.Delete(ctx, userCred); err != nil {
			return nil, err
		}
	}
	logclient.AddActionLogWithContext(ctx, w, logclient.ACT_MERGE_FROM, "", userCred, true)
	if input.MergeNetwork {
		err = w.StartMergeNetwork(ctx, userCred, "")
		if err != nil {
			return nil, errors.Wrap(err, "unableto StartMergeNetwork")
		}
	}
	return
}

func (w *SWire) PerformMergeTo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WireMergeInput) (ret jsonutils.JSONObject, err error) {
	if len(input.Target) == 0 {
		return nil, httperrors.NewMissingParameterError("target")
	}
	defer func() {
		if err != nil {
			logclient.AddActionLogWithContext(ctx, w, logclient.ACT_MERGE, err.Error(), userCred, false)
		}
	}()
	iw, err := WireManager.FetchByIdOrName(userCred, input.Target)
	if err == sql.ErrNoRows {
		err = httperrors.NewNotFoundError("Wire %q", input.Target)
		return
	}
	if err != nil {
		return
	}

	tw := iw.(*SWire)
	lockman.LockClass(ctx, WireManager, db.GetLockClassKey(WireManager, userCred))
	defer lockman.ReleaseClass(ctx, WireManager, db.GetLockClassKey(WireManager, userCred))

	err = WireManager.handleWireIdChange(ctx, &wireIdChangeArgs{
		oldWire: w,
		newWire: tw,
	})
	if err != nil {
		return
	}
	logclient.AddActionLogWithContext(ctx, w, logclient.ACT_MERGE, "", userCred, true)
	if err = w.Delete(ctx, userCred); err != nil {
		return nil, err
	}
	if input.MergeNetwork {
		err = tw.StartMergeNetwork(ctx, userCred, "")
		if err != nil {
			return nil, errors.Wrap(err, "unableto StartMergeNetwork")
		}
	}
	return
}

func (w *SWire) StartMergeNetwork(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "NetworksUnderWireMergeTask", w, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (wm *SWireManager) handleWireIdChange(ctx context.Context, args *wireIdChangeArgs) error {
	handlers := []wireIdChangeHandler{
		// HostwireManager,
		NetworkManager,
		LoadbalancerClusterManager,
		NetInterfaceManager,
	}

	errs := []error{}
	for _, h := range handlers {
		if err := h.handleWireIdChange(ctx, args); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		err := errors.NewAggregate(errs)
		return httperrors.NewGeneralError(err)
	}
	return nil
}

func (wire *SWire) PerformSetClassMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformSetClassMetadataInput) (jsonutils.JSONObject, error) {
	vpc, err := wire.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get vpc of wire %s", wire.GetId())
	}
	if vpc.GetId() != api.DEFAULT_VPC_ID {
		ok, err := db.IsInSameClass(ctx, vpc, db.ClassMetadataOwner(input))
		if err != nil {
			return nil, errors.Wrapf(err, "unable to check if vpc and wire are in same class")
		}
		if !ok {
			return nil, httperrors.NewForbiddenError("the vpc %s and this wire have different class metadata", vpc.GetName())
		}
	}
	return wire.SStatusInfrasResourceBase.PerformSetClassMetadata(ctx, userCred, query, input)
}

// 二层网络列表
func (manager *SWireManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WireListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	{
		managedFilter := query.ManagedResourceListInput
		query.ManagedResourceListInput = api.ManagedResourceListInput{}
		q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
		}
		q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, managedFilter)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
		}
		query.ManagedResourceListInput = managedFilter
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}

	hostStr := query.HostId
	if len(hostStr) > 0 {
		hostObj, err := HostManager.FetchByIdOrName(userCred, hostStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), hostStr)
		}
		sq := NetInterfaceManager.Query("wire_id").Equals("baremetal_id", hostObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("id"), sq.SubQuery()))
	}
	if len(query.HostType) > 0 {
		hs := HostManager.Query("id").Equals("host_type", query.HostType).SubQuery()
		sq := NetInterfaceManager.Query("wire_id")
		sq = sq.Join(hs, sqlchemy.Equals(sq.Field("baremetal_id"), hs.Field("id")))
		q = q.Filter(sqlchemy.In(q.Field("id"), sq.SubQuery()))
	}

	if query.Bandwidth != nil {
		q = q.Equals("bandwidth", *query.Bandwidth)
	}

	return q, nil
}

func (manager *SWireManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WireListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, err = manager.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByNetworkCount}) {
		networkQ := NetworkManager.Query()
		networkQ = networkQ.AppendField(networkQ.Field("wire_id"), sqlchemy.COUNT("network_count"))
		networkQ = networkQ.GroupBy(networkQ.Field("wire_id"))
		networkSQ := networkQ.SubQuery()
		q = q.LeftJoin(networkSQ, sqlchemy.Equals(networkSQ.Field("wire_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(networkSQ.Field("network_count"))
		q = db.OrderByFields(q, []string{query.OrderByNetworkCount}, []sqlchemy.IQueryField{q.Field("network_count")})
	}
	return q, nil
}

func (manager *SWireManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

type SWireUsageCount struct {
	Id string
	api.WireUsage
}

func wmquery(manager db.IModelManager, uniqField, field string, wireIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query("wire_id", uniqField)

	if filter != nil {
		q = filter(q)
	}

	sq := q.Distinct().SubQuery()

	return sq.Query(
		sq.Field("wire_id"),
		sqlchemy.COUNT(field),
	).In("wire_id", wireIds).GroupBy(sq.Field("wire_id")).SubQuery()
}

func (manager *SWireManager) TotalResourceCount(wireIds []string) (map[string]api.WireUsage, error) {
	// network
	networkSQ := wmquery(NetworkManager, "id", "network_cnt", wireIds, nil)
	hostSQ := wmquery(NetInterfaceManager, "baremetal_id", "host_cnt", wireIds, nil)

	wires := manager.Query().SubQuery()
	wireQ := wires.Query(
		sqlchemy.SUM("networks", networkSQ.Field("network_cnt")),
		sqlchemy.SUM("host_count", hostSQ.Field("host_cnt")),
	)

	wireQ.AppendField(wireQ.Field("id"))

	wireQ = wireQ.LeftJoin(networkSQ, sqlchemy.Equals(wireQ.Field("id"), networkSQ.Field("wire_id")))
	wireQ = wireQ.LeftJoin(hostSQ, sqlchemy.Equals(wireQ.Field("id"), hostSQ.Field("wire_id")))

	wireQ = wireQ.Filter(sqlchemy.In(wireQ.Field("id"), wireIds)).GroupBy(wireQ.Field("id"))

	wireCount := []SWireUsageCount{}
	err := wireQ.All(&wireCount)
	if err != nil {
		return nil, errors.Wrapf(err, "wireQ.All")
	}

	result := map[string]api.WireUsage{}
	for i := range wireCount {
		result[wireCount[i].Id] = wireCount[i].WireUsage
	}

	return result, nil
}

func (manager *SWireManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WireDetails {
	rows := make([]api.WireDetails, len(objs))

	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	managerList := make([]interface{}, len(rows))
	wireIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.WireDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			VpcResourceInfo:                 vpcRows[i],
			ZoneResourceInfoBase:            zoneRows[i].ZoneResourceInfoBase,
		}
		wire := objs[i].(*SWire)
		wireIds[i] = wire.Id
		managerList[i] = &SManagedResourceBase{wire.ManagerId}
	}

	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	usage, err := manager.TotalResourceCount(wireIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].WireUsage = usage[wireIds[i]]
		rows[i].ManagedResourceInfo = managerRows[i]
	}

	return rows
}

func (man *SWireManager) removeWiresByVpc(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc) error {
	wires, err := man.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("vpc_id", vpc.Id)
		return q
	})
	if err != nil {
		return errors.Wrap(err, "FetchWires")
	}
	var errs []error
	for i := range wires {
		wire := &wires[i]
		if err := wire.Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

/*func (swire *SWire) IsManaged() bool {
	vpc, _ := swire.GetVpc()
	if vpc == nil {
		return false
	}
	return vpc.IsManaged()
}*/

func (model *SWire) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	vpc, _ := model.GetVpc()
	if !data.Contains("public_scope") {
		if !model.IsManaged() && db.IsAdminAllowPerform(ctx, userCred, model, "public") && ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() && vpc != nil && vpc.IsPublic && vpc.PublicScope == string(rbacscope.ScopeSystem) {
			model.SetShare(rbacscope.ScopeSystem)
		} else {
			model.SetShare(rbacscope.ScopeNone)
		}
		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(model.PublicScope))
	}
	model.Status = api.WIRE_STATUS_AVAILABLE
	model.ManagerId = vpc.ManagerId
	return model.SInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (model *SWire) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	vpc, err := model.GetVpc()
	if err != nil {
		log.Errorf("unable to getvpc of wire %s: %s", model.GetId(), vpc.GetId())
	}
	err = db.InheritFromTo(ctx, userCred, vpc, model)
	if err != nil {
		log.Errorf("unable to inhert vpc to model %s: %s", model.GetId(), err.Error())
	}
}

func (wire *SWire) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{}
	vpc, _ := wire.GetVpc()
	if vpc != nil {
		candidates = append(candidates,
			vpc.GetChangeOwnerCandidateDomainIds(),
			db.ISharableChangeOwnerCandidateDomainIds(vpc))
	}
	return db.ISharableMergeChangeOwnerCandidateDomainIds(wire, candidates...)
}

func (wire *SWire) GetChangeOwnerRequiredDomainIds() []string {
	requires := stringutils2.SSortedStrings{}
	networks, _ := wire.getNetworks(nil, nil, rbacscope.ScopeNone)
	for i := range networks {
		requires = stringutils2.Append(requires, networks[i].DomainId)
	}
	return requires
}

func (wire *SWire) GetRequiredSharedDomainIds() []string {
	networks, _ := wire.getNetworks(nil, nil, rbacscope.ScopeNone)
	if len(networks) == 0 {
		return wire.SInfrasResourceBase.GetRequiredSharedDomainIds()
	}
	requires := make([][]string, len(networks))
	for i := range networks {
		requires[i] = db.ISharableChangeOwnerCandidateDomainIds(&networks[i])
	}
	return db.ISharableMergeShareRequireDomainIds(requires...)
}

func (manager *SWireManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (swire *SWire) GetDetailsTopology(ctx context.Context, userCred mcclient.TokenCredential, input *api.WireTopologyInput) (*api.WireTopologyOutput, error) {
	ret := &api.WireTopologyOutput{
		Name:      swire.Name,
		Status:    swire.Status,
		Bandwidth: swire.Bandwidth,
		Networks:  []api.NetworkTopologyOutput{},
		Hosts:     []api.HostTopologyOutput{},
	}
	if len(swire.ZoneId) > 0 {
		zone, _ := swire.GetZone()
		if zone != nil {
			ret.Zone = zone.Name
		}
	}
	hosts, err := swire.GetHosts()
	if err != nil {
		return nil, errors.Wrapf(err, "GetHosts for wire %s", swire.Id)
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
		ret.Hosts = append(ret.Hosts, host)
	}
	networks, err := swire.GetNetworks(nil, nil, rbacscope.ScopeSystem)
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

		network.GetNetworkAddressesOutput, err = networks[j].fetchAddressDetails(userCred, userCred, rbacscope.ScopeSystem)
		if err != nil {
			return nil, errors.Wrapf(err, "fetchAddressDetails")
		}

		ret.Networks = append(ret.Networks, network)
	}
	return ret, nil
}

func (wire *SWire) GetCloudproviderId() string {
	return wire.SManagedResourceBase.GetCloudproviderId()
}

func (wire *SWire) GetCloudprovider() *SCloudprovider {
	return wire.SManagedResourceBase.GetCloudprovider()
}

func (wire *SWire) GetProviderName() string {
	return wire.SManagedResourceBase.GetProviderName()
}

func (wire *SWire) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := NetworkAdditionalWireManager.DeleteWire(ctx, wire.Id); err != nil {
		return errors.Wrap(err, "NetworkAdditionalWireManager.DeleteWire")
	}
	return wire.SStatusInfrasResourceBase.Delete(ctx, userCred)
}
