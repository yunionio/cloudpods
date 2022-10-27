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

package jdcloud

import (
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/models"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetworkInterface struct {
	multicloud.SNetworkInterfaceBase
	JdcloudTags
	region *SRegion

	models.NetworkInterface
}

func (nic *SNetworkInterface) GetId() string {
	return nic.NetworkInterfaceId
}

func (nic *SNetworkInterface) GetName() string {
	return nic.NetworkInterfaceName
}

func (nic *SNetworkInterface) GetGlobalId() string {
	return nic.GetId()
}

func (nic *SNetworkInterface) GetAssociateId() string {
	return nic.InstanceId
}

func (nic *SNetworkInterface) GetAssociateType() string {
	if nic.InstanceType == "vm" {
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER
	}
	return ""
}

func (nic *SNetworkInterface) GetMacAddress() string {
	return nic.MacAddress
}

func (nic *SNetworkInterface) GetStatus() string {
	switch nic.NetworkInterfaceStatus {
	case "enabled":
		return api.NETWORK_INTERFACE_STATUS_AVAILABLE
	case "disabled":
		return api.NETWORK_INTERFACE_STATUS_DISABLED
	}
	return api.NETWORK_INTERFACE_STATUS_UNKNOWN
}

type SNetworkInterfacePrivateIp struct {
	subnetId  string
	isPrimary bool
	models.NetworkInterfacePrivateIp
}

func (ip *SNetworkInterfacePrivateIp) GetGlobalId() string {
	return ip.PrivateIpAddress
}

func (ip *SNetworkInterfacePrivateIp) GetINetworkId() string {
	return ip.subnetId
}

func (ip *SNetworkInterfacePrivateIp) GetIP() string {
	return ip.PrivateIpAddress
}

func (ip *SNetworkInterfacePrivateIp) IsPrimary() bool {
	return ip.isPrimary
}

func (nic *SNetworkInterface) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	address := make([]cloudprovider.ICloudInterfaceAddress, 0, len(nic.SecondaryIps)+1)
	address = append(address, &SNetworkInterfacePrivateIp{
		subnetId:                  nic.SubnetId,
		isPrimary:                 true,
		NetworkInterfacePrivateIp: nic.PrimaryIp,
	})
	for i := range nic.SecondaryIps {
		address = append(address, &SNetworkInterfacePrivateIp{
			subnetId:                  nic.SubnetId,
			isPrimary:                 false,
			NetworkInterfacePrivateIp: nic.SecondaryIps[i],
		})
	}
	return address, nil
}
