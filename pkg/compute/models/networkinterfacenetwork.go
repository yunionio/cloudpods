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
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNetworkinterfacenetworkManager struct {
	db.SJointResourceBaseManager
}

var NetworkinterfacenetworkManager *SNetworkinterfacenetworkManager

func init() {
	db.InitManager(func() {
		NetworkinterfacenetworkManager = &SNetworkinterfacenetworkManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SNetworkinterfacenetwork{},
				"networkinterfacenetworks_tbl",
				"networkinterfacenetwork",
				"networkinterfacenetworks",
				NetworkInterfaceManager,
				NetworkManager,
			),
		}
		NetworkinterfacenetworkManager.SetVirtualObject(NetworkinterfacenetworkManager)
	})
}

type SNetworkinterfacenetwork struct {
	db.SJointResourceBase

	Primary            bool   `nullable:"false" list:"user"`
	IpAddr             string `width:"16" charset:"ascii" nullable:"false" list:"user"`
	NetworkinterfaceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	NetworkId          string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SNetworkinterfacenetworkManager) GetMasterFieldName() string {
	return "networkinterface_id"
}

func (manager *SNetworkinterfacenetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (manager *SNetworkinterfacenetworkManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master db.IStandaloneModel, slave db.IStandaloneModel) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SNetworkinterfacenetwork) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNetworkinterfacenetwork) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNetworkinterfacenetwork) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNetworkinterfacenetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SNetworkinterfacenetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SNetworkinterfacenetworkManager) SyncInterfaceAddresses(ctx context.Context, userCred mcclient.TokenCredential, networkinterface *SNetworkInterface, exts []cloudprovider.ICloudInterfaceAddress) compare.SyncResult {
	lockman.LockRawObject(ctx, "interface-addrs", networkinterface.Id)
	defer lockman.ReleaseRawObject(ctx, "interface-addrs", networkinterface.Id)

	syncResult := compare.SyncResult{}

	dbResources, err := networkinterface.GetNetworks()
	if err != nil {
		return syncResult
	}

	removed := make([]SNetworkinterfacenetwork, 0)
	commondb := make([]SNetworkinterfacenetwork, 0)
	commonext := make([]cloudprovider.ICloudInterfaceAddress, 0)
	added := make([]cloudprovider.ICloudInterfaceAddress, 0)
	if err := compare.CompareSets(dbResources, exts, &removed, &commondb, &commonext, &added); err != nil {
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudkInterfaceAddress(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := manager.newFromCloudInterfaceAddress(ctx, userCred, networkinterface, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (self *SNetworkinterfacenetwork) SyncWithCloudkInterfaceAddress(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInterfaceAddress) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Primary = ext.IsPrimary()

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNetworkinterfacenetworkManager) newFromCloudInterfaceAddress(ctx context.Context, userCred mcclient.TokenCredential, networkinterface *SNetworkInterface, ext cloudprovider.ICloudInterfaceAddress) error {
	address := SNetworkinterfacenetwork{
		IpAddr:             ext.GetIP(),
		NetworkinterfaceId: networkinterface.Id,
		Primary:            ext.IsPrimary(),
	}
	address.SetModelManager(manager, &address)

	networkId := ext.GetINetworkId()
	_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		wire := WireManager.Query().SubQuery()
		vpc := VpcManager.Query().SubQuery()
		return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
			Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
			Filter(sqlchemy.Equals(vpc.Field("manager_id"), networkinterface.ManagerId))
	})
	if err != nil {
		return errors.Wrapf(err, "newFromCloudInterfaceAddress.FetchByExternalId(%s)", networkId)
	}

	ipAddr, err := netutils.NewIPV4Addr(address.IpAddr)
	if err != nil {
		return errors.Wrap(err, "netutils.NewIPV4Addr")
	}

	network := _network.(*SNetwork)
	if !network.IsAddressInRange(ipAddr) {
		return fmt.Errorf("ip %s not in network %s(%s) range", address.IpAddr, network.Name, network.Id)
	}

	address.NetworkId = network.Id

	err = manager.TableSpec().Insert(ctx, &address)
	if err != nil {
		return errors.Wrap(err, "TableSpec().Insert(&address)")
	}

	db.OpsLog.LogEvent(&address, db.ACT_CREATE, address.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SNetworkinterfacenetwork) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return network.(*SNetwork), nil
}

func (self *SNetworkinterfacenetwork) GetDetailJson() (jsonutils.JSONObject, error) {
	network, err := self.GetNetwork()
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(map[string]interface{}{
		"network_id":          self.NetworkId,
		"ip_addr":             self.IpAddr,
		"primary":             self.Primary,
		"networkinterface_id": self.NetworkinterfaceId,
		"network":             network.Name,
	}), nil
}

func (manager *SNetworkinterfacenetworkManager) InitializeData() error {
	sq := NetworkInterfaceManager.Query("id")
	q := manager.Query().NotIn("networkinterface_id", sq.SubQuery())
	networks := []SNetworkinterfacenetwork{}
	err := db.FetchModelObjects(manager, q, &networks)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range networks {
		_, err = db.Update(&networks[i], func() error {
			return networks[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrapf(err, "Delete %d", networks[i].RowId)
		}
	}
	log.Debugf("SNetworkinterfacenetworkManager cleaned %d deprecated networkinterface ipAddrs.", len(networks))
	return nil
}
