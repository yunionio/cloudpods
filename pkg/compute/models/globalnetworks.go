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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGlobalNetworkManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var GlobalNetworkManager *SGlobalNetworkManager

func init() {
	GlobalNetworkManager = &SGlobalNetworkManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SGlobalNetwork{},
			"globalnetworks_tbl",
			"globalnetwork",
			"globalnetworks",
		),
	}
	GlobalNetworkManager.SetVirtualObject(GlobalNetworkManager)
}

type SGlobalNetwork struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	Provider string `width:"64" charset:"ascii" list:"user"`
}

func (manager *SGlobalNetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	//current not support create
	return false
}

func (self *SGlobalNetwork) ValidateDeleteCondition(ctx context.Context) error {
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SGlobalNetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
}

func (self *SGlobalNetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
}

func (manager *SGlobalNetworkManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (self *SGlobalNetwork) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SGlobalNetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SGlobalNetwork delete do nothing")
	self.SetStatus(userCred, api.NETWORK_STATUS_START_DELETE, "")
	return nil
}

func (self *SGlobalNetwork) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.StartDeleteGlobalNetworkTask(ctx, userCred)
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SGlobalNetwork) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	self.SetStatus(userCred, api.NETWORK_STATUS_DELETED, "real delete")
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SGlobalNetwork) StartDeleteGlobalNetworkTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GlobalNetworkDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SGlobalNetworkManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (self *SGlobalNetwork) ValidateUpdateCondition(ctx context.Context) error {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}

func (manager *SGlobalNetworkManager) GetGlobalNetworksByManagerId(id string) ([]SGlobalNetwork, error) {
	q := manager.Query().Equals("manager_id", id)
	q = q.Filter(sqlchemy.NOT(sqlchemy.IsNullOrEmpty(q.Field("external_id"))))
	globalnetworks := []SGlobalNetwork{}
	err := db.FetchModelObjects(manager, q, &globalnetworks)
	if err != nil {
		return nil, err
	}
	return globalnetworks, nil
}

func (self *SGlobalNetwork) GetGlobalNetworkVpcs() ([]SGlobalnetworkVpc, error) {
	gnvs := []SGlobalnetworkVpc{}
	q := GlobalnetworkVpcManager.Query().Equals("globalnetwork_id", self.Id)
	err := db.FetchModelObjects(GlobalnetworkVpcManager, q, &gnvs)
	if err != nil {
		return nil, err
	}
	return gnvs, nil
}

func (manager *SGlobalNetworkManager) SyncGlobalnetworks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudGlobalnetwork) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	result := compare.SyncResult{}

	dbNetworks, err := manager.GetGlobalNetworksByManagerId(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SGlobalNetwork, 0)
	commondb := make([]SGlobalNetwork, 0)
	commonext := make([]cloudprovider.ICloudGlobalnetwork, 0)
	added := make([]cloudprovider.ICloudGlobalnetwork, 0)
	err = compare.CompareSets(dbNetworks, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "CompareSets"))
		return result
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveGlobalnetwork(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		// update
		err = commondb[i].syncWithCloudGlobalnetwork(ctx, userCred, provider, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err := manager.newFromCloudGlobalnetwork(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SGlobalNetwork) syncWithCloudGlobalnetwork(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudGlobalnetwork) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "UpdateWithLock")
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SGlobalNetwork) syncRemoveGlobalnetwork(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return self.RealDelete(ctx, userCred)
}

func (manager *SGlobalNetworkManager) newFromCloudGlobalnetwork(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudGlobalnetwork) error {
	network := SGlobalNetwork{}
	network.SetModelManager(manager, &network)

	newName, err := db.GenerateName(manager, nil, ext.GetName())
	if err != nil {
		return errors.Wrap(err, "GenerateName")
	}
	network.ExternalId = ext.GetGlobalId()
	network.Name = newName
	network.Status = ext.GetStatus()
	network.Enabled = true
	network.ManagerId = provider.Id
	network.Provider = provider.Provider

	err = manager.TableSpec().Insert(&network)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}
	db.OpsLog.LogEvent(&network, db.ACT_CREATE, network.GetShortDesc(ctx), userCred)
	return nil
}
