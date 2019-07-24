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

	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNatGetewayManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NatGatewayManager *SNatGetewayManager

func init() {
	NatGatewayManager = &SNatGetewayManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNatGateway{},
			"natgateways_tbl",
			"natgateway",
			"natgateways",
		),
	}
	NatGatewayManager.SetVirtualObject(NatGatewayManager)
}

type SNatGateway struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase
	SBillingResourceBase

	VpcId   string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NatSpec string `list:"user" get:"user" list:"user" create:"optional"` // NAT规格
}

func (manager *SNatGetewayManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SNatGetewayManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNatGetewayManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNatGateway) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNatGateway) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNatGateway) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (man *SNatGetewayManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "vpc", ModelKeyword: "vpc", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SNatGetewayManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (self *SNatGateway) GetVpc() (*SVpc, error) {
	_vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, err
	}
	return _vpc.(*SVpc), nil
}

func (manager *SNatGetewayManager) getNatgatewaysByProviderId(providerId string) ([]SNatGateway, error) {
	nats := []SNatGateway{}
	err := fetchByManagerId(manager, providerId, &nats)
	if err != nil {
		return nil, err
	}
	return nats, nil
}

func (self *SNatGateway) GetDTables() ([]SNatDTable, error) {
	tables := []SNatDTable{}
	q := NatDTableManager.Query().Equals("natgateway_id", self.Id)
	err := db.FetchModelObjects(NatDTableManager, q, &tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}

func (self *SNatGateway) GetSTables() ([]SNatSTable, error) {
	tables := []SNatSTable{}
	q := NatSTableManager.Query().Equals("natgateway_id", self.Id)
	err := db.FetchModelObjects(NatSTableManager, q, &tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}

func (self *SNatGateway) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return extra, nil
}

func (self *SNatGateway) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	accountInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if accountInfo != nil {
		extra.Update(accountInfo)
	}
	regionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	vpc, err := self.GetVpc()
	if err != nil {
		log.Errorf("failed to found vpc info for nat gateway %s(%s) error: %v", self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.NewString(vpc.Name), "vpc")
	return extra
}

func (manager *SNatGetewayManager) SyncNatGateways(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, vpc *SVpc, cloudNatGateways []cloudprovider.ICloudNatGateway) ([]SNatGateway, []cloudprovider.ICloudNatGateway, compare.SyncResult) {
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
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SNatGateway) SyncWithCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extNat cloudprovider.ICloudNatGateway) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extNat.GetStatus()
		self.NatSpec = extNat.GetNatSpec()

		factory, _ := provider.GetProviderFactory()
		if factory.IsSupportPrepaidResources() {
			self.BillingType = extNat.GetBillingType()
			self.ExpiredAt = extNat.GetExpiredAt()
		}

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatGetewayManager) newFromCloudNatGateway(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, vpc *SVpc, extNat cloudprovider.ICloudNatGateway) (*SNatGateway, error) {
	nat := SNatGateway{}
	nat.SetModelManager(manager, &nat)

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, err
	}

	newName, err := db.GenerateName(manager, ownerId, extNat.GetName())
	if err != nil {
		return nil, err
	}
	nat.Name = newName
	nat.VpcId = vpc.Id
	nat.Status = extNat.GetStatus()
	nat.NatSpec = extNat.GetNatSpec()
	if createdAt := extNat.GetCreatedAt(); !createdAt.IsZero() {
		nat.CreatedAt = extNat.GetCreatedAt()
	}
	nat.ExternalId = extNat.GetGlobalId()
	nat.CloudregionId = region.Id
	nat.ManagerId = provider.Id
	nat.IsEmulated = extNat.IsEmulated()

	factory, _ := provider.GetProviderFactory()
	if factory.IsSupportPrepaidResources() {
		nat.BillingType = extNat.GetBillingType()
		nat.ExpiredAt = extNat.GetExpiredAt()
	}

	err = manager.TableSpec().Insert(&nat)
	if err != nil {
		log.Errorf("newFromCloudNatGateway fail %s", err)
		return nil, err
	}

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
		} else {
			result.Add()
		}
	}

	return result
}
