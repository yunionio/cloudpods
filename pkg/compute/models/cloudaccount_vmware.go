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
}

type CANetConf struct {
	CASimpleNetConf

	Name        string `json:"name"`
	Description string `json:"description"`
}

type CAPWire struct {
	VsId         string
	WireId       string
	Name         string
	Distributed  bool
	Description  string
	Hosts        []esxi.SSimpleHostDev
	HostNetworks []CANetConf
	// GuestNetworks []CANetConf
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
	// check network
	nInfo, err := esxiClient.HostVmIPsPro(ctx)
	if err != nil {
		return errors.Wrap(err, "esxiClient.HostVmIPsPro")
	}
	log.Infof("HostVmIPsPro: %s", jsonutils.Marshal(nInfo).String())

	capWires := make([]CAPWire, 0)
	vsList := nInfo.VsMap.List()
	log.Infof("vsList: %s", jsonutils.Marshal(vsList))

	onPremiseNets, err := NetworkManager.fetchAllOnpremiseNetworks("", tristate.None)
	if err != nil {
		return errors.Wrap(err, "NetworkManager.fetchAllOnpremiseNetworks")
	}

	if zoneId == "" {
		zoneId, err = guessEsxiZoneId(vsList, onPremiseNets)
		if err != nil {
			return errors.Wrap(err, "fail to find zone of esxi")
		}
	}

	desc := fmt.Sprintf("Auto create for cloudaccount %q", account.Name)
	for i := range vsList {
		vs := vsList[i]
		wireName := fmt.Sprintf("%s/%s", account.Name, vs.Name)
		capWire, err := guessEsxiNetworks(vs, wireName, onPremiseNets)
		if err != nil {
			return errors.Wrap(err, "guessEsxiNetworks")
		}
		if len(capWire.WireId) == 0 {
			capWire.Description = desc
		}
		capWires = append(capWires, *capWire)
	}
	log.Infof("capWires: %v", capWires)
	host2Wire, err := account.createNetworks(ctx, zoneId, capWires)
	if err != nil {
		return errors.Wrap(err, "account.createNetworks")
	}
	err = account.SetHost2Wire(ctx, userCred, host2Wire)
	if err != nil {
		return errors.Wrap(err, "account.SetHost2Wire")
	}
	return nil
}

func guessEsxiZoneId(vsList []esxi.SVirtualSwitchSpec, onPremiseNets []SNetwork) (string, error) {
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
	for i := range vsList {
		vs := vsList[i]
		for _, ips := range vs.HostIps {
			for _, ip := range ips {
				for _, net := range onPremiseNets {
					if net.IsAddressInRange(ip) || net.IsAddressInNet(ip) {
						zone, _ := net.GetZone()
						if zone != nil && !utils.IsInStringArray(zone.Id, zoneIds) {
							zoneIds = append(zoneIds, zone.Id)
						}
					}
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

func guessEsxiNetworks(vs esxi.SVirtualSwitchSpec, wireName string, onPremiseNets []SNetwork) (*CAPWire, error) {
	capWire := CAPWire{
		Name:        wireName,
		VsId:        vs.Id,
		Distributed: vs.Distributed,
		Hosts:       vs.Hosts,
	}
	ipList := make([]netutils.IPV4Addr, 0)
	for _, ips := range vs.HostIps {
		for j := 0; j < len(ips); j++ {
			var net *SNetwork
			for _, n := range onPremiseNets {
				if n.IsAddressInRange(ips[j]) {
					// already covered by existing network
					net = &n
					break
				}
			}
			if net != nil {
				// found a network contains this IP, no need to create
				if len(capWire.WireId) > 0 && capWire.WireId != net.WireId {
					return nil, errors.Wrapf(httperrors.ErrConflict, "%s seems attaching conflict wires %s and %s", wireName, capWire.WireId, net.WireId)
				}
				capWire.WireId = net.WireId
				continue
			}
			ipList = append(ipList, ips[j])
		}
	}
	if len(ipList) == 0 {
		return &capWire, nil
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
			if len(capWire.WireId) > 0 && capWire.WireId != net.WireId {
				return nil, errors.Wrapf(httperrors.ErrConflict, "%s seems attaching conflict wires %s and %s", wireName, capWire.WireId, net.WireId)
			}
			capWire.WireId = net.WireId
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
		capWire.HostNetworks = append(capWire.HostNetworks, CANetConf{
			Name:            fmt.Sprintf("%s-host-network-%d", wireName, len(capWire.HostNetworks)+1),
			Description:     fmt.Sprintf("Auto create for cloudaccount %q", wireName),
			CASimpleNetConf: simNetConfs,
		})
	}
	return &capWire, nil
}

func (account *SCloudaccount) createNetworks(ctx context.Context, zoneId string, capWires []CAPWire) (map[string][]SVs2Wire, error) {
	var err error
	ret := make(map[string][]SVs2Wire)
	for i := range capWires {
		// if len(capWires[i].GuestNetworks)+len(capWires[i].HostNetworks) == 0 {
		if len(capWires[i].HostNetworks) == 0 {
			continue
		}
		var wireId = capWires[i].WireId
		if len(wireId) == 0 {
			wireId, err = account.createWire(ctx, api.DEFAULT_VPC_ID, zoneId, capWires[i].Name, capWires[i].Description)
			if err != nil {
				return nil, errors.Wrapf(err, "can't create wire %s", capWires[i].Name)
			}
			capWires[i].WireId = wireId
		}
		for _, net := range capWires[i].HostNetworks {
			err := account.createNetwork(ctx, wireId, api.NETWORK_TYPE_BAREMETAL, net)
			if err != nil {
				return nil, errors.Wrapf(err, "can't create network %v", net)
			}
		}
		for _, host := range capWires[i].Hosts {
			ret[host.Id] = append(ret[host.Id], SVs2Wire{
				VsId:        capWires[i].VsId,
				WireId:      capWires[i].WireId,
				Distributed: capWires[i].Distributed,
				Mac:         host.Mac,
			})
		}
	}
	return ret, nil
}

// NETWORK_TYPE_GUEST     = "guest"
// NETWORK_TYPE_BAREMETAL = "baremetal"
func (account *SCloudaccount) createNetwork(ctx context.Context, wireId, networkType string, net CANetConf) error {
	network := &SNetwork{}
	network.Name = net.Name
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

	network.SetModelManager(NetworkManager, network)
	// TODO: Prevent IP conflict
	log.Infof("create network %s succussfully", network.Id)
	err := NetworkManager.TableSpec().Insert(ctx, network)
	return err
}

func (account *SCloudaccount) createWire(ctx context.Context, vpcId, zoneId, wireName, desc string) (string, error) {
	wire := &SWire{
		Bandwidth: 10000,
		Mtu:       1500,
	}
	wire.VpcId = vpcId
	wire.ZoneId = zoneId
	wire.IsEmulated = false
	wire.Name = wireName
	wire.DomainId = account.GetOwnerId().GetDomainId()
	wire.Description = desc
	wire.Status = api.WIRE_STATUS_AVAILABLE
	wire.SetModelManager(WireManager, wire)
	err := WireManager.TableSpec().Insert(ctx, wire)
	if err != nil {
		return "", err
	}
	log.Infof("create wire %s succussfully", wire.GetId())
	return wire.GetId(), nil
}
