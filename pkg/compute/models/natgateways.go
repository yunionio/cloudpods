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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNatGatewayManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager

	SDeletePreventableResourceBaseManager
}

var NatGatewayManager *SNatGatewayManager

func init() {
	NatGatewayManager = &SNatGatewayManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SNatGateway{},
			"natgateways_tbl",
			"natgateway",
			"natgateways",
		),
	}
	NatGatewayManager.SetVirtualObject(NatGatewayManager)
}

type SNatGateway struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SBillingResourceBase
	SVpcResourceBase

	SDeletePreventableResourceBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	IpAddr    string `width:"16" charset:"ascii" nullable:"false" list:"user"`

	BandwidthMb int    `nullable:"false" list:"user"`
	NatSpec     string `list:"user" create:"optional"` // NAT规格
}

func (manager *SNatGatewayManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager, VpcManager},
	}
}

// NAT网关列表
func (man *SNatGatewayManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatGetewayListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = man.SDeletePreventableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DeletePreventableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDeletePreventableResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

// NAT网关列表
func (man *SNatGatewayManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatGetewayListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SNatGatewayManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SNatGatewayManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NatgatewayCreateInput) (api.NatgatewayCreateInput, error) {
	if len(input.NetworkId) == 0 {
		return input, httperrors.NewMissingParameterError("network_id")
	}
	_network, err := validators.ValidateModel(userCred, NetworkManager, &input.NetworkId)
	if err != nil {
		return input, err
	}
	network := _network.(*SNetwork)
	vpc := network.GetVpc()
	if vpc == nil {
		return input, httperrors.NewGeneralError(errors.Errorf("failed to get network %s %s vpc", network.Name, network.Id))
	}
	input.VpcId = vpc.Id
	region, err := vpc.GetRegion()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "vpc.GetRegion"))
	}
	if len(input.Duration) > 0 {
		billingCycle, err := billing.ParseBillingCycle(input.Duration)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid duration %s", input.Duration)
		}

		if !utils.IsInStringArray(input.BillingType, []string{billing_api.BILLING_TYPE_PREPAID, billing_api.BILLING_TYPE_POSTPAID}) {
			input.BillingType = billing_api.BILLING_TYPE_PREPAID
		}

		if input.BillingType == billing_api.BILLING_TYPE_PREPAID {
			if !region.GetDriver().IsSupportedBillingCycle(billingCycle, man.KeywordPlural()) {
				return input, httperrors.NewInputParameterError("unsupported duration %s", input.Duration)
			}
		}
		tm := time.Time{}
		input.BillingCycle = billingCycle.String()
		input.ExpiredAt = billingCycle.EndAt(tm)
	}
	if len(input.Eip) > 0 || input.EipBw > 0 {
		if len(input.Eip) > 0 {
			_eip, err := validators.ValidateModel(userCred, ElasticipManager, &input.Eip)
			if err != nil {
				return input, err
			}
			eip := _eip.(*SElasticip)
			if eip.Status != api.EIP_STATUS_READY {
				return input, httperrors.NewInvalidStatusError("eip %s status invalid %s", input.Eip, eip.Status)
			}

			if eip.IsAssociated() {
				return input, httperrors.NewResourceBusyError("eip %s has been associated", input.Eip)
			}

			if eip.CloudregionId != vpc.CloudregionId {
				return input, httperrors.NewDuplicateResourceError("elastic ip %s and vpc %s not in same region", eip.Name, vpc.Name)
			}

			provider := eip.GetCloudprovider()
			if provider != nil && provider.Id != vpc.ManagerId {
				return input, httperrors.NewConflictError("cannot assoicate with eip %s: different cloudprovider", eip.Id)
			}
		} else {
			// create new
		}
	}
	input.StatusInfrasResourceBaseCreateInput, err = man.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	driver := region.GetDriver()
	return driver.ValidateCreateNatGateway(ctx, userCred, input)
}

func (self *SNatGateway) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	err := self.StartNatGatewayCreateTask(ctx, userCred, data.(*jsonutils.JSONDict))
	if err != nil {
		self.SetStatus(userCred, api.NAT_STATUS_CREATE_FAILED, err.Error())
		return
	}
	self.SetStatus(userCred, api.NAT_STATUS_ALLOCATE, "start allocate")
}

