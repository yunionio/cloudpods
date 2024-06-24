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
	"sort"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type CASimpleNetConf struct {
	IpStart string `json:"guest_ip_start"`
	IpEnd   string `json:"guest_ip_end"`
	IpMask  int8   `json:"guest_ip_mask"`
	Gateway string `json:"guest_gateway"`
	VlanID  int32  `json:"vlan_id"`

	WireId string `json:"wire_id"`
}

type CANetConf struct {
	CASimpleNetConf

	Name        string `json:"name"`
	Description string `json:"description"`
}

var ipMaskLen int8 = 24

func (account *SCloudaccount) PrepareEsxiHostNetwork(ctx context.Context, userCred mcclient.TokenCredential, zoneId string) error {
	cProvider, err := account.GetProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "account.GetProvider")
	}
	// fetch esxiclient
	iregion, err := cProvider.GetOnPremiseIRegion()
	if err != nil {
		return errors.Wrap(err, "cProvider.GetOnPremiseIRegion")
	}
	esxiClient, ok := iregion.(*esxi.SESXiClient)
	if !ok {
		return errors.Wrap(httperrors.ErrNotSupported, "not a esxi provider")
	}
	iHosts, err := esxiClient.GetIHosts()
	if err != nil {
		return errors.Wrap(err, "esxiClient.GetIHosts")
	}

	hostIps := make([]netutils.IPV4Addr, 0)
	for i := range iHosts {
		accessIp := iHosts[i].GetAccessIp()
		hostNics, err := iHosts[i].GetIHostNics()
		if err != nil {
			return errors.Wrapf(err, "iHosts[%d].GetIHostNics()", i)
		}
		findAccessIp := false
		for _, hn := range hostNics {
			if len(hn.GetBridge()) > 0 {
				// a bridged nic, must be a virtual port group, skip
				continue
			}
			ipAddrStr := hn.GetIpAddr()
			if len(ipAddrStr) == 0 {
				// skip interface without a valid ip address
				continue
			}
			if accessIp == ipAddrStr {
				findAccessIp = true
			}
			ipAddr, err := netutils.NewIPV4Addr(ipAddrStr)
			if err != nil {
				log.Errorf("fail to parse ipv4 addr %s: %s", ipAddrStr, err)
			} else {
				hostIps = append(hostIps, ipAddr)
			}
		}
		if !findAccessIp {
			log.Errorf("Fail to find access ip %s NIC for esxi host %s", accessIp, iHosts[i].GetName())
		}
	}

	onPremiseNets, err := NetworkManager.fetchAllOnpremiseNetworks("", tristate.None)
	if err != nil {
		return errors.Wrap(err, "NetworkManager.fetchAllOnpremiseNetworks")
	}

	if zoneId == "" {
		zoneIds, err := fetchOnpremiseZoneIds(onPremiseNets)
		if err != nil {
			return errors.Wrap(err, "fetchOnpremiseZoneIds")
		}
		if len(zoneIds) == 0 {
			return errors.Wrap(httperrors.ErrInvalidStatus, "empty zone id?")
		}
		if len(zoneIds) == 1 {
			zoneId = zoneIds[0]
		} else {
			zoneId, err = guessEsxiZoneId(hostIps, onPremiseNets)
			if err != nil {
				return errors.Wrap(err, "fail to find zone of esxi")
			}
		}
	}

	netConfs, err := guessEsxiNetworks(hostIps, account.Name, onPremiseNets)
	if err != nil {
		return errors.Wrap(err, "guessEsxiNetworks")
	}
	log.Infof("netConfs: %s", jsonutils.Marshal(netConfs))
	{
		err := account.createNetworks(ctx, account.Name, zoneId, netConfs)
		if err != nil {
			return errors.Wrap(err, "account.createNetworks")
		}
	}

	return nil
}

