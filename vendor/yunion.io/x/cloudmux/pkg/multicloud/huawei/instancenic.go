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

package huawei

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	ipAddr   string
	macAddr  string

	subAddrs []string
	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	return self.subAddrs, nil
}

func (self *SInstanceNic) GetMAC() string {
	return self.macAddr
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetINetworkId() string {
	subnets, _ := self.instance.host.zone.region.getSubnetIdsByInstanceId(self.instance.GetId())
	for _, id := range subnets {
		return id
	}
	return ""
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ListServerInterfaces
func (self *SRegion) getSubnetIdsByInstanceId(instanceId string) ([]string, error) {
	resp, err := self.list(SERVICE_ECS, fmt.Sprintf("cloudservers/%s/os-interface", instanceId), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list os interface")
	}
	ret := struct {
		InterfaceAttachments []struct {
			NetId string
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	subnets := []string{}
	for _, att := range ret.InterfaceAttachments {
		subnets = append(subnets, att.NetId)
	}
	return subnets, nil
}