func (self *SNatGateway) StartNatGatewayCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "NatGatewayCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SNatGateway) AllowPerformSnatResources(ctx context.Context, userCred mcclient.TokenCredential,
	qurey jsonutils.JSONObject) bool {

	return true
}

func (self *SNatGateway) PerformSnatResources(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	q := NatSEntryManager.Query("ip", "network_id").Equals("natgateway_id", self.Id)

	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrapf(err, "fetch resource with natgateway_id %s error", self.Id)
	}
	ipset, ip := make(map[string]struct{}), ""
	networks, network := make([]string, 0), ""
	for rows.Next() {
		err := rows.Scan(&ip, &network)
		if err != nil {
			return nil, err
		}
		if _, ok := ipset[ip]; !ok {
			ipset[ip] = struct{}{}
		}
		networks = append(networks, network)
	}
	ips := make([]string, 0, len(ipset))
	for ip := range ipset {
		ips = append(ips, ip)
	}

	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(ips), "eips")
	ret.Add(jsonutils.Marshal(networks), "networks")

	return ret, nil
}

func (self *SNatGateway) AllowPerformDnatResources(ctx context.Context, userCred mcclient.TokenCredential,
	qurey jsonutils.JSONObject) bool {

	return true
}
func (self *SNatGateway) PerformDnatResources(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	q := NatDEntryManager.Query("external_ip").Equals("natgateway_id", self.Id)

	ips, err := self.extractEipAddr(q)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(ips), "eips")
	return ret, nil
}

func (self *SNatGateway) extractEipAddr(q *sqlchemy.SQuery) ([]string, error) {
	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrapf(err, "fetch resource with natgateway_id %s error", self.Id)
	}
	ipset, ip := make(map[string]struct{}), ""
	for rows.Next() {
		err := rows.Scan(&ip)
		if err != nil {
			return nil, err
		}
		if _, ok := ipset[ip]; !ok {
			ipset[ip] = struct{}{}
		}
	}
	ips := make([]string, 0, len(ipset))
	for ip := range ipset {
		ips = append(ips, ip)
	}

	return ips, nil
}

func (manager *SNatGatewayManager) getNatgatewaysByProviderId(providerId string) ([]SNatGateway, error) {
	nats := []SNatGateway{}
	err := fetchByVpcManagerId(manager, providerId, &nats)
	if err != nil {
		return nil, err
	}
	return nats, nil
}

func (self *SNatGateway) GetDTable() ([]SNatDEntry, error) {
	tables := []SNatDEntry{}
	q := NatDEntryManager.Query().Equals("natgateway_id", self.Id)
	err := db.FetchModelObjects(NatDEntryManager, q, &tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}

func (self *SNatGateway) GetSTable() ([]SNatSEntry, error) {
	tables := []SNatSEntry{}
	q := NatSEntryManager.Query().Equals("natgateway_id", self.Id)
	err := db.FetchModelObjects(NatSEntryManager, q, &tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}

func (self *SNatGateway) GetSTableSize(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) (int, error) {
	q := NatSEntryManager.Query().Equals("natgateway_id", self.Id)
	q = filter(q)
	return q.CountWithError()
}

func (self *SNatGateway) GetDTableSize(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) (int, error) {
	q := NatDEntryManager.Query().Equals("natgateway_id", self.Id)
	q = filter(q)
	return q.CountWithError()
}

func (self *SNatGateway) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.NatgatewayDetails, error) {
	return api.NatgatewayDetails{}, nil
}

func (manager SNatGatewayManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NatgatewayDetails {
	rows := make([]api.NatgatewayDetails, len(objs))
	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.NatgatewayDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			VpcResourceInfo:                 vpcRows[i],
		}
		nat := objs[i].(*SNatGateway)
		netIds[i] = nat.NetworkId
	}

	netMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 for nat network")
		return rows
	}
	for i := range rows {
		rows[i].Network, _ = netMaps[netIds[i]]
	}
	return rows
}

