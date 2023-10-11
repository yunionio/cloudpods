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

package ctyun

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"
)

type SInstanceNic struct {
	instance *SInstance

	IPv4Address   string
	IPv6Address   []string
	IsMaster      bool
	SubnetCidr    string
	NetworkCardId string
	Gateway       string
	SecurityGroup []string
	SubnetId      string

	cloudprovider.DummyICloudNic
}

type FixedIP struct {
	IPAddress string `json:"ip_address"`
	SubnetID  string `json:"subnet_id"`
}

func (self *SInstanceNic) GetIP() string {
	return self.IPv4Address
}

func (self *SInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(self.GetIP())
	return ip.ToMac("fa:16:")
}

func (self *SInstanceNic) GetId() string {
	return self.NetworkCardId
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.SubnetId
}
