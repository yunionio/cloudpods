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

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SNetInterface struct {
	db.SResourceBase

	// Mac地址
	Mac string `width:"36" charset:"ascii" primary:"true"` // Column(VARCHAR(36, charset='ascii'), primary_key=True)
	// VLAN ID
	VlanId int `nullable:"false" default:"1" primary:"true"`

	BaremetalId string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	WireId      string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	Rate int `nullable:"true"` // Column(Integer, nullable=True) # Mbps

	NicType compute.TNicType `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	Index  int8  `nullable:"true"` // Column(TINYINT, nullable=True)
	LinkUp bool  `nullable:"true"` // Column(Boolean, nullable=True)
	Mtu    int16 `nullable:"true"` // Column(SMALLINT, nullable=True)

	// Bridge名称
	Bridge string `width:"64" charset:"ascii" nullable:"false"`
	// 接口名称
	Interface string `width:"16" charset:"ascii" nullable:"false"`
}

// +onecloud:swagger-gen-ignore
type SNetInterfaceManager struct {
	db.SResourceBaseManager
}

var NetInterfaceManager *SNetInterfaceManager

func init() {
	NetInterfaceManager = &SNetInterfaceManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SNetInterface{},
			"netinterfaces_tbl",
			"netinterface",
			"netinterfaces",
		),
	}
	NetInterfaceManager.SetVirtualObject(NetInterfaceManager)
}

func (netif SNetInterface) GetId() string {
	return fmt.Sprintf("%s-%d", netif.Mac, netif.VlanId)
}

func (manager *SNetInterfaceManager) FetchByMac(mac string) ([]SNetInterface, error) {
	q := manager.Query().Equals("mac", mac)
	ret := make([]SNetInterface, 0, 1)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func (manager *SNetInterfaceManager) FetchByMacVlan(mac string, vlanId int) (*SNetInterface, error) {
	q := manager.Query().Equals("mac", mac).Equals("vlan_id", vlanId)
	ret := &SNetInterface{}
	err := q.First(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "First")
	}
	ret.SetModelManager(manager, ret)
	return ret, nil
}

func (netif *SNetInterface) UnsetWire() error {
	_, err := db.Update(netif, func() error {
		netif.WireId = ""
		return nil
	})
	return err
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

func (netif *SNetInterface) GetHost() *SHost {
	if len(netif.BaremetalId) > 0 {
		hostModel, _ := HostManager.FetchById(netif.BaremetalId)
		if hostModel != nil {
			return hostModel.(*SHost)
		}
	}
	return nil
}

func (netif *SNetInterface) GetHostNetwork() *SHostnetwork {
	bn := SHostnetwork{}
	bn.SetModelManager(HostnetworkManager, &bn)

	q := HostnetworkManager.Query()
	q = q.Equals("baremetal_id", netif.BaremetalId).Equals("mac_addr", netif.Mac).Equals("vlan_id", netif.VlanId)
	err := q.First(&bn)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("fetch baremetal error: %s", err)
		}
		return nil
	}
	return &bn
}

/*func (netIf *SNetInterface) guestNetworkToJson(bn *SGuestnetwork) *jsonutils.JSONDict {
	jsonDesc := jsonutils.Marshal(netIf)
	desc := jsonDesc.(*jsonutils.JSONDict)

	if bn == nil {
		return desc
	} else {
		return netIf.networkToJson(bn.IpAddr, bn.GetNetwork(), desc)
	}
}

func (netIf *SNetInterface) hostNetworkToJson(bn *SHostnetwork) *jsonutils.JSONDict {
	jsonDesc := jsonutils.Marshal(netIf)
	desc := jsonDesc.(*jsonutils.JSONDict)

	if bn == nil {
		return desc
	} else {
		return netIf.networkToJson(bn.IpAddr, bn.GetNetwork(), desc)
	}
}*/

func (netIf *SNetInterface) networkToNic(ipAddr string, network *SNetwork, nic *types.SNic) *types.SNic {
	if len(ipAddr) > 0 {
		nic.IpAddr = ipAddr
	}

	if network != nil {
		if len(network.GuestGateway) > 0 && regutils.MatchIP4Addr(network.GuestGateway) {
			nic.Gateway = network.GuestGateway
		}
		nic.Dns = network.GetDNS("")
		nic.Domain = network.GetDomain()
		nic.Ntp = network.GetNTP()

		routes := network.GetRoutes()
		if len(routes) > 0 {
			nic.Routes = routes
		}

		nic.MaskLen = network.GuestIpMask
		nic.Net = network.Name
		nic.NetId = network.Id
	}

	return nic
}

func (netIf *SNetInterface) getServernetwork() *SGuestnetwork {
	host := netIf.GetHost()
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

	q := GuestnetworkManager.Query().Equals("guest_id", server.Id).Equals("mac_addr", netIf.Mac)

	err = q.First(obj)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("query fail %s", err)
		}
		return nil
	}
	return obj.(*SGuestnetwork)
}

func (netIf *SNetInterface) getServerJsonDesc() *api.GuestnetworkJsonDesc {
	desc := &api.GuestnetworkJsonDesc{}
	jsonutils.Update(desc, netIf)
	gn := netIf.getServernetwork()
	if gn != nil {
		jsonutils.Update(desc, gn.getJsonDescAtBaremetal(netIf.GetHost()))
		desc.Index = netIf.Index // override, preserve orginal network interface index
	}
	return desc
}

func (netif *SNetInterface) getBaremetalJsonDesc() *types.SNic {
	nic := &types.SNic{
		Mac:    netif.Mac,
		VlanId: netif.VlanId,

		Type:      netif.NicType,
		Rate:      netif.Rate,
		Mtu:       netif.Mtu,
		LinkUp:    netif.LinkUp,
		Interface: netif.Interface,
		Bridge:    netif.Bridge,
	}
	wire := netif.GetWire()
	if wire != nil {
		nic.WireId = wire.Id
		nic.Wire = wire.Name
		nic.Bandwidth = wire.Bandwidth
	}
	bn := netif.GetHostNetwork()
	if bn != nil {
		nic = netif.networkToNic(bn.IpAddr, bn.GetNetwork(), nic)
	}
	return nic
}

func (netif *SNetInterface) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.Update(netif, func() error {
		netif.WireId = ""
		netif.Bridge = ""
		netif.Interface = ""
		netif.BaremetalId = ""
		netif.Rate = 0
		netif.NicType = ""
		netif.Index = -1
		netif.LinkUp = false
		netif.Mtu = 0
		return nil
	})
	if err != nil {
		log.Errorf("Save Updates: %s", err)
		return errors.Wrap(err, "db.Update")
	}
	return netif.SResourceBase.Delete(ctx, userCred)
}

func (netIf *SNetInterface) GetCandidateNetworkForIp(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, ipAddr string) (*SNetwork, error) {
	wire := netIf.GetWire()
	if wire == nil {
		return nil, nil
	}
	log.Infof("ipAddr: %s, netiName: %s, wire: %s", ipAddr, netIf.GetName(), wire.GetName())
	return wire.GetCandidateNetworkForIp(userCred, ownerId, scope, ipAddr)
}

func (netif *SNetInterface) IsUsableServernic() bool {
	if netif.NicType == api.NIC_TYPE_IPMI {
		return false
	}
	if len(netif.WireId) == 0 {
		return false
	}
	if !netif.LinkUp {
		return false
	}
	if netif.getServernetwork() != nil {
		return false
	}
	return true
}

/*func (netif *SNetInterface) syncClassMetadata(ctx context.Context) error {
	host := netif.GetHost()
	wire := netif.GetWire()
	if host == nil {
		return errors.Wrap(errors.ErrInvalidStatus, "empty host")
	}
	if wire == nil {
		return errors.Wrap(errors.ErrInvalidStatus, "empty wire")
	}
	err := db.InheritFromTo(ctx, wire, host)
	if err != nil {
		log.Errorf("Inherit class metadata from host to wire fail: %s", err)
		return errors.Wrap(err, "InheritFromTo")
	}
	return nil
}*/

func (netif *SNetInterface) setNicType(tp compute.TNicType) error {
	_, err := db.Update(netif, func() error {
		netif.NicType = tp
		return nil
	})
	return errors.Wrap(err, "setNicType Update")
}
