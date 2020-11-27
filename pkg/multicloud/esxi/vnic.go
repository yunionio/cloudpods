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
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SVirtualNIC struct {
	SVirtualDevice

	cloudprovider.DummyICloudNic
}

func NewVirtualNIC(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualNIC {
	return SVirtualNIC{
		SVirtualDevice: NewVirtualDevice(vm, dev, index),
	}
}

func (nic *SVirtualNIC) getVirtualEthernetCard() *types.VirtualEthernetCard {
	card := types.VirtualEthernetCard{}
	if FetchAnonymousFieldValue(nic.dev, &card) {
		return &card
	}
	return nil
}

func (nic *SVirtualNIC) GetId() string {
	return ""
}

func (nic *SVirtualNIC) GetIP() string {
	guestIps := nic.vm.getGuestIps()
	if ip, ok := guestIps[nic.GetMAC()]; ok {
		return ip
	}
	log.Warningf("cannot find ip for mac %s", nic.GetMAC())
	return ""
}

func (nic *SVirtualNIC) GetDriver() string {
	return nic.SVirtualDevice.GetDriver()
}

func (nic *SVirtualNIC) GetMAC() string {
	return netutils.FormatMacAddr(nic.getVirtualEthernetCard().MacAddress)
}

func (nic *SVirtualNIC) InClassicNetwork() bool {
	return false
}

func (nic *SVirtualNIC) GetINetwork() cloudprovider.ICloudNetwork {
	return nil
}
