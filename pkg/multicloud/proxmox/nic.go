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

package proxmox

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	cloudprovider.DummyICloudNic
	ins *SInstance

	MacAddr   string
	IpAddr    string
	NicId     string
	Model     string
	NetworkId string
}

func (self *SInstanceNic) GetId() string {
	return fmt.Sprintf("%d/%s", self.ins.VmID, self.NicId)
}

func (self *SInstanceNic) GetIP() string {
	return self.IpAddr
}

func (self *SInstanceNic) GetMAC() string {
	return self.MacAddr
}

func (self *SInstanceNic) GetDriver() string {
	return strings.ToLower(self.Model)
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return true
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	return []string{}, nil
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.NetworkId
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotSupported
}
