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

package remotefile

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"
)

type SInstanceNic struct {
	SResourceBase

	Ip        string
	Mac       string
	Classic   bool
	Driver    string
	NetworkId string
	SubAddr   []string
}

func (self *SInstanceNic) GetIP() string {
	return self.Ip
}

func (self *SInstanceNic) GetMAC() string {
	if len(self.Mac) == 0 {
		ip, _ := netutils.NewIPV4Addr(self.GetIP())
		return ip.ToMac("00:16:")
	}
	return self.Mac
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return self.Classic
}

func (self *SInstanceNic) GetDriver() string {
	return self.Driver
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.NetworkId
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	return self.SubAddr, nil
}

func (self *SInstanceNic) AssignNAddress(count int) ([]string, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotSupported
}
