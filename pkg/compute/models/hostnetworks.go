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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (bn *SHostnetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := bn.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(bn, extra)
	netif := bn.GetNetInterface()
	if netif != nil {
		extra.Add(jsonutils.NewString(netif.NicType), "nic_type")
	}
	return extra
}

func (bn *SHostnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := bn.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return db.JointModelExtra(bn, extra), nil
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
