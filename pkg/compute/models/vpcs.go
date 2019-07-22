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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVpcManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var VpcManager *SVpcManager

func init() {
	VpcManager = &SVpcManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SVpc{},
			"vpcs_tbl",
			"vpc",
			"vpcs",
		),
	}
	VpcManager.SetVirtualObject(VpcManager)
}

type SVpc struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	IsDefault bool `default:"false" list:"admin" create:"admin_optional"`

	CidrBlock string `charset:"ascii" nullable:"true" list:"admin" create:"admin_required"`

	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
}

func (manager *SVpcManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SVpcManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SVpcManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SVpc) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SVpc) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SVpc) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SVpc) GetCloudRegionId() string {
	if len(self.CloudregionId) == 0 {
		return api.DEFAULT_REGION_ID
	} else {
		return self.CloudregionId
	}
}

func (self *SVpc) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		self.Id = idstr
	}
	return self.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SVpc) getNatgatewayQuery() *sqlchemy.SQuery {
	return NatGatewayManager.Query().Equals("vpc_id", self.Id)
}

func (self *SVpc) GetNatgatewayCount() (int, error) {
	return self.getNatgatewayQuery().CountWithError()
}

func (self *SVpc) GetNatgateways() ([]SNatGateway, error) {
	nats := []SNatGateway{}
	err := db.FetchModelObjects(NatGatewayManager, self.getNatgatewayQuery(), &nats)
	if err != nil {
		return nil, err
	}
	return nats, nil
}

func (self *SVpc) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetNetworkCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetNetworkCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("VPC not empty, please delete network first")
	}
	cnt, err = self.GetNatgatewayCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetNatgatewayCount fail %v", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("VPC not empty, please delete nat gateway first")
	}
	if self.Id == api.DEFAULT_VPC_ID {
		return httperrors.NewProtectedResourceError("not allow to delete default vpc")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SVpc) getWireQuery() *sqlchemy.SQuery {
	wires := WireManager.Query()
	if self.Id == api.DEFAULT_VPC_ID {
		return wires.Filter(sqlchemy.OR(sqlchemy.IsNull(wires.Field("vpc_id")),
			sqlchemy.IsEmpty(wires.Field("vpc_id")),
			sqlchemy.Equals(wires.Field("vpc_id"), self.Id)))
	} else {
		return wires.Equals("vpc_id", self.Id)
	}
}

func (self *SVpc) GetWireCount() (int, error) {
	q := self.getWireQuery()
	return q.CountWithError()
}

func (self *SVpc) GetWires() []SWire {
	wires := make([]SWire, 0)
	q := self.getWireQuery()
	err := db.FetchModelObjects(WireManager, q, &wires)
	if err != nil {
		log.Errorf("getWires fail %s", err)
		return nil
	}
	return wires
}

func (self *SVpc) getNetworkQuery() *sqlchemy.SQuery {
	q := NetworkManager.Query()
	wireQ := self.getWireQuery().SubQuery()
	q = q.In("wire_id", wireQ.Query(wireQ.Field("id")).SubQuery())
	return q
}

func (self *SVpc) GetNetworkCount() (int, error) {
	q := self.getNetworkQuery()
	return q.CountWithError()
}

func (self *SVpc) GetRouteTableQuery() *sqlchemy.SQuery {
	return RouteTableManager.Query().Equals("vpc_id", self.Id)
}

func (self *SVpc) GetRouteTables() []SRouteTable {
	q := self.GetRouteTableQuery()
	routes := []SRouteTable{}
	db.FetchModelObjects(RouteTableManager, q, &routes)
	return routes
}

func (self *SVpc) GetRouteTableCount() (int, error) {
	return self.GetRouteTableQuery().CountWithError()
}

func (self *SVpc) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	cnt, _ := self.GetWireCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "wire_count")
	cnt, _ = self.GetNetworkCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "network_count")
	cnt, _ = self.GetRouteTableCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "routetable_count")
	cnt, _ = self.GetNatgatewayCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "natgateway_count")
	/* region, err := self.GetRegion()
	if err != nil {
		log.Errorf("failed getting region for vpc %s(%s)", self.Name, self.Id)
		return extra
	}
	extra.Add(jsonutils.NewString(region.GetName()), "region")
	if len(region.GetExternalId()) > 0 {
		extra.Add(jsonutils.NewString(region.GetExternalId()), "region_external_id")
	}*/

	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	return extra
}

func (self *SVpc) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := self.GetRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

func (self *SVpc) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, err
	}
	return region.(*SCloudregion), nil
}

