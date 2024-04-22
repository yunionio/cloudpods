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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SDBInstanceNetworkManager struct {
	SDBInstanceJointsManager
	SNetworkResourceBaseManager
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

// +onecloud:swagger-gen-ignore
type SDBInstanceNetwork struct {
	SDBInstanceJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	IpAddr    string `width:"16" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDBInstanceNetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (self *SDBInstanceNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

// RDS网络列表
func (manager *SDBInstanceNetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceNetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDBInstanceJointsManager.ListItemFilter(ctx, q, userCred, query.DBInstanceJoinListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceJointsManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (self *SDBInstanceNetwork) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return network.(*SNetwork), nil
}

func (manager *SDBInstanceNetworkManager) newNetwork(ctx context.Context, userCred mcclient.TokenCredential, rdsId, networkId, ipAddr string) error {
	ds := &SDBInstanceNetwork{}
	ds.SetModelManager(DBInstanceNetworkManager, ds)
	ds.DBInstanceId = rdsId
	ds.NetworkId = networkId
	ds.IpAddr = ipAddr
	return manager.TableSpec().Insert(ctx, ds)
}

func (manager *SDBInstanceNetworkManager) SyncDBInstanceNetwork(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, exts []cloudprovider.SDBInstanceNetwork) compare.SyncResult {
	result := compare.SyncResult{}

	networks, err := dbinstance.GetDBNetworks()
	if err != nil {
		result.Error(err)
		return result
	}

	localMap := map[string]SDBInstanceNetwork{}
	for i := range networks {
		localMap[networks[i].NetworkId+networks[i].IpAddr] = networks[i]
	}
	remoteMap := map[string]bool{}

	for i := range exts {
		_network, err := db.FetchByExternalIdAndManagerId(NetworkManager, exts[i].NetworkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), dbinstance.ManagerId))
		})
		if err != nil {
			result.Error(err)
			continue
		}
		network := _network.(*SNetwork)
		exts[i].NetworkId = network.GetId()
		remoteMap[exts[i].NetworkId+exts[i].IP] = true
		_, ok := localMap[exts[i].NetworkId+exts[i].IP]
		if !ok {
			ipAddr, err := netutils.NewIPV4Addr(exts[i].IP)
			if err != nil {
				result.AddError(errors.Wrapf(err, "invalid ip"))
			}

			if !network.IsAddressInRange(ipAddr) {
				result.AddError(fmt.Errorf("IP %s not in network %s(%s) address range", exts[i].IP, network.Name, network.Id))
				continue
			}

			err = manager.newNetwork(ctx, userCred, dbinstance.Id, exts[i].NetworkId, exts[i].IP)
			if err != nil {
				result.AddError(err)
				continue
			}
			result.Add()
		}
	}
	for i := range networks {
		_, ok := remoteMap[networks[i].NetworkId+networks[i].IpAddr]
		if ok {
			continue
		}
		err = networks[i].Detach(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	return result
}

func (manager *SDBInstanceNetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceNetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SDBInstanceJointsManager.OrderByExtraFields(ctx, q, userCred, query.DBInstanceJoinListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceJointsManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SDBInstanceNetworkManager) InitializeData() error {
	sq := DBInstanceManager.Query("id")
	q := manager.Query().NotIn("dbinstance_id", sq.SubQuery())
	networks := []SDBInstanceNetwork{}
	err := db.FetchModelObjects(manager, q, &networks)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range networks {
		_, err := db.Update(&networks[i], func() error {
			return networks[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update.MarkDelete")
		}
	}
	log.Debugf("SDBInstanceNetworkManager cleaned %d dirty data.", len(networks))
	return nil
}
