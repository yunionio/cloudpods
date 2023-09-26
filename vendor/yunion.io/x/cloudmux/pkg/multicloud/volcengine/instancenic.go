// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"

	"github.com/golang-plus/errors"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	cloudprovider.DummyICloudNic

	instance *SInstance

	id      string
	ipAddr  string
	macAddr string
}

func (nic *SInstanceNic) GetId() string {
	return nic.id
}

func (nic *SInstanceNic) GetIP() string {
	return nic.ipAddr
}

func (nic *SInstanceNic) GetMAC() string {
	return nic.macAddr
}

func (nic *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetworkId() string {
	return nic.instance.NetworkInterfaces[0].SubnetId
}

func (nic *SInstanceNic) GetSubAddress() ([]string, error) {
	return nic.instance.host.zone.region.GetSubAddress(nic.id)
}

func (nic *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return nic.instance.host.zone.region.AssignAddres(nic.id, ipAddrs)
}

func (nic *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	return nic.instance.host.zone.region.UnassignAddress(nic.id, ipAddrs)
}

func (region *SRegion) GetSubAddress(nicId string) ([]string, error) {
	params := map[string]string{
		"NetworkInterfaceId.1": nicId,
	}
	body, err := region.vpcRequest("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeNetworkInterfaces")
	}

	interfaces := []SNetworkInterface{}
	err = body.Unmarshal(&interfaces, "Result", "NetworkInterfaceSets")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	ipAddrs := []string{}
	for _, net := range interfaces {
		if net.NetworkInterfaceId != nicId {
			continue
		}
		for _, addr := range net.PrivateIpSets.PrivateIpSet {
			if !addr.Primary {
				ipAddrs = append(ipAddrs, addr.PrivateIpAddress)
			}
		}
	}
	return ipAddrs, nil
}

func (region *SRegion) AssignAddres(nicId string, ipAddrs []string) error {
	params := make(map[string]string)
	params["NetworkInterfaceId"] = nicId
	for idx, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", idx+1)] = addr
	}
	_, err := region.vpcRequest("AssignPrivateIpAddresses", params)
	if err != nil {
		return errors.Wrapf(err, "AssignPrivateIpAddresses")
	}
	return nil
}

func (region *SRegion) UnassignAddress(nicId string, ipAddrs []string) error {
	params := make(map[string]string)
	params["NetworkInterfaceId"] = nicId
	for idx, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", idx+1)] = addr
	}
	_, err := region.vpcRequest("UnassignPrivateAddress", params)
	if err != nil {
		return errors.Wrapf(err, "UnassignPrivateAdress")
	}
	return nil
}
