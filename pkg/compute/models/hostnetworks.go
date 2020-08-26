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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SHostnetworkManager struct {
	SHostJointsManager
	SNetworkResourceBaseManager
}

var HostnetworkManager *SHostnetworkManager

func init() {
	db.InitManager(func() {
		HostnetworkManager = &SHostnetworkManager{
			SHostJointsManager: NewHostJointsManager(
				"baremetal_id",
				SHostnetwork{},
				"baremetalnetworks_tbl",
				"baremetalnetwork",
				"baremetalnetworks",
				NetworkManager,
			),
		}
		HostnetworkManager.SetVirtualObject(HostnetworkManager)
	})
}

type SHostnetwork struct {
	SHostJointsBase

	// 宿主机ID
	BaremetalId string `width:"36" charset:"ascii" nullable:"false" list:"domain"`
	// 网络ID
	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"domain"`
	// IP地址
	IpAddr string `width:"16" charset:"ascii" list:"domain"`
	// MAC地址
	MacAddr string `width:"18" charset:"ascii" list:"domain"`
}

func (manager *SHostnetworkManager) GetMasterFieldName() string {
	return manager.getHostIdFieldName()
}

func (manager *SHostnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (bn *SHostnetwork) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.HostnetworkDetails, error) {
	return api.HostnetworkDetails{}, nil
}

func (manager *SHostnetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostnetworkDetails {
	rows := make([]api.HostnetworkDetails, len(objs))

	hostRows := manager.SHostJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostIds := make([]string, len(rows))
	netIds := make([]string, len(rows))
	macIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.HostnetworkDetails{
			HostJointResourceDetails: hostRows[i],
		}
		hostIds[i] = objs[i].(*SHostnetwork).BaremetalId
		netIds[i] = objs[i].(*SHostnetwork).NetworkId
		macIds[i] = objs[i].(*SHostnetwork).MacAddr
	}

	hostIdMaps, err := db.FetchIdNameMap2(HostManager, hostIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 hostIds fail %s", err)
		return rows
	}
	netIdMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 netIds fail %s", err)
		return rows
	}
	netifs := make(map[string]SNetInterface)
	netifQ := NetInterfaceManager.Query("mac", "nic_type")
	err = db.FetchQueryObjectsByIds(netifQ, "mac", macIds, &netifs)
	if err != nil {
		log.Errorf("FetchQueryObjectsByIds macIds fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := hostIdMaps[hostIds[i]]; ok {
			rows[i].Host = name
		}
		if name, ok := netIdMaps[netIds[i]]; ok {
			rows[i].Network = name
		}
		if netif, ok := netifs[macIds[i]]; ok {
			rows[i].NicType = netif.NicType
		}
	}

	return rows
}

func (bn *SHostnetwork) GetHost() *SHost {
	master, _ := HostManager.FetchById(bn.BaremetalId)
	if master != nil {
		return master.(*SHost)
	}
	return nil
}

func (bn *SHostnetwork) GetNetwork() *SNetwork {
	slave, _ := NetworkManager.FetchById(bn.NetworkId)
	if slave != nil {
		return slave.(*SNetwork)
	}
	return nil
}

func (bn *SHostnetwork) GetNetInterface() *SNetInterface {
	netIf, _ := NetInterfaceManager.FetchByMac(bn.MacAddr)
	return netIf
}

func (bn *SHostnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, bn)
}

func (bn *SHostnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, bn)
}

func (man *SHostnetworkManager) QueryByAddress(addr string) *sqlchemy.SQuery {
	q := HostnetworkManager.Query()
	return q.Filter(sqlchemy.Equals(q.Field("ip_addr"), addr))
}

func (man *SHostnetworkManager) GetHostNetworkByAddress(addr string) *SHostnetwork {
	network := SHostnetwork{}
	err := man.QueryByAddress(addr).First(&network)
	if err == nil {
		return &network
	}
	return nil
}

func (man *SHostnetworkManager) GetNetworkByAddress(addr string) *SNetwork {
	net := man.GetHostNetworkByAddress(addr)
	if net == nil {
		return nil
	}
	return net.GetNetwork()
}

func (man *SHostnetworkManager) GetHostByAddress(addr string) *SHost {
	networks := man.TableSpec().Instance()
	hosts := HostManager.Query()
	q := hosts.Join(networks, sqlchemy.AND(
		sqlchemy.IsFalse(networks.Field("deleted")),
		sqlchemy.Equals(networks.Field("ip_addr"), addr),
		sqlchemy.Equals(networks.Field("baremetal_id"), hosts.Field("id")),
	))
	host := &SHost{}
	host.SetModelManager(HostManager, host)
	err := q.First(host)
	if err == nil {
		return host
	}
	return nil
}

func (manager *SHostnetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemFilter(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}
	if len(query.MacAddr) > 0 {
		q = q.In("mac_addr", query.MacAddr)
	}

	return q, nil
}

func (manager *SHostnetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.OrderByExtraFields(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SHostnetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SHostJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SNetworkResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
