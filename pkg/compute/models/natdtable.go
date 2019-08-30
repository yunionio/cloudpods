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
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNatDEntryManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NatDEntryManager *SNatDEntryManager

func init() {
	NatDEntryManager = &SNatDEntryManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNatDEntry{},
			"natdtables_tbl",
			"natdentry",
			"natdentries",
		),
	}
	NatDEntryManager.SetVirtualObject(NatDEntryManager)
}

type SNatDEntry struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	ExternalIP   string `width:"17" charset:"ascii" list:"user" create:"required"`
	ExternalPort int    `list:"user" create:"required"`

	InternalIP   string `width:"17" charset:"ascii" list:"user" create:"required"`
	InternalPort int    `list:"user" create:"required"`
	IpProtocol   string `width:"8" charset:"ascii" list:"user" create:"required"`

	NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SNatDEntryManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{NatGatewayManager},
	}
}

func (self *SNatDEntryManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNatDEntryManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNatDEntry) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNatDEntry) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNatDEntry) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNatDEntry) GetNatgateway() (*SNatGateway, error) {
	_natgateway, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, err
	}
	return _natgateway.(*SNatGateway), nil
}

func (man *SNatDEntryManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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

func (man *SNatDEntryManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := &api.SNatDCreateInput{}
	err := data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input failed %s", err)
	}
	if len(input.NatgatewayId) == 0 || len(input.ExternalIpId) == 0 || len(input.ExternalIp) == 0 {
		return nil, httperrors.NewMissingParameterError("natgateway_id or external_ip_id or external_ip")
	}
	if input.ExternalPort < 1 || input.ExternalPort > 65535 {
		return nil, httperrors.NewInputParameterError("Port value error")
	}
	if input.InternalPort < 1 || input.InternalPort > 65535 {
		return nil, httperrors.NewInputParameterError("Port value error")
	}
	if !regutils.MatchIPAddr(input.InternalIp) {
		return nil, httperrors.NewInputParameterError("invalid internal ip address: %s", input.InternalIp)
	}

	model, err := ElasticipManager.FetchById(input.ExternalIpId)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, httperrors.NewInputParameterError("No such eip")
	}
	eip := model.(*SElasticip)
	if eip.IpAddr != input.ExternalIp {
		return nil, errors.Error("No such eip")
	}

	// check that eip is suitable
	if len(eip.AssociateId) != 0 {
		if eip.AssociateId != input.NatgatewayId {
			return nil, httperrors.NewInputParameterError("eip has been binding to another instance")
		} else if !man.canBindIP(eip.IpAddr) {
			return nil, httperrors.NewInputParameterError("eip has been binding to dnat rules")
		}
	} else {
		data.Add(jsonutils.NewBool(true), "need_bind")
	}
	data.Remove("external_ip_id")
	data.Add(jsonutils.NewString(eip.ExternalId), "external_ip_id")
	return data, nil
}

func (manager *SNatDEntryManager) SyncNatDTable(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, nat *SNatGateway, extDTable []cloudprovider.ICloudNatDEntry) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	result := compare.SyncResult{}
	dbNatDTables, err := nat.GetDTable()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatDEntry, 0)
	commondb := make([]SNatDEntry, 0)
	commonext := make([]cloudprovider.ICloudNatDEntry, 0)
	added := make([]cloudprovider.ICloudNatDEntry, 0)
	if err := compare.CompareSets(dbNatDTables, extDTable, &removed, &commondb, &commonext, &added); err != nil {
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

func (self *SNatDEntry) syncRemoveCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SNatDEntry) SyncWithCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential, extEntry cloudprovider.ICloudNatDEntry) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extEntry.GetStatus()
		self.ExternalIP = extEntry.GetExternalIp()
		self.ExternalPort = extEntry.GetExternalPort()
		self.InternalIP = extEntry.GetInternalIp()
		self.InternalPort = extEntry.GetInternalPort()
		self.IpProtocol = extEntry.GetIpProtocol()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatDEntryManager) newFromCloudNatDTable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, nat *SNatGateway, extEntry cloudprovider.ICloudNatDEntry) (*SNatDEntry, error) {
	table := SNatDEntry{}
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
	table.ExternalIP = extEntry.GetExternalIp()
	table.ExternalPort = extEntry.GetExternalPort()
	table.InternalIP = extEntry.GetInternalIp()
	table.InternalPort = extEntry.GetInternalPort()
	table.IpProtocol = extEntry.GetIpProtocol()

	err = manager.TableSpec().Insert(&table)
	if err != nil {
		log.Errorf("newFromCloudNatDTable fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&table, db.ACT_CREATE, table.GetShortDesc(ctx), userCred)

	return &table, nil
}

func (self *SNatDEntry) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return extra, nil
}

func (self *SNatDEntry) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	natgateway, err := self.GetNatgateway()
	if err != nil {
		log.Errorf("failed to get naggateway %s for dtable %s(%s) error: %v", self.NatgatewayId, self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.NewString(natgateway.Name), "natgateway")
	return extra
}

func (self *SNatDEntry) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if len(self.NatgatewayId) == 0 {
		return
	}
	// ValidateCreateData function make data must contain 'externalIpId' key
	taskData := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "SNatDEntryCreateTask", self, userCred, taskData, "", "", nil)
	if err != nil {
		log.Errorf("SNatDEntryCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (self *SNatDEntry) GetINatGateway() (cloudprovider.ICloudNatGateway, error) {
	model, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch NatGateway whose id is %s failed", self.NatgatewayId)
	}
	natgateway := model.(*SNatGateway)
	return natgateway.GetINatGateway()
}

func (self *SNatDEntry) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("DNAT delete do nothing")
	self.SetStatus(userCred, api.NAT_STATUS_DELETING, "")
	return nil
}

func (self *SNatDEntry) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.StartDeleteVpcTask(ctx, userCred)
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SNatDEntry) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := db.DeleteModel(ctx, userCred, self)
	if err != nil {
		return err
	}
	self.SetStatus(userCred, api.NAT_STATUS_DELETED, "real delete")
	return nil
}

func (self *SNatDEntry) StartDeleteVpcTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SNatDEntryDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start dnatEntry deleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SNatDEntryManager) canBindIP(ipAddr string) bool {
	q := NatSEntryManager.Query().Equals("ip", ipAddr)
	count, _ := q.CountWithError()
	if count != 0 {
		return false
	}
	return true
}