func (self *SVpc) getZoneByExternalId(externalId string) (*SZone, error) {
	region, err := self.GetRegion()
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

func (self *SVpc) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SVpc) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (manager *SVpcManager) getVpcsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(manager, q, &vpcs)
	if err != nil {
		return nil, err
	}
	return vpcs, nil
}

func (self *SVpc) setDefault(def bool) error {
	var err error
	if self.IsDefault != def {
		_, err = db.Update(self, func() error {
			self.IsDefault = def
			return nil
		})
	}
	return err
}

func (manager *SVpcManager) SyncVPCs(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, vpcs []cloudprovider.ICloudVpc) ([]SVpc, []cloudprovider.ICloudVpc, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	localVPCs := make([]SVpc, 0)
	remoteVPCs := make([]cloudprovider.ICloudVpc, 0)
	syncResult := compare.SyncResult{}

	dbVPCs, err := manager.getVpcsByRegion(region, provider)
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
		err = commondb[i].SyncWithCloudVpc(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localVPCs = append(localVPCs, commondb[i])
			remoteVPCs = append(remoteVPCs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudVpc(ctx, userCred, added[i], provider, region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localVPCs = append(localVPCs, *new)
			remoteVPCs = append(remoteVPCs, added[i])
			syncResult.Add()
		}
	}

	return localVPCs, remoteVPCs, syncResult
}

func (self *SVpc) syncRemoveCloudVpc(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		self.markAllNetworksUnknown(userCred)
		_, err = self.PerformDisable(ctx, userCred, nil, nil)
		if err == nil {
			err = self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
		}
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

func (self *SVpc) SyncWithCloudVpc(ctx context.Context, userCred mcclient.TokenCredential, extVPC cloudprovider.ICloudVpc) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		extVPC.Refresh()
		// self.Name = extVPC.GetName()
		self.Status = extVPC.GetStatus()
		self.CidrBlock = extVPC.GetCidrBlock()
		self.IsDefault = extVPC.GetIsDefault()
		self.ExternalId = extVPC.GetGlobalId()

		self.IsEmulated = extVPC.IsEmulated()

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SVpcManager) newFromCloudVpc(ctx context.Context, userCred mcclient.TokenCredential, extVPC cloudprovider.ICloudVpc, provider *SCloudprovider, region *SCloudregion) (*SVpc, error) {
	vpc := SVpc{}
	vpc.SetModelManager(manager, &vpc)

	newName, err := db.GenerateName(manager, userCred, extVPC.GetName())
	if err != nil {
		return nil, err
	}
	vpc.Name = newName
	vpc.Status = extVPC.GetStatus()
	vpc.ExternalId = extVPC.GetGlobalId()
	vpc.IsDefault = extVPC.GetIsDefault()
	vpc.CidrBlock = extVPC.GetCidrBlock()
	vpc.CloudregionId = region.Id

	vpc.ManagerId = provider.Id

	vpc.IsEmulated = extVPC.IsEmulated()

	err = manager.TableSpec().Insert(&vpc)
	if err != nil {
		log.Errorf("newFromCloudVpc fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&vpc, db.ACT_CREATE, vpc.GetShortDesc(ctx), userCred)

	return &vpc, nil
}

func (self *SVpc) markAllNetworksUnknown(userCred mcclient.TokenCredential) error {
	wires := self.GetWires()
	if wires == nil || len(wires) == 0 {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wires[i].markNetworkUnknown(userCred)
	}
	return nil
}

func (manager *SVpcManager) InitializeData() error {
	vpcObj, err := manager.FetchById(api.DEFAULT_VPC_ID)
	if err != nil {
		if err == sql.ErrNoRows {
			defVpc := SVpc{}
			defVpc.SetModelManager(VpcManager, &defVpc)

			defVpc.Id = api.DEFAULT_VPC_ID
			defVpc.Name = "Default"
			defVpc.CloudregionId = api.DEFAULT_REGION_ID
			defVpc.Description = "Default VPC"
			defVpc.Status = api.VPC_STATUS_AVAILABLE
			defVpc.IsDefault = true
			err = manager.TableSpec().Insert(&defVpc)
			if err != nil {
				log.Errorf("Insert default vpc fail: %s", err)
			}
			return err
		} else {
			return err
		}
	} else {
		vpc := vpcObj.(*SVpc)
		if vpc.Status != api.VPC_STATUS_AVAILABLE {
			_, err = db.Update(vpc, func() error {
				vpc.Status = api.VPC_STATUS_AVAILABLE
				return nil
			})
			return err
		}
	}
	return nil
}

func (manager *SVpcManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionId, err := data.GetString("cloudregion_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("cloudregion_id")
	}
	region := CloudregionManager.FetchRegionById(regionId)
	if region == nil {
		return nil, httperrors.NewInputParameterError("Invalid cloudregion_id")
	}
	if region.isManaged() {
		managerStr := jsonutils.GetAnyString(data, []string{"manager_id", "manager"})
		if len(managerStr) == 0 {
			return nil, httperrors.NewMissingParameterError("manager_id")
		}
		managerObj := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Cloud provider/manager %s not found", managerStr)
		}
		data.Add(jsonutils.NewString(managerObj.GetId()), "manager_id")
	} else {
		return nil, httperrors.NewNotImplementedError("Cannot create VPC in private cloud")
	}

	cidrBlock, _ := data.GetString("cidr_block")
	if len(cidrBlock) > 0 {
		blocks := strings.Split(cidrBlock, ",")
		for _, block := range blocks {
			_, err = netutils.NewIPV4Prefix(block)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid cidr_block %s", cidrBlock)
			}
		}
	}
	data, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateCreateVpcData(ctx, userCred, data)
}

func (self *SVpc) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if len(self.ManagerId) == 0 {
		return
	}
	task, err := taskman.TaskManager.NewTask(ctx, "VpcCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("VpcCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (self *SVpc) GetIRegion() (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, err
	}
	provider, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SVpc) GetIVpc() (cloudprovider.ICloudVpc, error) {
	provider, err := self.GetDriver()
	if err != nil {
		log.Errorf("fail to find cloud provider")
		return nil, err
	}
	var iregion cloudprovider.ICloudRegion
	if provider.GetFactory().IsOnPremise() {
		iregion, err = provider.GetOnPremiseIRegion()
	} else {
		region, err := self.GetRegion()
		if err != nil {
			return nil, err
		}
		iregion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		log.Errorf("fail to find iregion: %s", err)
		return nil, err
	}
	ivpc, err := iregion.GetIVpcById(self.ExternalId)
	if err != nil {
		log.Errorf("fail to find ivpc by id %s %s", self.ExternalId, err)
		return nil, err
	}
	return ivpc, nil
}

func (self *SVpc) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SVpc delete do nothing")
	self.SetStatus(userCred, api.VPC_STATUS_START_DELETE, "")
	return nil
}

