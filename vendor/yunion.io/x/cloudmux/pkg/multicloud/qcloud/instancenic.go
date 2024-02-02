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

package qcloud

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"
)

type SInstanceNic struct {
	instance *SInstance

	id      string
	ipAddr  string
	ip6Addr string
	macAddr string
	classic bool

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.id
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetIP6() string {
	return self.ip6Addr
}

func (self *SInstanceNic) GetMAC() string {
	if len(self.macAddr) == 0 {
		ip, _ := netutils.NewIPV4Addr(self.GetIP())
		return ip.ToMac("00:16:")
	}
	return self.macAddr
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return self.classic
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.instance.VirtualPrivateCloud.SubnetId
}
