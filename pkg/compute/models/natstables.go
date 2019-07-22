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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNatSTableManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NatSTableManager *SNatSTableManager

func init() {
	NatSTableManager = &SNatSTableManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNatSTable{},
			"natstables_tbl",
			"natstable",
			"natstables",
		),
	}
	NatSTableManager.SetVirtualObject(NatSTableManager)
}

type SNatSTable struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	IP         string `width:"17" charset:"ascii" list:"user" create:"required"`
	SourceCIDR string `width:"22" charset:"ascii" list:"user" create:"required"`

	NetworkId    string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SNatSTableManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{NatGatewayManager},
	}
}

func (self *SNatSTableManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNatSTableManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNatSTable) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNatSTable) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNatSTable) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNatSTable) GetNatgateway() (*SNatGateway, error) {
	_natgateway, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _natgateway.(*SNatGateway), nil
}

func (self *SNatSTable) GetNetwork() (*SNetwork, error) {
	_network, err := NetworkManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _network.(*SNetwork), nil
}

func (man *SNatSTableManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "network", ModelKeyword: "network", OwnerId: userCred},
		{Key: "natgateway", ModelKeyword: "natgateway", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}

	q, err = managedResourceFilterByAccount(q, query, "natgateway_id", func() *sqlchemy.SQuery {
		natgateways := NatGatewayManager.Query().SubQuery()
		return natgateways.Query(natgateways.Field("id"))
	})

	return q, nil
}

func (man *SNatSTableManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SNatSTableManager) SyncNatSTables(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, nat *SNatGateway, extTables []cloudprovider.ICloudSNatEntry) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	result := compare.SyncResult{}
	dbNatSTables, err := nat.GetSTables()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatSTable, 0)
	commondb := make([]SNatSTable, 0)
	commonext := make([]cloudprovider.ICloudSNatEntry, 0)
	added := make([]cloudprovider.ICloudSNatEntry, 0)
	if err := compare.CompareSets(dbNatSTables, extTables, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudNatSTable(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudNatSTable(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		routeTableNew, err := manager.newFromCloudNatSTable(ctx, userCred, syncOwnerId, nat, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, routeTableNew, added[i])
		result.Add()
	}
	return result
}

func (self *SNatSTable) syncRemoveCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SNatSTable) SyncWithCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, extTable cloudprovider.ICloudSNatEntry) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extTable.GetStatus()
		self.IP = extTable.GetIP()
		self.SourceCIDR = extTable.GetSourceCIDR()
		if extNetworkId := extTable.GetNetworkId(); len(extNetworkId) > 0 {
			network, err := db.FetchByExternalId(NetworkManager, extNetworkId)
			if err != nil {
				return err
			}
			self.NetworkId = network.GetId()
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatSTableManager) newFromCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, nat *SNatGateway, extTable cloudprovider.ICloudSNatEntry) (*SNatSTable, error) {
	table := SNatSTable{}
	table.SetModelManager(manager, &table)

	newName, err := db.GenerateName(manager, ownerId, extTable.GetName())
	if err != nil {
		return nil, err
	}
	table.Name = newName
	table.Status = extTable.GetStatus()
	table.ExternalId = extTable.GetGlobalId()
	table.IsEmulated = extTable.IsEmulated()
	table.NatgatewayId = nat.Id

	table.IP = extTable.GetIP()
	table.SourceCIDR = extTable.GetSourceCIDR()
	if extNetworkId := extTable.GetNetworkId(); len(extNetworkId) > 0 {
		network, err := db.FetchByExternalId(NetworkManager, extNetworkId)
		if err != nil {
			return nil, err
		}
		table.NetworkId = network.GetId()
	}

	err = manager.TableSpec().Insert(&table)
	if err != nil {
		log.Errorf("newFromCloudNatSTable fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&table, db.ACT_CREATE, table.GetShortDesc(ctx), userCred)

	return &table, nil
}

func (self *SNatSTable) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return extra, nil
}

func (self *SNatSTable) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	natgateway, err := self.GetNatgateway()
	if err != nil {
		log.Errorf("failed to get naggateway %s for stable %s(%s) error: %v", self.NatgatewayId, self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.NewString(natgateway.Name), "natgateway")
	network, err := self.GetNetwork()
	if err != nil {
		log.Errorf("failed to get network %s for stable %s(%s) error: %v", self.NetworkId, self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.NewString(network.Name), "network")
	return extra
}