func (manager *SNatGatewayManager) SyncNatGateways(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, vpc *SVpc, cloudNatGateways []cloudprovider.ICloudNatGateway) ([]SNatGateway, []cloudprovider.ICloudNatGateway, compare.SyncResult) {
	lockman.LockRawObject(ctx, "natgateways", vpc.Id)
	defer lockman.ReleaseRawObject(ctx, "natgateways", vpc.Id)

	localNatGateways := make([]SNatGateway, 0)
	remoteNatGateways := make([]cloudprovider.ICloudNatGateway, 0)
	syncResult := compare.SyncResult{}

	dbNatGateways, err := vpc.GetNatgateways()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SNatGateway, 0)
	commondb := make([]SNatGateway, 0)
	commonext := make([]cloudprovider.ICloudNatGateway, 0)
	added := make([]cloudprovider.ICloudNatGateway, 0)
	if err := compare.CompareSets(dbNatGateways, cloudNatGateways, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudNatGateway(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudNatGateway(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localNatGateways = append(localNatGateways, commondb[i])
		remoteNatGateways = append(remoteNatGateways, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		routeTableNew, err := manager.newFromCloudNatGateway(ctx, userCred, syncOwnerId, provider, vpc, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, routeTableNew, added[i])
		localNatGateways = append(localNatGateways, *routeTableNew)
		remoteNatGateways = append(remoteNatGateways, added[i])
		syncResult.Add()
	}
	return localNatGateways, remoteNatGateways, syncResult
}

func (self *SNatGateway) syncRemoveCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	self.DeletePreventionOff(self, userCred)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.NAT_STATUS_UNKNOWN, "sync to delete")
	}
	return self.purge(ctx, userCred)
}

func (self *SNatGateway) ValidateDeleteCondition(ctx context.Context) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("Nat is locked, cannot delete")
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SNatGateway) SyncWithCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extNat cloudprovider.ICloudNatGateway) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extNat.GetStatus()
		self.NatSpec = extNat.GetNatSpec()
		self.BandwidthMb = extNat.GetBandwidthMb()

		vpc, err := self.GetVpc()
		if err != nil {
			return errors.Wrapf(err, "GetVpc")
		}
		if networId := extNat.GetINetworkId(); len(networId) > 0 {
			_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				sq := WireManager.Query("id").Equals("vpc_id", vpc.Id).SubQuery()
				return q.In("wire_id", sq)
			})
			if err != nil {
				log.Errorf("failed to found nat %s network by external id %s", self.Name, networId)
			} else {
				network := _network.(*SNetwork)
				self.NetworkId = network.Id
				self.IpAddr = extNat.GetIpAddr()
			}
		}

		factory, _ := provider.GetProviderFactory()
		if factory.IsSupportPrepaidResources() {
			self.BillingType = extNat.GetBillingType()
			if expired := extNat.GetExpiredAt(); !expired.IsZero() {
				self.ExpiredAt = expired
			}
			self.AutoRenew = extNat.IsAutoRenew()
		}

		return nil
	})
	if err != nil {
		return err
	}

	SyncCloudDomain(userCred, self, provider.GetOwnerId())

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatGatewayManager) newFromCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, vpc *SVpc, extNat cloudprovider.ICloudNatGateway) (*SNatGateway, error) {
	nat := SNatGateway{}
	nat.SetModelManager(manager, &nat)

	nat.VpcId = vpc.Id
	nat.Status = extNat.GetStatus()
	nat.NatSpec = extNat.GetNatSpec()
	nat.BandwidthMb = extNat.GetBandwidthMb()
	if createdAt := extNat.GetCreatedAt(); !createdAt.IsZero() {
		nat.CreatedAt = extNat.GetCreatedAt()
	}
	nat.ExternalId = extNat.GetGlobalId()
	nat.IsEmulated = extNat.IsEmulated()

	factory, _ := provider.GetProviderFactory()
	if factory.IsSupportPrepaidResources() {
		nat.BillingType = extNat.GetBillingType()
		if expired := extNat.GetExpiredAt(); !expired.IsZero() {
			nat.ExpiredAt = expired
		}
		nat.AutoRenew = extNat.IsAutoRenew()
	}
	if networId := extNat.GetINetworkId(); len(networId) > 0 {
		_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := WireManager.Query("id").Equals("vpc_id", vpc.Id).SubQuery()
			return q.In("wire_id", sq)
		})
		if err != nil {
			log.Errorf("failed to found nat %s network by external id %s", nat.Name, networId)
		} else {
			network := _network.(*SNetwork)
			nat.NetworkId = network.Id
			nat.IpAddr = extNat.GetIpAddr()
		}
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, ownerId, extNat.GetName())
		if err != nil {
			return errors.Wrap(err, "db.GenerateName")
		}
		nat.Name = newName

		return manager.TableSpec().Insert(ctx, &nat)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	SyncCloudDomain(userCred, &nat, provider.GetOwnerId())

	db.OpsLog.LogEvent(&nat, db.ACT_CREATE, nat.GetShortDesc(ctx), userCred)

	return &nat, nil
}