func fetchOnpremiseZoneIds(onPremiseNets []SNetwork) ([]string, error) {
	var zoneIds []string
	for i := range onPremiseNets {
		zone, err := onPremiseNets[i].GetZone()
		if err != nil {
			return nil, errors.Wrapf(err, "onPremiseNets[%d].GetZone", i)
		}
		if !utils.IsInArray(zone.Id, zoneIds) {
			zoneIds = append(zoneIds, zone.Id)
		}
	}
	return zoneIds, nil
}

func guessEsxiZoneId(hostIps []netutils.IPV4Addr, onPremiseNets []SNetwork) (string, error) {
	zoneIds, err := ZoneManager.getOnpremiseZoneIds()
	if err != nil {
		return "", errors.Wrap(err, "getOnpremiseZoneIds")
	}
	if len(zoneIds) == 1 {
		return zoneIds[0], nil
	} else if len(zoneIds) == 0 {
		return "", errors.Wrap(httperrors.ErrNotFound, "no valid on-premise zone")
	}
	// there are multiple zones
	zoneIds = make([]string, 0)
	for _, ip := range hostIps {
		for _, net := range onPremiseNets {
			if net.IsAddressInRange(ip) || net.IsAddressInNet(ip) {
				zone, _ := net.GetZone()
				if zone != nil && !utils.IsInStringArray(zone.Id, zoneIds) {
					zoneIds = append(zoneIds, zone.Id)
				}
			}
		}
	}
	if len(zoneIds) == 0 {
		// no any clue
		return "", errors.Wrap(httperrors.ErrNotFound, "no valid on-premise networks")
	}
	if len(zoneIds) > 1 {
		// network span multiple zones
		return "", errors.Wrap(httperrors.ErrConflict, "spans multiple zones?")
	}
	return zoneIds[0], nil
}

type sIpv4List []netutils.IPV4Addr

func (a sIpv4List) Len() int           { return len(a) }
func (a sIpv4List) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sIpv4List) Less(i, j int) bool { return uint32(a[i]) < uint32(a[j]) }

func guessEsxiNetworks(hostIps []netutils.IPV4Addr, accountName string, onPremiseNets []SNetwork) ([]CANetConf, error) {
	netConfs := make([]CANetConf, 0)

	ipList := make([]netutils.IPV4Addr, 0)

	for j := 0; j < len(hostIps); j++ {
		var net *SNetwork
		for _, n := range onPremiseNets {
			if n.IsAddressInRange(hostIps[j]) {
				// already covered by existing network
				net = &n
				break
			}
		}
		if net != nil {
			// found a network contains this IP, no need to create
			continue
		}
		ipList = append(ipList, hostIps[j])
	}
	if len(ipList) == 0 {
		// no need to create network
		return nil, nil
	}
	sort.Sort(sIpv4List(ipList))
	for i := 0; i < len(ipList); {
		simNetConfs := CASimpleNetConf{}
		var net *SNetwork
		for _, n := range onPremiseNets {
			if n.IsAddressInNet(ipList[i]) {
				// already covered by existing network
				net = &n
				break
			}
		}

		if net != nil {
			simNetConfs.WireId = net.WireId
			simNetConfs.Gateway = net.GuestGateway
			simNetConfs.IpMask = net.GuestIpMask
		} else {
			simNetConfs.Gateway = (ipList[i].NetAddr(ipMaskLen) + netutils.IPV4Addr(options.Options.DefaultNetworkGatewayAddressEsxi)).String()
			simNetConfs.IpMask = ipMaskLen
		}
		netRange := netutils.NewIPV4AddrRange(ipList[i].NetAddr(simNetConfs.IpMask), ipList[i].BroadcastAddr(simNetConfs.IpMask))
		j := i
		for j < len(ipList)-1 {
			if ipList[j]+1 == ipList[j+1] && netRange.Contains(ipList[j+1]) {
				j++
			} else {
				break
			}
		}
		simNetConfs.IpStart = ipList[i].String()
		simNetConfs.IpEnd = ipList[j].String()
		i = j + 1
		netConfs = append(netConfs, CANetConf{
			Name:            fmt.Sprintf("%s-esxi-host-network", accountName),
			Description:     fmt.Sprintf("Auto created network for cloudaccount %q", accountName),
			CASimpleNetConf: simNetConfs,
		})
	}
	return netConfs, nil
}

