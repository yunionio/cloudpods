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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SHostnetworkManager struct {
	SHostJointsManager
}

var HostnetworkManager *SHostnetworkManager

func init() {
	db.InitManager(func() {
		HostnetworkManager = &SHostnetworkManager{
			SHostJointsManager: NewHostJointsManager(
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

	BaremetalId string `width:"36" charset:"ascii" nullable:"false" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	NetworkId   string `width:"36" charset:"ascii" nullable:"false" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	IpAddr      string `width:"16" charset:"ascii" list:"admin"`                  // Column(VARCHAR(16, charset='ascii'))
	MacAddr     string `width:"18" charset:"ascii" list:"admin"`                  // Column(VARCHAR(18, charset='ascii'))
}

func (manager *SHostnetworkManager) GetMasterFieldName() string {
	return "baremetal_id"
}

func (manager *SHostnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (bn *SHostnetwork) Master() db.IStandaloneModel {
	return db.JointMaster(bn)
}

func (bn *SHostnetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(bn)
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
			rows[i].Baremetal = name
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
