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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNatSEntryManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NatSEntryManager *SNatSEntryManager

func init() {
	NatSEntryManager = &SNatSEntryManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNatSEntry{},
			"natstables_tbl",
			"natsentry",
			"natsentries",
		),
	}
	NatSEntryManager.SetVirtualObject(NatSEntryManager)
}

type SNatSEntry struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	IP         string `width:"17" charset:"ascii" list:"user" create:"required"`
	SourceCIDR string `width:"22" charset:"ascii" list:"user" create:"required"`

	NetworkId    string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SNatSEntryManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{NatGatewayManager},
	}
}

func (self *SNatSEntryManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNatSEntryManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNatSEntry) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNatSEntry) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNatSEntry) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNatSEntry) GetNatgateway() (*SNatGateway, error) {
	_natgateway, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _natgateway.(*SNatGateway), nil
}

func (self *SNatSEntry) GetNetwork() (*SNetwork, error) {
	_network, err := NetworkManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _network.(*SNetwork), nil
}

func (man *SNatSEntryManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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

func (man *SNatSEntryManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !data.Contains("external_ip_id") {
		return nil, errors.Error("Request body should contain key 'externalIpId'")
	}
	return data, nil
}

func (manager *SNatSEntryManager) SyncNatSTable(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, nat *SNatGateway, extTable []cloudprovider.ICloudNatSEntry) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	result := compare.SyncResult{}
	dbNatSTables, err := nat.GetSTable()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatSEntry, 0)
	commondb := make([]SNatSEntry, 0)
	commonext := make([]cloudprovider.ICloudNatSEntry, 0)
	added := make([]cloudprovider.ICloudNatSEntry, 0)
	if err := compare.CompareSets(dbNatSTables, extTable, &removed, &commondb, &commonext, &added); err != nil {
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

func (self *SNatSEntry) syncRemoveCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SNatSEntry) SyncWithCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, extEntry cloudprovider.ICloudNatSEntry) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extEntry.GetStatus()
		self.IP = extEntry.GetIP()
		self.SourceCIDR = extEntry.GetSourceCIDR()
		if extNetworkId := extEntry.GetNetworkId(); len(extNetworkId) > 0 {
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

func (manager *SNatSEntryManager) newFromCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, nat *SNatGateway, extEntry cloudprovider.ICloudNatSEntry) (*SNatSEntry, error) {
	table := SNatSEntry{}
	table.SetModelManager(manager, &table)

	newName, err := db.GenerateName(manager, ownerId, extEntry.GetName())
	if err != nil {
		return nil, err
	}
	table.Name = newName
	table.Status = extEntry.GetStatus()
	table.ExternalId = extEntry.GetGlobalId()
	table.IsEmulated = extEntry.IsEmulated()
	table.NatgatewayId = nat.Id

	table.IP = extEntry.GetIP()
	table.SourceCIDR = extEntry.GetSourceCIDR()
	if extNetworkId := extEntry.GetNetworkId(); len(extNetworkId) > 0 {
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

func (self *SNatSEntry) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return extra, nil
}

func (self *SNatSEntry) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
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

func (self *SNatSEntry) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if len(self.NatgatewayId) == 0 {
		return
	}
	// ValidateCreateData function make data must contain 'externalIpId' key
	externalIPID, _ := data.GetString("external_ip_id")
	taskData := jsonutils.NewDict()
	taskData.Set("external_ip_id", jsonutils.NewString(externalIPID))
	task, err := taskman.TaskManager.NewTask(ctx, "SNatSEntryCreateTask", self, userCred, taskData, "", "", nil)
	if err != nil {
		log.Errorf("SNatSEntryCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (self *SNatSEntry) GetINatGateway() (cloudprovider.ICloudNatGateway, error) {
	model, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch NatGateway whose id is %s failed", self.NatgatewayId)
	}
	natgateway := model.(*SNatGateway)
	return natgateway.GetINatGateway()
}

func (self *SNatSEntry) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.startDeleteVpcTask(ctx, userCred)
	} else {
		return self.realDelete(ctx, userCred)
	}
}

func (self *SNatSEntry) realDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	self.SetStatus(userCred, api.NAT_STATUS_DELETED, "real delete")
	return nil
}

func (self *SNatSEntry) startDeleteVpcTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SNatSEntryDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start snatEntry deleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
