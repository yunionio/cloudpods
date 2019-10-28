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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceNetworkManager struct {
	SDBInstanceJointsManager
}

var DBInstanceNetworkManager *SDBInstanceNetworkManager

func init() {
	db.InitManager(func() {
		DBInstanceNetworkManager = &SDBInstanceNetworkManager{
			SDBInstanceJointsManager: NewDBInstanceJointsManager(
				SDBInstanceNetwork{},
				"dbinstancenetworks_tbl",
				"dbinstancenetwork",
				"dbinstancenetworks",
				NetworkManager,
			),
		}
		DBInstanceNetworkManager.SetVirtualObject(DBInstanceNetworkManager)
		DBInstanceNetworkManager.TableSpec().AddIndex(true, "ip_addr", "dbinstance_id")
	})
}

type SDBInstanceNetwork struct {
	SDBInstanceJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	IpAddr    string `width:"16" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDBInstanceNetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (joint *SDBInstanceNetwork) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SDBInstanceNetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SDBInstanceNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SDBInstanceNetwork) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SDBInstanceNetwork) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return network.(*SNetwork), nil
}

type SDBInstanceNetworkRequestData struct {
	DBInstance *SDBInstance
	NetworkId  string
	reserved   bool                      // allocate from reserved
	Address    string                    // the address user intends to use
	strategy   api.IPAllocationDirection // allocate bottom up, top down, randomly
}

func (m *SDBInstanceNetworkManager) NewDBInstanceNetwork(ctx context.Context, userCred mcclient.TokenCredential, req *SDBInstanceNetworkRequestData) (*SDBInstanceNetwork, error) {
	networkMan := db.GetModelManager("network").(*SNetworkManager)
	if networkMan == nil {
		return nil, fmt.Errorf("failed getting network manager")
	}
	im, err := networkMan.FetchById(req.NetworkId)
	if err != nil {
		return nil, err
	}
	network := im.(*SNetwork)
	in := &SDBInstanceNetwork{
		NetworkId: network.Id,
	}
	in.DBInstanceId = req.DBInstance.Id
	in.SetModelManager(m, in)

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)
	usedMap := network.GetUsedAddresses()
	recentReclaimed := map[string]bool{}
	ipAddr, err := network.GetFreeIP(ctx, userCred,
		usedMap, recentReclaimed, req.Address, req.strategy, req.reserved)
	if err != nil {
		return nil, err
	}
	in.IpAddr = ipAddr
	err = m.TableSpec().Insert(in)
	if err != nil {
		// NOTE no need to free ipAddr as GetFreeIP has no side effect
		return nil, err
	}
	return in, nil
}

func (manager *SDBInstanceNetworkManager) SyncDBInstanceNetwork(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, network *cloudprovider.SDBInstanceNetwork) compare.SyncResult {
	result := compare.SyncResult{}

	dbNetwork, err := dbinstance.GetDBNetwork()
	if err != nil && err != sql.ErrNoRows {
		result.Error(err)
		return result
	}

	if dbNetwork == nil {
		err = manager.newFromCloudDBNetwork(ctx, userCred, dbinstance, network)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	} else {
		err = dbNetwork.syncWithCloudDBNetwork(ctx, userCred, dbinstance, network)
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
		}
	}
	return result
}

func (self *SDBInstanceNetwork) syncWithCloudDBNetwork(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, network *cloudprovider.SDBInstanceNetwork) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		_localnetwork, err := db.FetchByExternalId(NetworkManager, network.NetworkId)
		if err != nil {
			return errors.Wrapf(err, "FetchByExternalId")
		}
		localnetwork := _localnetwork.(*SNetwork)
		self.NetworkId = localnetwork.Id

		ipAdd, err := netutils.NewIPV4Addr(network.IP)
		if err != nil {
			return errors.Wrapf(err, "NewIPV4Addr")
		}
		if !localnetwork.IsAddressInRange(ipAdd) {
			return fmt.Errorf("IP %s not in network %s(%s) address range", network.IP, localnetwork.Name, localnetwork.Id)
		}
		self.IpAddr = network.IP

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "syncWithCloudDBNetwork.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceNetworkManager) newFromCloudDBNetwork(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, network *cloudprovider.SDBInstanceNetwork) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	dbNetwork := SDBInstanceNetwork{}
	dbNetwork.SetModelManager(manager, &dbNetwork)

	dbNetwork.DBInstanceId = dbinstance.Id
	_localnetwork, err := db.FetchByExternalId(NetworkManager, network.NetworkId)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBNetwork.FetchByExternalId")
	}

	localnetwork := _localnetwork.(*SNetwork)
	ipAdd, err := netutils.NewIPV4Addr(network.IP)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBNetwork.NewIPV4Addr")
	}
	if !localnetwork.IsAddressInRange(ipAdd) {
		return fmt.Errorf("IP %s not in network %s(%s) address range", network.IP, localnetwork.Name, localnetwork.Id)
	}
	dbNetwork.NetworkId = localnetwork.Id
	dbNetwork.IpAddr = network.IP

	err = manager.TableSpec().Insert(&dbNetwork)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBNetwork.Insert")
	}
	return nil
}
