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

package cucloud

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"
)

type SInstanceNic struct {
	cloudprovider.DummyICloudNic

	SecurityGroupId  string
	FixIpAddress     string
	VirtualMachineId string
	NetCardName      string
	NetCardDefault   string
	InnerFloatingIP  string
	NetCardId        string
	NetworkId        string
	SubNetworkId     string
}

func (nic *SInstanceNic) GetId() string {
	return nic.NetCardId
}

func (nic *SInstanceNic) GetIP() string {
	return nic.FixIpAddress
}

func (nic *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (nic *SInstanceNic) GetIP6() string {
	return ""
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetworkId() string {
	return nic.SubNetworkId
}

func (nic *SInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(nic.GetIP())
	return ip.ToMac("00:16:")
}

func (nic *SInstanceNic) GetSubAddress() ([]string, error) {
	return []string{}, nil
}
