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

type SNatDTableManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NatDTableManager *SNatDTableManager

func init() {
	NatDTableManager = &SNatDTableManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNatDTable{},
			"natdtables_tbl",
			"natdtable",
			"natdtables",
		),
	}
}

type SNatDTable struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	ExternalIP   string `width:"17" charset:"ascii" list:"user" create:"required"`
	ExternalPort int    `list:"user" create:"required"`

	InternalIP   string `width:"17" charset:"ascii" list:"user" create:"required"`
	InternalPort int    `list:"user" create:"required"`
	IpProtocol   string `width:"8" charset:"ascii" list:"user" create:"required"`

	NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SNatDTableManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{NatGatewayManager},
	}
}

func (self *SNatDTableManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNatDTableManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNatDTable) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNatDTable) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNatDTable) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNatDTable) GetNatgateway() (*SNatGateway, error) {
	_natgateway, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _natgateway.(*SNatGateway), nil
}

func (man *SNatDTableManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
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

func (man *SNatDTableManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SNatDTableManager) SyncNatDTables(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, nat *SNatGateway, extDtables []cloudprovider.ICloudNatDTable) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	result := compare.SyncResult{}
	dbNatDTables, err := nat.GetDTables()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatDTable, 0)
	commondb := make([]SNatDTable, 0)
	commonext := make([]cloudprovider.ICloudNatDTable, 0)
	added := make([]cloudprovider.ICloudNatDTable, 0)
	if err := compare.CompareSets(dbNatDTables, extDtables, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudNatDTable(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudNatDTable(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		routeTableNew, err := manager.newFromCloudNatDTable(ctx, userCred, syncOwnerId, nat, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, routeTableNew, added[i])
		result.Add()
	}
	return result
}

func (self *SNatDTable) syncRemoveCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SNatDTable) SyncWithCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential, extTable cloudprovider.ICloudNatDTable) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extTable.GetStatus()
		self.ExternalIP = extTable.GetExternalIp()
		self.ExternalPort = extTable.GetExternalPort()
		self.InternalIP = extTable.GetInternalIp()
		self.InternalPort = extTable.GetInternalPort()
		self.IpProtocol = extTable.GetIpProtocol()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatDTableManager) newFromCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, nat *SNatGateway, extTable cloudprovider.ICloudNatDTable) (*SNatDTable, error) {
	table := SNatDTable{}
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
	table.ExternalIP = extTable.GetExternalIp()
	table.ExternalPort = extTable.GetExternalPort()
	table.InternalIP = extTable.GetInternalIp()
	table.InternalPort = extTable.GetInternalPort()
	table.IpProtocol = extTable.GetIpProtocol()

	err = manager.TableSpec().Insert(&table)
	if err != nil {
		log.Errorf("newFromCloudNatDTable fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&table, db.ACT_CREATE, table.GetShortDesc(ctx), userCred)

	return &table, nil
}

func (self *SNatDTable) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return extra, nil
}

func (self *SNatDTable) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	natgateway, err := self.GetNatgateway()
	if err != nil {
		log.Errorf("failed to get naggateway %s for dtable %s(%s) error: %v", self.NatgatewayId, self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.NewString(natgateway.Name), "natgateway")
	return extra
}
