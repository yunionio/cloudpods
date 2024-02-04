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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SPrivateIp struct {
	nic              *SNetworkInterface
	Primary          bool
	PrivateIpAddress string
}

func (ip *SPrivateIp) GetGlobalId() string {
	return ip.PrivateIpAddress
}

func (ip *SPrivateIp) GetINetworkId() string {
	return ip.nic.SubnetId
}

func (ip *SPrivateIp) GetIP() string {
	return ip.PrivateIpAddress
}

func (ip *SPrivateIp) IsPrimary() bool {
	return ip.Primary
}

type SAssociatedElasticIp struct {
	AllocationId string
	EipAddress   string
}

type SPrivateIpSets struct {
	PrivateIpSet []SPrivateIp
}

func (nic *SNetworkInterface) GetName() string {
	return nic.NetworkInterfaceName
}

func (nic *SNetworkInterface) GetId() string {
	return nic.NetworkInterfaceId
}

func (nic *SNetworkInterface) GetGlobalId() string {
	return nic.NetworkInterfaceId
}

func (nic *SNetworkInterface) GetAssociateId() string {
	return nic.InstanceId
}

func (nic *SNetworkInterface) GetAssociateType() string {
	return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER
}

func (nic *SNetworkInterface) GetMacAddress() string {
	return nic.MacAddress
}

func (nic *SNetworkInterface) GetStatus() string {
	switch nic.Status {
	case "Available":
		return api.NETWORK_INTERFACE_STATUS_AVAILABLE
	}
	return nic.Status
}

func (region *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	interfaces, err := region.GetNetworkInterfaces("", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(interfaces); i++ {
		if len(interfaces[i].InstanceId) == 0 {
			interfaces[i].region = region
			ret = append(ret, &interfaces[i])
		}
	}
	return ret, nil
}

func (nic *SNetworkInterface) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	address := []cloudprovider.ICloudInterfaceAddress{}
	for i := 0; i < len(nic.PrivateIpSets.PrivateIpSet); i++ {
		nic.PrivateIpSets.PrivateIpSet[i].nic = nic
		address = append(address, &nic.PrivateIpSets.PrivateIpSet[i])
	}
	return address, nil
}