// 删除NAT
func (self *SNatGateway) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query api.ServerDeleteInput, input api.NatgatewayDeleteInput) error {
	if !input.Force {
		eips, err := self.GetEips()
		if err != nil {
			return errors.Wrapf(err, "self.GetEips")
		}
		if len(eips) > 0 {
			return httperrors.NewNotEmptyError("natgateway has bind %d eips", len(eips))
		}
		dnat, err := self.GetDTable()
		if err != nil {
			return errors.Wrapf(err, "GetDTable()")
		}
		if len(dnat) > 0 {
			return httperrors.NewNotEmptyError("natgateway has %d stable", len(dnat))
		}
		snat, err := self.GetSTable()
		if err != nil {
			return errors.Wrapf(err, "GetSTable")
		}
		if len(snat) > 0 {
			return httperrors.NewNotEmptyError("natgateway has %d dtable", len(snat))
		}
	}
	err := self.StartNatGatewayDeleteTask(ctx, userCred, nil)
	if err != nil {
		return err
	}
	self.SetStatus(userCred, api.NAT_STATUS_DELETING, jsonutils.Marshal(input).String())
	return nil
}

func (self *SNatGateway) StartNatGatewayDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "NatGatewayDeleteTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SNatGateway) GetEips() ([]SElasticip, error) {
	q := ElasticipManager.Query().Equals("associate_id", self.Id)
	eips := []SElasticip{}
	err := db.FetchModelObjects(ElasticipManager, q, &eips)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return eips, nil
}

func (self *SNatGateway) SyncNatGatewayEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extEips []cloudprovider.ICloudEIP) compare.SyncResult {
	lockman.LockRawObject(ctx, "elasticip", self.Id)
	defer lockman.ReleaseRawObject(ctx, "elasticip", self.Id)

	result := compare.SyncResult{}

	dbEips, err := self.GetEips()
	if err != nil {
		result.AddError(err)
		return result
	}

	removed := make([]SElasticip, 0)
	commondb := make([]SElasticip, 0)
	commonext := make([]cloudprovider.ICloudEIP, 0)
	added := make([]cloudprovider.ICloudEIP, 0)
	if err := compare.CompareSets(dbEips, extEips, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].Dissociate(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(added); i += 1 {
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, added[i], provider, self.GetRegion(), provider.GetOwnerId())
		if err != nil {
			result.AddError(err)
			continue
		}
		if len(neip.AssociateId) > 0 && neip.AssociateId != self.Id {
			err = neip.Dissociate(ctx, userCred)
			if err != nil {
				result.AddError(err)
				continue
			}
		}
		err = neip.AssociateNatGateway(ctx, userCred, self)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SNatGateway) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

// 同步NAT网关状态
func (self *SNatGateway) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.NatGatewaySyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Nat gateway has %d task active, can't sync status", count)
	}

	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SNatGateway) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "NatGatewaySyncstatusTask", parentTaskId)
}

func (self *SNatGateway) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch vpc by ID %s failed", self.VpcId)
	}
	return vpc.(*SVpc), nil
}