func (self *SVpc) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.StartDeleteVpcTask(ctx, userCred)
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SVpc) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	self.SetStatus(userCred, api.VPC_STATUS_DELETED, "real delete")
	routes := self.GetRouteTables()
	for i := 0; i < len(routes); i++ {
		routes[i].RealDelete(ctx, userCred)
	}
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SVpc) StartDeleteVpcTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "VpcDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start vpcdeleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVpc) getPrefix() []netutils.IPV4Prefix {
	if len(self.CidrBlock) > 0 {
		ret := []netutils.IPV4Prefix{}
		blocks := strings.Split(self.CidrBlock, ",")
		for _, block := range blocks {
			prefix, _ := netutils.NewIPV4Prefix(block)
			ret = append(ret, prefix)
		}
		return ret
	}
	return []netutils.IPV4Prefix{{}}
}

func (self *SVpc) getIPRanges() []netutils.IPV4AddrRange {
	ret := []netutils.IPV4AddrRange{}
	prefs := self.getPrefix()
	for _, pref := range prefs {
		ret = append(ret, pref.ToIPRange())
	}

	return ret
}

func (self *SVpc) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SVpc) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	provider := self.GetCloudprovider()
	if provider != nil {
		if provider.Enabled {
			return nil, httperrors.NewInvalidStatusError("Cannot purge vpc on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (manager *SVpcManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = managedResourceFilterByDomain(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (self *SVpc) SyncRemoteWires(ctx context.Context, userCred mcclient.TokenCredential) error {
	ivpc, err := self.GetIVpc()
	if err != nil {
		return err
	}

	provider := CloudproviderManager.FetchCloudproviderById(self.ManagerId)
	syncVpcWires(ctx, userCred, nil, provider, self, ivpc, &SSyncRange{})

	hosts := HostManager.GetHostsByManagerAndRegion(provider.Id, self.CloudregionId)
	for i := 0; i < len(hosts); i += 1 {
		ihost, err := hosts[i].GetIHost()
		if err != nil {
			return err
		}
		syncHostWires(ctx, userCred, nil, provider, &hosts[i], ihost)
	}
	return nil
}
