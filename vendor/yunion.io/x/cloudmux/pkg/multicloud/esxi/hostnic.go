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
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type sHostNicInfo struct {
	Dev     string
	Driver  string
	Mac     string
	Index   int8
	LinkUp  bool
	IpAddr  string
	IpAddr6 string
	Mtu     int32
	NicType compute.TNicType

	VlanId int

	IpAddrPrefixLen  int8
	IpAddr6PrefixLen int8

	host    *SHost
	network IVMNetwork
}

func (nic *sHostNicInfo) GetDevice() string {
	return nic.Dev
}

func (nic *sHostNicInfo) GetDriver() string {
	return nic.Driver
}

func (nic *sHostNicInfo) GetMac() string {
	if len(nic.Mac) > 0 {
		return nic.Mac
	}
	if nic.network != nil {
		return cloudprovider.HashIdsMac(nic.host.GetGlobalId(), nic.network.GetId())
	}
	panic("sHostNicInfo: empty mac and network?")
}

func (nic *sHostNicInfo) GetVlanId() int {
	if nic.VlanId > 0 {
		return nic.VlanId
	}
	return 1
}

func (nic *sHostNicInfo) GetIndex() int8 {
	return nic.Index
}

func (nic *sHostNicInfo) IsLinkUp() tristate.TriState {
	if nic.LinkUp {
		return tristate.True
	}
	return tristate.False
}

func (nic *sHostNicInfo) GetIpAddr() string {
	return nic.IpAddr
}

func (nic *sHostNicInfo) GetIpAddrPrefixLen() int8 {
	return nic.IpAddrPrefixLen
}

func (nic *sHostNicInfo) GetIpAddr6() string {
	return nic.IpAddr6
}

func (nic *sHostNicInfo) GetIpAddr6PrefixLen() int8 {
	return nic.IpAddr6PrefixLen
}

func (nic *sHostNicInfo) GetMtu() int32 {
	return nic.Mtu
}

func (nic *sHostNicInfo) GetNicType() string {
	return string(nic.NicType)
}

func (nic *sHostNicInfo) GetBridge() string {
	if nic.network != nil {
		return nic.network.GetId()
	}
	return ""
}

func (nic *sHostNicInfo) GetIWire() cloudprovider.ICloudWire {
	if nic.network != nil {
		return &sWire{
			network: nic.network,
			client:  nic.host.manager,
		}
	}
	return nil
}
