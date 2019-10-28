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

package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	return ip.nic.VSwitchId
}

func (ip *SPrivateIp) GetIP() string {
	return ip.PrivateIpAddress
}

func (ip *SPrivateIp) IsPrimary() bool {
	return ip.Primary
}

type SPrivateIpSets struct {
	PrivateIpSet []SPrivateIp
}

type SNetworkInterface struct {
	multicloud.SNetworkInterfaceBase
	region *SRegion

	InstanceId           string
	CreationTime         time.Time
	MacAddress           string
	NetworkInterfaceName string
	PrivateIpSets        SPrivateIpSets
	ResourceGroupId      string
	SecurityGroupIds     SSecurityGroupIds
	Status               string
	Type                 string
	VSwitchId            string
	VpcId                string
	ZoneId               string
	NetworkInterfaceId   string
	PrimaryIpAddress     string
	PrivateIpAddress     string
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
	interfaces := []SNetworkInterface{}
	for {
		parts, total, err := region.GetNetworkInterfaces("", len(interfaces), 50)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, parts...)
		if len(interfaces) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(interfaces); i++ {
		// 阿里云实例的弹性网卡已经在guestnetwork同步了
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

func (region *SRegion) GetNetworkInterfaces(instanceId string, offset int, limit int) ([]SNetworkInterface, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}

	params := map[string]string{
		"RegionId":   region.RegionId,
		"PageSize":   fmt.Sprintf("%d", limit),
		"PageNumber": fmt.Sprintf("%d", (offset/limit)+1),
	}

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}

	body, err := region.ecsRequest("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeNetworkInterfaces")
	}

	interfaces := []SNetworkInterface{}
	err = body.Unmarshal(&interfaces, "NetworkInterfaceSets", "NetworkInterfaceSet")
	if err != nil {
		return nil, 0, errors.Wrap(err, "Unmarshal")
	}
	total, _ := body.Int("TotalCount")
	return interfaces, int(total), nil
}
