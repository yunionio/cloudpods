package models

import (
	"context"
	"database/sql"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/util/regutils"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
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
	NetInterfaceManager = &SNetInterfaceManager{SModelBaseManager: db.NewModelBaseManager(SNetInterface{}, "netinterfaces_tbl", "netinterface", "netinterfaces")}
}

func (netif *SNetInterface) GetId() string {
	return netif.Mac
}

func (manager *SNetInterfaceManager) FetchByMac(mac string) (*SNetInterface, error) {
	netif, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	err = manager.TableSpec().Query().Equals("mac", mac).First(&netif)
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

func (self *SNetInterface) toJson(ipAddr string, network *SNetwork) *jsonutils.JSONDict {
	jsonDesc := jsonutils.Marshal(self)
	desc := jsonDesc.(*jsonutils.JSONDict)

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

	wire := self.GetWire()
	if wire != nil {
		desc.Add(jsonutils.NewString(wire.Name), "wire")
	}
	return desc
}

func (self *SNetInterface) getServernetwork() *SGuestnetwork {
	host := self.GetBaremetal()
	if host != nil {
		return nil
	}
	server := host.getBaremetalServer()
	if server != nil {
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
		log.Errorf("query fail %s", err)
		return nil
	}
	return obj.(*SGuestnetwork)
}

func (self *SNetInterface) getServerJsonDesc() *jsonutils.JSONDict {
	bn := self.getServernetwork()
	desc := self.toJson(bn.IpAddr, bn.GetNetwork())
	if bn != nil {
		desc.Add(jsonutils.JSONFalse, "virtual")
		desc.Add(jsonutils.NewString(bn.IpAddr), "ip")
		desc.Add(jsonutils.NewInt(int64(bn.getBandwidth())), "bw")
		desc.Add(jsonutils.NewInt(int64(bn.Index)), "index")
		desc.Add(jsonutils.JSONTrue, "manual")
		vips := bn.GetVirtualIPs()
		if vips != nil && len(vips) > 0 {
			desc.Add(jsonutils.NewStringArray(vips), "virtual_ips")
		}
	}
	return desc
}

func (self *SNetInterface) getBaremetalJsonDesc() *jsonutils.JSONDict {
	bn := self.GetBaremetalNetwork()
	return self.toJson(bn.IpAddr, bn.GetNetwork())
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
		hw, err := HostwireManager.FetchByIds(host.Id, wire.Id)
		if err != nil {
			log.Errorf("NetInterface remove HostwireManager.FetchByIds error %s", err)
			return err
		}
		if hw != nil {
			hw.Delete(ctx, userCred)
		}
	}
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.WireId = ""
		self.BaremetalId = ""
		self.Rate = 0
		self.NicType = ""
		self.Index = 0
		self.LinkUp = false
		self.Mtu = 0
		return nil
	})
	if err != nil {
		log.Errorf("Save Updates: %s", err)
	}
	return nil
}