func (account *SCloudaccount) createNetworks(ctx context.Context, accountName string, zoneId string, netConfs []CANetConf) error {
	var err error
	var defWireId string
	for i := range netConfs {
		var wireId = netConfs[i].WireId
		if len(wireId) == 0 {
			if len(defWireId) == 0 {
				// need to create one
				name := fmt.Sprintf("%s-wire", accountName)
				desc := fmt.Sprintf("Auto created wire for cloudaccount %q", accountName)
				wireId, err = account.createWire(ctx, api.DEFAULT_VPC_ID, zoneId, name, desc)
				if err != nil {
					return errors.Wrapf(err, "can't create wire %s", name)
				}
				defWireId = wireId
			}
			netConfs[i].WireId = defWireId
			wireId = defWireId
		}
		err := account.createNetwork(ctx, wireId, api.NETWORK_TYPE_BAREMETAL, netConfs[i])
		if err != nil {
			return errors.Wrapf(err, "can't create network %s", jsonutils.Marshal(netConfs[i]))
		}
	}
	return nil
}

// NETWORK_TYPE_GUEST     = "guest"
// NETWORK_TYPE_BAREMETAL = "baremetal"
func (account *SCloudaccount) createNetwork(ctx context.Context, wireId string, networkType api.TNetworkType, net CANetConf) error {
	network := &SNetwork{}

	if hint, err := NetworkManager.NewIfnameHint(net.Name); err != nil {
		log.Errorf("can't NewIfnameHint form hint %s", net.Name)
	} else {
		network.IfnameHint = hint
	}
	network.GuestIpStart = net.IpStart
	network.GuestIpEnd = net.IpEnd
	network.GuestIpMask = net.IpMask
	network.GuestGateway = net.Gateway
	network.VlanId = int(net.VlanID)
	network.WireId = wireId
	network.ServerType = networkType
	network.IsPublic = true
	network.Status = api.NETWORK_STATUS_AVAILABLE
	network.PublicScope = string(rbacscope.ScopeDomain)
	network.ProjectId = account.ProjectId
	network.DomainId = account.DomainId
	network.Description = net.Description

	lockman.LockClass(ctx, NetworkManager, network.ProjectId)
	defer lockman.ReleaseClass(ctx, NetworkManager, network.ProjectId)

	ownerId := network.GetOwnerId()
	nName, err := db.GenerateName(ctx, NetworkManager, ownerId, net.Name)
	if err != nil {
		return errors.Wrap(err, "GenerateName")
	}
	network.Name = nName

	network.SetModelManager(NetworkManager, network)
	// TODO: Prevent IP conflict
	log.Infof("create network %s succussfully", network.Id)
	err = NetworkManager.TableSpec().Insert(ctx, network)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}

	return nil
}

func (account *SCloudaccount) createWire(ctx context.Context, vpcId, zoneId, wireName, desc string) (string, error) {
	wire := &SWire{
		Bandwidth: 10000,
		Mtu:       1500,
	}
	wire.VpcId = vpcId
	wire.ZoneId = zoneId
	wire.IsEmulated = false
	wire.DomainId = account.GetOwnerId().GetDomainId()
	wire.Description = desc
	wire.Status = api.WIRE_STATUS_AVAILABLE
	wire.SetModelManager(WireManager, wire)

	lockman.LockClass(ctx, WireManager, wire.DomainId)
	defer lockman.ReleaseClass(ctx, WireManager, wire.DomainId)

	ownerId := wire.GetOwnerId()
	wName, err := db.GenerateName(ctx, WireManager, ownerId, wireName)
	if err != nil {
		return "", errors.Wrap(err, "GenerateName")
	}
	wire.Name = wName

	err = WireManager.TableSpec().Insert(ctx, wire)
	if err != nil {
		return "", errors.Wrap(err, "Insert")
	}
	log.Infof("create wire %s succussfully", wire.GetId())
	return wire.GetId(), nil
}
