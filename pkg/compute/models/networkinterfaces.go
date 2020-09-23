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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNetworkInterfaceManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var NetworkInterfaceManager *SNetworkInterfaceManager

func init() {
	NetworkInterfaceManager = &SNetworkInterfaceManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SNetworkInterface{},
			"networkinterfaces_tbl",
			"networkinterface",
			"networkinterfaces",
		),
	}
	NetworkInterfaceManager.SetVirtualObject(NetworkInterfaceManager)
}

type SNetworkInterface struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	// MAC地址
	Mac string `width:"36" charset:"ascii" list:"user"`
	// 绑定资源类型
	AssociateType string `width:"36" charset:"ascii" list:"user" nullable:"true" create:"optional"`
	// 绑定资源Id
	AssociateId string `width:"36" charset:"ascii" list:"user"`
}

func (manager *SNetworkInterfaceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

// 虚拟网卡列表
func (manager *SNetworkInterfaceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkInterfaceListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	if len(query.Mac) > 0 {
		q = q.In("mac", query.Mac)
	}
	if len(query.AssociateType) > 0 {
		q = q.In("associate_type", query.AssociateType)
	}
	if len(query.AssociateId) > 0 {
		q = q.In("associate_id", query.AssociateId)
	}

	return q, nil
}

func (manager *SNetworkInterfaceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkInterfaceListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SNetworkInterfaceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SNetworkInterface) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.NetworkInterfaceDetails, error) {
	return api.NetworkInterfaceDetails{}, nil
}

func (self *SNetworkInterface) getMoreDetails(out api.NetworkInterfaceDetails) (api.NetworkInterfaceDetails, error) {
	networks, err := self.GetNetworks()
	if err != nil {
		log.Errorf("failed to get network for networkinterface %s(%s) error: %v", self.Name, self.Id, err)
		return out, nil
	}
	out.Networks = []api.NetworkInterfaceNetworkInfo{}
	for _, network := range networks {
		_network, err := network.GetNetwork()
		if err != nil {
			return out, err
		}
		out.Networks = append(out.Networks, api.NetworkInterfaceNetworkInfo{
			NetworkId:          network.NetworkId,
			IpAddr:             network.IpAddr,
			Primary:            network.Primary,
			NetworkinterfaceId: network.NetworkinterfaceId,
			Network:            _network.Name,
		})
	}
	return out, nil
}

func (manager *SNetworkInterfaceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkInterfaceDetails {
	rows := make([]api.NetworkInterfaceDetails, len(objs))

	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.NetworkInterfaceDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regRows[i],
		}
		rows[i], _ = objs[i].(*SNetworkInterface).getMoreDetails(rows[i])
	}

	return rows
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
	lockman.LockRawObject(ctx, "network-interfaces", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "network-interfaces", fmt.Sprintf("%s-%s", provider.Id, region.Id))

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
		self.SetStatus(userCred, api.NETWORK_INTERFACE_STATUS_UNKNOWN, "sync to delete")
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}

	networks, err := self.GetNetworks()
	if err != nil {
		return errors.Wrapf(err, "GetNetworks")
	}
	for i := range networks {
		err = networks[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete networkinterfacenetwork %d", networks[i].RowId)
		}
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

	SyncCloudDomain(userCred, self, provider.GetOwnerId())
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SNetworkInterface) Associate(associateId string) error {
	switch self.AssociateType {
	case api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER:
		guest, err := db.FetchByExternalIdAndManagerId(GuestManager, associateId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := HostManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(q.Field("manager_id"), self.ManagerId))
		})
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

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return err
		}
		networkinterface.Name = newName

		return manager.TableSpec().Insert(ctx, &networkinterface)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	SyncCloudDomain(userCred, &networkinterface, provider.GetOwnerId())

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

func (manager *SNetworkInterfaceManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
