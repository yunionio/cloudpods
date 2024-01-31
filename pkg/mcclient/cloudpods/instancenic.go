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

package cloudpods

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SInstanceNic struct {
	multicloud.SResourceBase
	ins *SInstance

	api.GuestnetworkDetails
}

func (self *SInstanceNic) GetId() string {
	return fmt.Sprintf("%d", self.RowId)
}

func (self *SInstanceNic) GetIP() string {
	return self.IpAddr
}

func (self *SInstanceNic) GetIP6() string {
	return self.Ip6Addr
}

func (self *SInstanceNic) GetDriver() string {
	return self.Driver
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetMAC() string {
	return self.MacAddr
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	return []string{}, nil
}

func (self *SInstanceNic) AssignNAddress(count int) ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstanceNic) UnassignAddress(IpAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.NetworkId
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.host.zone.region.GetGuestnetworks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNic{}
	for i := range nics {
		nics[i].ins = self
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SRegion) GetGuestnetworks(serverId string) ([]SInstanceNic, error) {
	ret := []SInstanceNic{}
	params := map[string]interface{}{
		"scope": "system",
	}
	if len(serverId) > 0 {
		params["server_id"] = serverId
	}
	resp, err := modules.Servernetworks.List(self.cli.s, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	return ret, jsonutils.Update(&ret, resp.Data)
}
