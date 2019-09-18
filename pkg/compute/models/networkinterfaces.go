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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNetworkInterfaceManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var NetworkInterfaceManager *SNetworkInterfaceManager

func init() {
	NetworkInterfaceManager = &SNetworkInterfaceManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SNetworkInterface{},
			"networkinterfaces_tbl",
			"networkinterface",
			"networkinterfaces",
		),
	}
	NetworkInterfaceManager.SetVirtualObject(NetworkInterfaceManager)
}

type SNetworkInterface struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	Mac           string `width:"36" charset:"ascii" list:"user"`
	AssociateType string `width:"36" charset:"ascii" list:"user" nullable:"true" create:"optional"`
	AssociateId   string `width:"36" charset:"ascii" list:"user"`
}

func (manager *SNetworkInterfaceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SNetworkInterfaceManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SNetworkInterfaceManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SNetworkInterface) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNetworkInterface) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNetworkInterface) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SNetworkInterfaceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	return q, err
}

func (self *SNetworkInterface) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	accountInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if accountInfo != nil {
		extra.Update(accountInfo)
	}
	regionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	networks, err := self.GetNetworks()
	if err != nil {
		log.Errorf("failed to get network for networkinterface %s(%s) error: %v", self.Name, self.Id, err)
		return extra
	}
	extra.Add(jsonutils.Marshal(networks), "networks")
	return extra
}

func (manager *SNetworkInterfaceManager) getNetworkInterfacesByProviderId(providerId string) ([]SNetworkInterface, error) {
	nics := []SNetworkInterface{}
	err := fetchByManagerId(manager, providerId, &nics)
	if err != nil {
		return nil, err
	}
	return nics, nil
}

func (manager *SNetworkInterfaceManager) SyncNetworkInterfaces(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, exts []cloudprovider.ICloudNetworkInterface) ([]SNetworkInterface, []cloudprovider.ICloudNetworkInterface, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

	localResources := make([]SNetworkInterface, 0)
	remoteResources := make([]cloudprovider.ICloudNetworkInterface, 0)
	syncResult := compare.SyncResult{}

	dbResources, err := region.GetNetworkInterfaces()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SNetworkInterface, 0)
	commondb := make([]SNetworkInterface, 0)
	commonext := make([]cloudprovider.ICloudNetworkInterface, 0)
	added := make([]cloudprovider.ICloudNetworkInterface, 0)
	if err := compare.CompareSets(dbResources, exts, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudNetworkInterface(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudNetworkInterface(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localResources = append(localResources, commondb[i])
		remoteResources = append(remoteResources, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudNetworkInterface(ctx, userCred, provider, region, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, new, added[i])
		localResources = append(localResources, *new)
		remoteResources = append(remoteResources, added[i])
		syncResult.Add()
	}
	return localResources, remoteResources, syncResult
}

func (self *SNetworkInterface) syncRemoveCloudNetworkInterface(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return self.SetStatus(userCred, api.NETWORK_INTERFACE_STATUS_UNKNOWN, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SNetworkInterface) SyncWithCloudNetworkInterface(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudNetworkInterface) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		self.AssociateType = ext.GetAssociateType()
		if associateId := ext.GetAssociateId(); len(associateId) > 0 {
			self.Associate(associateId)
		}

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SNetworkInterface) Associate(associateId string) error {
	switch self.AssociateType {
	case api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER:
		guest, err := db.FetchByExternalId(GuestManager, associateId)
		if err != nil {
			return errors.Wrapf(err, "failed to get guest for networkinterface %s associateId %s", self.Name, associateId)
		}
		self.AssociateId = guest.GetId()
	}
	return nil
}

func (manager *SNetworkInterfaceManager) newFromCloudNetworkInterface(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, ext cloudprovider.ICloudNetworkInterface) (*SNetworkInterface, error) {
	networkinterface := SNetworkInterface{}
	networkinterface.SetModelManager(manager, &networkinterface)

	newName, err := db.GenerateName(manager, provider.GetOwnerId(), ext.GetName())
	if err != nil {
		return nil, err
	}
	networkinterface.Name = newName

	networkinterface.Status = ext.GetStatus()
	networkinterface.ExternalId = ext.GetGlobalId()
	networkinterface.CloudregionId = region.Id
	networkinterface.ManagerId = provider.Id
	networkinterface.IsEmulated = ext.IsEmulated()
	networkinterface.Mac = ext.GetMacAddress()
	networkinterface.AssociateType = ext.GetAssociateType()

	if associatId := ext.GetAssociateId(); len(associatId) > 0 {
		err := networkinterface.Associate(associatId)
		if err != nil {
			log.Warningf("associate error: %s", err)
		}
	}

	err = manager.TableSpec().Insert(&networkinterface)
	if err != nil {
		return nil, errors.Wrap(err, "TableSpec().Insert(&networkinterface)")
	}

	db.OpsLog.LogEvent(&networkinterface, db.ACT_CREATE, networkinterface.GetShortDesc(ctx), userCred)

	return &networkinterface, nil
}

func (self *SNetworkInterface) GetNetworks() ([]SNetworkinterfacenetwork, error) {
	networks := []SNetworkinterfacenetwork{}
	q := NetworkinterfacenetworkManager.Query().Equals("networkinterface_id", self.Id)
	err := db.FetchModelObjects(NetworkinterfacenetworkManager, q, &networks)
	if err != nil {
		return nil, err
	}
	return networks, nil
}