func (self *SNatGateway) GetINatGateway() (cloudprovider.ICloudNatGateway, error) {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrap(err, "GetVpc")
	}
	iVpc, err := vpc.GetIVpc()
	if err != nil {
		return nil, errors.Wrap(err, "vpc.GetIVpc")
	}
	iNats, err := iVpc.GetINatGateways()
	if err != nil {
		return nil, errors.Wrapf(err, "iVpc.GetINatGateways")
	}
	for i := range iNats {
		if iNats[i].GetGlobalId() == self.ExternalId {
			return iNats[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (self *SNatGateway) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SNatGateway) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	dnats, err := self.GetDTable()
	if err != nil {
		return errors.Wrap(err, "fetch dnat table failed")
	}
	snats, err := self.GetSTable()
	if err != nil {
		return errors.Wrap(err, "fetch snat table failed")
	}
	for i := range dnats {
		err = dnats[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "delete dnat %s failed", dnats[i].GetId())
		}
	}
	for i := range snats {
		err = snats[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "delete snat %s failed", snats[i].GetId())
		}
	}
	return self.SInfrasResourceBase.Delete(ctx, userCred)
}

type SNatEntryManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SNatgatewayResourceBaseManager
}

func NewNatEntryManager(dt interface{}, tableName string, keyword string, keywordPlural string) SNatEntryManager {
	return SNatEntryManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

type SNatEntry struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SNatgatewayResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SNatEntryManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{NatGatewayManager},
	}
}

// NAT网关转发规则列表
func (man *SNatEntryManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatEntryListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SNatgatewayResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NatGatewayFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNatgatewayResourceBaseManager.ListItemFilter")
	}

	q, err = managedResourceFilterByAccount(q, query.ManagedResourceListInput, "natgateway_id", func() *sqlchemy.SQuery {
		natgateways := NatGatewayManager.Query().SubQuery()
		return natgateways.Query(natgateways.Field("id"))
	})
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByAccount")
	}

	return q, nil
}

func (man *SNatEntryManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatEntryListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SNatgatewayResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NatGatewayFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNatgatewayResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SNatEntryManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SNatgatewayResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (entry *SNatEntry) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.NatEntryDetails, error) {
	return api.NatEntryDetails{}, nil
}

func (manager *SNatEntryManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NatEntryDetails {
	rows := make([]api.NatEntryDetails, len(objs))
	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	natRows := manager.SNatgatewayResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.NatEntryDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			NatGatewayResourceInfo:          natRows[i],
		}
	}
	return rows
}

func (self *SNatGateway) AllowPerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "cancel-expire")
}

func (self *SNatGateway) PerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, self.CancelExpireTime(ctx, userCred)
}

func (self *SNatGateway) CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential) error {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return httperrors.NewBadRequestError("nat billing type %s not support cancel expire", self.BillingType)
	}

	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set expired_at = NULL and billing_cycle = NULL where id = ?",
			NatGatewayManager.TableSpec().Name(),
		), self.Id,
	)
	if err != nil {
		return errors.Wrap(err, "nat cancel expire time")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, "nat cancel expire time", userCred)
	return nil
}

func (self *SNatGateway) AllowPerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "postpaid-expire")
}

func (self *SNatGateway) PerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PostpaidExpireInput) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("nat gateway billing type is %s", self.BillingType)
	}

	bc, err := ParseBillingCycleInput(&self.SBillingResourceBase, input)
	if err != nil {
		return nil, err
	}

	err = self.SaveRenewInfo(ctx, userCred, bc, nil, billing_api.BILLING_TYPE_POSTPAID)
	return nil, err
}

func (self *SNatGateway) SaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	_, err := db.Update(self, func() error {
		if billingType == "" {
			billingType = billing_api.BILLING_TYPE_PREPAID
		}
		if self.BillingType == "" {
			self.BillingType = billingType
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SNatGateway) AllowPerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "renew")
}

func (self *SNatGateway) PerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.RenewInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.NAT_STAUTS_AVAILABLE}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do renew nat gateway in status %s required status %s", self.Status, api.NAT_SKU_AVAILABLE)
	}

	if len(input.Duration) == 0 {
		return nil, httperrors.NewMissingParameterError("duration")
	}

	bc, err := billing.ParseBillingCycle(input.Duration)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", input.Duration, err)
	}

	if !self.GetRegion().GetDriver().IsSupportedBillingCycle(bc, NatGatewayManager.KeywordPlural()) {
		return nil, httperrors.NewInputParameterError("unsupported duration %s", input.Duration)
	}

	return nil, self.StartRenewTask(ctx, userCred, input.Duration, "")
}

