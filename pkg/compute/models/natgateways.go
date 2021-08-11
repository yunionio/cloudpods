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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNatGatewayManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager
	// SManagedResourceBaseManager
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
	// SManagedResourceBase
	SBillingResourceBase
	SVpcResourceBase

	NatSpec string `list:"user" create:"optional"` // NAT规格
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

func (man *SNatGatewayManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
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
	for i := range rows {
		rows[i] = api.NatgatewayDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			VpcResourceInfo:                 vpcRows[i],
		}
		rows[i], _ = objs[i].(*SNatGateway).getMoreDetails(ctx, userCred, rows[i])
	}
	return rows
}

func (self *SNatGateway) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	out api.NatgatewayDetails) (api.NatgatewayDetails, error) {

	region, _ := self.GetRegion()
	out.NatSpec = region.GetDriver().DealNatGatewaySpec(self.NatSpec)

	return out, nil
}

func (manager *SNatGatewayManager) SyncNatGateways(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, vpc *SVpc, cloudNatGateways []cloudprovider.ICloudNatGateway) ([]SNatGateway, []cloudprovider.ICloudNatGateway, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

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

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.NAT_STATUS_UNKNOWN, "sync to delete")
	}
	return self.purge(ctx, userCred)
}

func (self *SNatGateway) SyncWithCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extNat cloudprovider.ICloudNatGateway) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extNat.GetStatus()
		self.NatSpec = extNat.GetNatSpec()

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

	/*region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "vpc.GetRegion")
	}*/

	newName, err := db.GenerateName(manager, ownerId, extNat.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "db.GenerateName")
	}
	nat.Name = newName
	nat.VpcId = vpc.Id
	nat.Status = extNat.GetStatus()
	nat.NatSpec = extNat.GetNatSpec()
	if createdAt := extNat.GetCreatedAt(); !createdAt.IsZero() {
		nat.CreatedAt = extNat.GetCreatedAt()
	}
	nat.ExternalId = extNat.GetGlobalId()
	// nat.CloudregionId = region.Id
	// nat.ManagerId = provider.Id
	nat.IsEmulated = extNat.IsEmulated()

	factory, _ := provider.GetProviderFactory()
	if factory.IsSupportPrepaidResources() {
		nat.BillingType = extNat.GetBillingType()
		if expired := extNat.GetExpiredAt(); !expired.IsZero() {
			nat.ExpiredAt = expired
		}
		nat.AutoRenew = extNat.IsAutoRenew()
	}

	err = manager.TableSpec().Insert(ctx, &nat)
	if err != nil {
		log.Errorf("newFromCloudNatGateway fail %s", err)
		return nil, errors.Wrap(err, "Insert")
	}

	SyncCloudDomain(userCred, &nat, provider.GetOwnerId())

	db.OpsLog.LogEvent(&nat, db.ACT_CREATE, nat.GetShortDesc(ctx), userCred)

	return &nat, nil
}

func (self *SNatGateway) GetEips() ([]SElasticip, error) {
	q := ElasticipManager.Query().Equals("associate_id", self.Id)
	eips := []SElasticip{}
	if err := db.FetchModelObjects(ElasticipManager, q, &eips); err != nil {
		return nil, err
	}
	return eips, nil
}

func (self *SNatGateway) SyncNatGatewayEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extEips []cloudprovider.ICloudEIP) compare.SyncResult {
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
			result.AddError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(added); i += 1 {
		region, _ := self.GetRegion()
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, added[i], provider, region, provider.GetOwnerId())
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
		} else {
			result.Add()
		}
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

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "NatGatewaySyncstatusTask", "")
}

func (self *SNatGateway) GetINatGateway() (cloudprovider.ICloudNatGateway, error) {
	model, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, errors.Wrap(err, "Fetch vpc by ID failed")
	}
	vpc := model.(*SVpc)
	cloudVpc, err := vpc.GetIVpc()
	if err != nil {
		return nil, errors.Wrap(err, "Fetch IVpc failed")
	}
	cloudNatGateways, err := cloudVpc.GetINatGateways()
	if err != nil {
		return nil, errors.Wrapf(err, "Get INatGateways of vpc %s failed", cloudVpc.GetGlobalId())
	}
	for i := range cloudNatGateways {
		if cloudNatGateways[i].GetGlobalId() == self.ExternalId {
			return cloudNatGateways[i], nil
		}
	}
	return nil, errors.Error("CloudNatGateway Not Found")
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
	return self.Delete(ctx, userCred)
}

func (nm *SNatGatewayManager) NatNameToReal(name string, natgatewayId string) string {
	index := strings.Index(name, natgatewayId)
	if index < 0 {
		return name
	}
	return name[:index-1]
}

func (nm *SNatGatewayManager) NatNameFromReal(name string, natgatewayId string) string {
	return fmt.Sprintf("%s-%s", name, natgatewayId)
}

type INatHelper interface {
	db.IModel
	CountByEIP() (int, error)
	GetNatgateway() (*SNatGateway, error)
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
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
	// NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
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
		var base *SNatEntry
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && base.NatgatewayId != "" {
			rows[i].RealName = NatGatewayManager.NatNameToReal(base.Name, base.NatgatewayId)
		}
	}
	return rows
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

func (self *SNatEntry) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	if data.Contains("name") {
		name, _ := data.GetString("name")
		natgateway, err := self.GetNatgateway()
		if err != nil {
			return nil, err
		}
		data.Set("name", jsonutils.NewString(NatGatewayManager.NatNameFromReal(name, natgateway.GetId())))
	}
	return nil, nil
}
