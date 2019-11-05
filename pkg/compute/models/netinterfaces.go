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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNetInterface struct {
	db.SModelBase

	Mac         string `width:"36" charset:"ascii" primary:"true"`  // Column(VARCHAR(36, charset='ascii'), primary_key=True)
	BaremetalId string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	WireId      string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Rate        int    `nullable:"true"`                            // Column(Integer, nullable=True) # Mbps
	NicType     string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Index       int8   `nullable:"true"`                            // Column(TINYINT, nullable=True)
	LinkUp      bool   `nullable:"true"`                            // Column(Boolean, nullable=True)
	Mtu         int16  `nullable:"true"`                            // Column(SMALLINT, nullable=True)
}

type SNetInterfaceManager struct {
	db.SModelBaseManager
}

var NetInterfaceManager *SNetInterfaceManager

func init() {
	NetInterfaceManager = &SNetInterfaceManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNetInterface{},
			"netinterfaces_tbl",
			"netinterface",
			"netinterfaces",
		),
	}
	NetInterfaceManager.SetVirtualObject(NetInterfaceManager)
}

func (netif *SNetInterface) GetId() string {
	return netif.Mac
}

func (manager *SNetInterfaceManager) FetchByMac(mac string) (*SNetInterface, error) {
	netif, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	err = manager.Query().Equals("mac", mac).First(netif)
	if err != nil {
		return nil, err
	}
	return netif.(*SNetInterface), nil
}

func (netif *SNetInterface) GetWire() *SWire {
	if len(netif.WireId) > 0 {
		wireModel, _ := WireManager.FetchById(netif.WireId)
		if wireModel != nil {
			return wireModel.(*SWire)
		}
	}
	return nil
}

func (netif *SNetInterface) GetBaremetal() *SHost {
	if len(netif.BaremetalId) > 0 {
		hostModel, _ := HostManager.FetchById(netif.BaremetalId)
		if hostModel != nil {
			return hostModel.(*SHost)
		}
	}
	return nil
}

func (netif *SNetInterface) GetBaremetalNetwork() *SHostnetwork {
	bn := SHostnetwork{}
	bn.SetModelManager(HostnetworkManager, &bn)

	q := HostnetworkManager.Query()
	q = q.Equals("baremetal_id", netif.BaremetalId).Equals("mac_addr", netif.Mac)
	err := q.First(&bn)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("fetch baremetal error: %s", err)
		}
		return nil
	}
	return &bn
}

/*func (self *SNetInterface) guestNetworkToJson(bn *SGuestnetwork) *jsonutils.JSONDict {
	jsonDesc := jsonutils.Marshal(self)
	desc := jsonDesc.(*jsonutils.JSONDict)

	if bn == nil {
		return desc
	} else {
		return self.networkToJson(bn.IpAddr, bn.GetNetwork(), desc)
	}
}

func (self *SNetInterface) hostNetworkToJson(bn *SHostnetwork) *jsonutils.JSONDict {
	jsonDesc := jsonutils.Marshal(self)
	desc := jsonDesc.(*jsonutils.JSONDict)

	if bn == nil {
		return desc
	} else {
		return self.networkToJson(bn.IpAddr, bn.GetNetwork(), desc)
	}
}*/

func (self *SNetInterface) networkToJson(ipAddr string, network *SNetwork, desc *jsonutils.JSONDict) *jsonutils.JSONDict {
	if len(ipAddr) > 0 {
		desc.Add(jsonutils.NewString(ipAddr), "ip_addr")
	}

	if network != nil {
		if len(network.GuestGateway) > 0 && regutils.MatchIP4Addr(network.GuestGateway) {
			desc.Add(jsonutils.NewString(network.GuestGateway), "gateway")
		}
		desc.Add(jsonutils.NewString(network.GetDNS()), "dns")
		desc.Add(jsonutils.NewString(network.GetDomain()), "domain")

		routes := network.GetRoutes()
		if routes != nil && len(routes) > 0 {
			desc.Add(jsonutils.Marshal(routes), "routes")
		}
		desc.Add(jsonutils.NewInt(int64(network.GuestIpMask)), "masklen")
		desc.Add(jsonutils.NewString(network.Name), "net")
		desc.Add(jsonutils.NewString(network.Id), "net_id")
	}

	return desc
}

