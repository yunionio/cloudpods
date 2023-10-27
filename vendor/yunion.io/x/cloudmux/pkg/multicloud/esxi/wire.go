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

package esxi

import (
	"fmt"
	"time"

	"github.com/vmware/govmomi/vim25/mo"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type sWire struct {
	network IVMNetwork

	client *SESXiClient
}

func (wire *sWire) GetId() string {
	return wire.network.GetId()
}

func (wire *sWire) GetName() string {
	return wire.network.GetName()
}

func (wire *sWire) GetGlobalId() string {
	if wire.client.IsVCenter() {
		return fmt.Sprintf("%s/%s", wire.client.GetUUID(), wire.network.GetId())
	} else {
		return wire.network.GetId()
	}
}

func (wire *sWire) GetCreatedAt() time.Time {
	return time.Time{}
}

func (wire *sWire) GetDescription() string {
	return fmt.Sprintf("%s %s %s", wire.network.GetType(), wire.network.GetName(), wire.network.GetId())
}

func (wire *sWire) GetStatus() string {
	return compute.WIRE_STATUS_AVAILABLE
}

func (wire *sWire) Refresh() error {
	return nil
}

func (wire *sWire) IsEmulated() bool {
	return false
}

func (wire *sWire) GetTags() (map[string]string, error) {
	return nil, nil
}

func (wire *sWire) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (wire *sWire) GetIVpc() cloudprovider.ICloudVpc {
	return wire.client.fakeVpc
}

func (wire *sWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (wire *sWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	return nil, errors.ErrNotSupported
}

func (wire *sWire) GetBandwidth() int {
	return 10000
}

func (wire *sWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	return nil, errors.ErrNotFound
}

func (wire *sWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, errors.ErrNotSupported
}

func (wire *sWire) getAvailableIpsMacs() ([]string, []string, error) {
	var hosts []mo.HostSystem
	err := wire.client.references2Objects(wire.network.GetHosts(), HOST_PROPS, &hosts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "references2Objects HOST_PROPS")
	}
	retIps := make([]string, 0)
	retMacs := make([]string, 0)
	for i := range hosts {
		h := hosts[i]
		ips, macs, err := wire.getAvailableIpsMacsOnHost(h)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "getAvailableIpsOnHost %s", h.Name)
		}
		retIps = append(retIps, ips...)
		retMacs = append(retMacs, macs...)
	}
	return retIps, retMacs, nil
}

func (wire *sWire) getAvailableIpsMacsOnVM(vm mo.VirtualMachine) ([]string, []string) {
	ips := make([]string, 0)
	macs := make([]string, 0)
	for _, net := range vm.Guest.Net {
		if net.Network != wire.GetName() {
			continue
		}
		if len(net.MacAddress) > 0 {
			mac := netutils.FormatMacAddr(net.MacAddress)
			if len(mac) > 0 {
				macs = append(macs, mac)
			}
		}
		for _, ip := range net.IpAddress {
			if regutils.MatchIP4Addr(ip) {
				ips = append(ips, ip)
			}
		}
	}
	return ips, macs
}

func (wire *sWire) getAvailableIpsMacsOnHost(host mo.HostSystem) ([]string, []string, error) {
	var vms []mo.VirtualMachine
	err := wire.client.references2Objects(host.Vm, VM_PROPS, &vms)
	if err != nil {
		return nil, nil, errors.Wrap(err, "references2Objects VM_PROPS")
	}
	retIps := make([]string, 0)
	retMacs := make([]string, 0)
	for _, vm := range vms {
		ips, macs := wire.getAvailableIpsMacsOnVM(vm)
		retIps = append(retIps, ips...)
		retMacs = append(retMacs, macs...)
	}
	return retIps, retMacs, nil
}
