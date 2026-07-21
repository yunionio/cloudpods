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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// PrepareProxmoxHostNetwork prepares on-premise L2 wires and IP subnets for Proxmox hosts.
// Proxmox does not sync remote L2 wires; host-nics only attach to on-premise wires.
// Flow:
//  1. collect host IP list from Proxmox host nics (bridges included)
//  2. check whether local KVM/on-premise networks already cover those IPs
//  3. if not, create a L2 wire and corresponding IP subnet
//  4. later SyncHostExternalNics associates host-nics onto the prepared on-premise wire by IP
func (account *SCloudaccount) PrepareProxmoxHostNetwork(ctx context.Context, userCred mcclient.TokenCredential, zoneId string) error {
	cProvider, err := account.GetProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "GetProvider")
	}
	region, err := cProvider.GetOnPremiseIRegion()
	if err != nil {
		return errors.Wrap(err, "GetOnPremiseIRegion")
	}
	iHosts, err := region.GetIHosts()
	if err != nil {
		return errors.Wrap(err, "GetIHosts")
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
			ipAddrStr := hn.GetIpAddr()
			if len(ipAddrStr) == 0 {
				// skip interface without a valid ip address
				continue
			}
			if ipAddrStr == "127.0.0.1" {
				continue
			}
			if accessIp == ipAddrStr {
				findAccessIp = true
			}
			ipAddr, err := netutils.NewIPV4Addr(ipAddrStr)
			if err != nil {
				log.Errorf("fail to parse ipv4 addr %s: %s", ipAddrStr, err)
				continue
			}
			hostIps = append(hostIps, ipAddr)
		}
		if !findAccessIp && len(accessIp) > 0 {
			log.Errorf("Fail to find access ip %s NIC for proxmox host %s", accessIp, iHosts[i].GetName())
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
			zoneIds, err = ZoneManager.getOnpremiseZoneIds()
			if err != nil {
				return errors.Wrap(err, "getOnpremiseZoneIds")
			}
		}
		if len(zoneIds) == 0 {
			return errors.Wrap(httperrors.ErrInvalidStatus, "empty zone id?")
		}
		if len(zoneIds) == 1 {
			zoneId = zoneIds[0]
		} else {
			zoneId, err = guessEsxiZoneId(hostIps, onPremiseNets)
			if err != nil {
				return errors.Wrap(err, "fail to find zone of proxmox")
			}
		}
	}

	netConfs, err := guessEsxiNetworks(hostIps, account.Name, onPremiseNets)
	if err != nil {
		return errors.Wrap(err, "guessEsxiNetworks")
	}
	for i := range netConfs {
		netConfs[i].Name = fmt.Sprintf("%s-proxmox-host-network", account.Name)
		netConfs[i].Description = fmt.Sprintf("Auto created network for proxmox cloudaccount %q", account.Name)
	}
	log.Infof("proxmox netConfs: %s", jsonutils.Marshal(netConfs))
	err = account.createNetworks(ctx, account.Name, zoneId, netConfs)
	if err != nil {
		return errors.Wrap(err, "account.createNetworks")
	}

	return nil
}