func (self *SNetInterface) getServernetwork() *SGuestnetwork {
	host := self.GetBaremetal()
	if host == nil {
		return nil
	}
	server := host.GetBaremetalServer()
	if server == nil {
		return nil
	}
	obj, err := db.NewModelObject(GuestnetworkManager)
	if err != nil {
		log.Errorf("new fail %s", err)
		return nil
	}

	q := GuestnetworkManager.Query().Equals("guest_id", server.Id).Equals("mac_addr", self.Mac)

	err = q.First(obj)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("query fail %s", err)
		}
		return nil
	}
	return obj.(*SGuestnetwork)
}

func (self *SNetInterface) getServerJsonDesc() *jsonutils.JSONDict {
	desc := jsonutils.Marshal(self).(*jsonutils.JSONDict)
	gn := self.getServernetwork()
	if gn != nil {
		desc.Update(gn.getJsonDescAtBaremetal(self.GetBaremetal()))
		desc.Set("index", jsonutils.NewInt(int64(self.Index))) // override, preserve orginal network interface index
	}
	return desc
}

func (self *SNetInterface) getBaremetalJsonDesc() *jsonutils.JSONDict {
	desc := jsonutils.Marshal(self).(*jsonutils.JSONDict)
	wire := self.GetWire()
	if wire != nil {
		desc.Add(jsonutils.NewString(wire.Name), "wire")
	}
	bn := self.GetBaremetalNetwork()
	if bn != nil {
		return self.networkToJson(bn.IpAddr, bn.GetNetwork(), desc)
	} else {
		return desc
	}
}

/*



def is_usable_servernic(self):
from clouds.baremetal import nictypes
if self.nic_type == nictypes.NIC_TYPE_IPMI:
return False
if self.wire_id is None:
return False
if not self.link_up:
return False
if self.get_servernetwork() is not None:
return False
return True

def get_candidate_network_for_ip(self, user_cred, ipaddr):
wire = self.get_wire()
if wire is None:
return None
print 'Find wire for ip', wire, ipaddr
return wire.get_candidate_network_for_ip(user_cred, ipaddr)*/

func (self *SNetInterface) Remove(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := self.GetBaremetal()
	wire := self.GetWire()
	if host != nil && wire != nil {
		hw, err := HostwireManager.FetchByHostIdAndMac(host.Id, self.Mac)
		if err != nil {
			log.Errorf("NetInterface remove HostwireManager.FetchByIds error %s", err)
			return err
		}
		if hw.WireId != wire.Id {
			return fmt.Errorf("NetInterface not attached to this wire???")
		}
		err = hw.Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	_, err := db.Update(self, func() error {
		self.WireId = ""
		self.BaremetalId = ""
		self.Rate = 0
		self.NicType = ""
		self.Index = -1
		self.LinkUp = false
		self.Mtu = 0
		return nil
	})
	if err != nil {
		log.Errorf("Save Updates: %s", err)
	}
	return err
}

func (self *SNetInterface) GetCandidateNetworkForIp(userCred mcclient.TokenCredential, ipAddr string) (*SNetwork, error) {
	wire := self.GetWire()
	if wire == nil {
		return nil, nil
	}
	return wire.GetCandidateNetworkForIp(userCred, ipAddr)
}

func (self *SNetInterface) IsUsableServernic() bool {
	if self.NicType == api.NIC_TYPE_IPMI {
		return false
	}
	if len(self.WireId) == 0 {
		return false
	}
	if !self.LinkUp {
		return false
	}
	if self.getServernetwork() != nil {
		return false
	}
	return true
}