func (self *SNatGateway) StartRenewTask(ctx context.Context, userCred mcclient.TokenCredential, duration string, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("duration", jsonutils.NewString(duration))
	task, err := taskman.TaskManager.NewTask(ctx, "NatGatewayRenewTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.NAT_STATUS_RENEWING, "")
	return task.ScheduleRun(nil)
}

func (self *SNatGateway) AllowPerformSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "set-auto-renew")
}

func (self *SNatGateway) SetAutoRenew(autoRenew bool) error {
	_, err := db.Update(self, func() error {
		self.AutoRenew = autoRenew
		return nil
	})
	return err
}

// 设置自动续费
// 要求NAT状态为available
// 要求NAT计费类型为包年包月(预付费)
func (self *SNatGateway) PerformSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.AutoRenewInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.NAT_STAUTS_AVAILABLE}) {
		return nil, httperrors.NewUnsupportOperationError("The nat gateway status need be %s, current is %s", api.NAT_STAUTS_AVAILABLE, self.Status)
	}

	if self.BillingType != billing_api.BILLING_TYPE_PREPAID {
		return nil, httperrors.NewUnsupportOperationError("Only %s nat gateway support this operation", billing_api.BILLING_TYPE_PREPAID)
	}

	if self.AutoRenew == input.AutoRenew {
		return nil, nil
	}

	region := self.GetRegion()
	if region == nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("filed to get nat %s region", self.Name))
	}

	driver := region.GetDriver()
	if !driver.IsSupportedNatAutoRenew() {
		err := self.SetAutoRenew(input.AutoRenew)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		logclient.AddSimpleActionLog(self, logclient.ACT_SET_AUTO_RENEW, input, userCred, true)
		return nil, nil
	}

	return nil, self.StartSetAutoRenewTask(ctx, userCred, input.AutoRenew, "")
}

func (self *SNatGateway) StartSetAutoRenewTask(ctx context.Context, userCred mcclient.TokenCredential, autoRenew bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("auto_renew", jsonutils.NewBool(autoRenew))
	task, err := taskman.TaskManager.NewTask(ctx, "NatGatewaySetAutoRenewTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.NAT_STATUS_SET_AUTO_RENEW, "")
	return task.ScheduleRun(nil)
}

func (self *SNatEntry) GetINatGateway() (cloudprovider.ICloudNatGateway, error) {
	model, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch NatGateway whose id is %s failed", self.NatgatewayId)
	}
	natgateway := model.(*SNatGateway)
	return natgateway.GetINatGateway()
}

func (self *SNatEntry) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("NAT Entry delete do nothing")
	self.SetStatus(userCred, api.NAT_STATUS_DELETING, "")
	return nil
}

func (self *SNatEntry) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := db.DeleteModel(ctx, userCred, self)
	if err != nil {
		return err
	}
	self.SetStatus(userCred, api.NAT_STATUS_DELETED, "real delete")
	return nil
}

func (manager *SNatGatewayManager) getExpiredPostpaids() ([]SNatGateway, error) {
	q := ListExpiredPostpaidResources(manager.Query(), options.Options.ExpiredPrepaidMaxCleanBatchSize)

	nats := make([]SNatGateway, 0)
	err := db.FetchModelObjects(manager, q, &nats)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return nats, nil
}

func (self *SNatGateway) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	iNat, err := self.GetINatGateway()
	if err != nil {
		return errors.Wrapf(err, "GetINatGateway")
	}
	return self.SyncWithCloudNatGateway(ctx, userCred, self.GetCloudprovider(), iNat)
}

func (manager *SNatGatewayManager) DeleteExpiredPostpaids(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	nats, err := manager.getExpiredPostpaids()
	if err != nil {
		log.Errorf("Nats getExpiredPostpaids error: %v", err)
		return
	}
	for i := 0; i < len(nats); i += 1 {
		if len(nats[i].ExternalId) > 0 {
			err := nats[i].doExternalSync(ctx, userCred)
			if err == nil && nats[i].IsValidPostPaid() {
				continue
			}
		}
		nats[i].DeletePreventionOff(&nats[i], userCred)
		nats[i].StartNatGatewayDeleteTask(ctx, userCred, nil)
	}
}
