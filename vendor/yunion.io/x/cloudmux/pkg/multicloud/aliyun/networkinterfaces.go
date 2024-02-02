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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	AliyunTags
	region *SRegion

	InstanceId           string
	CreationTime         time.Time
	MacAddress           string
	ServiceManaged       bool
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
	Ipv6Sets             struct {
		Ipv6Set []struct {
			Ipv6Address string
		}
	}

	Attachment struct {
		InstanceId              string
		DeviceIndex             int
		TrunkNetworkInterfaceId string
	}
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
	nextToken := ""
	for {
		parts, _nextToken, err := region.GetNetworkInterfaces("", "Available", nextToken, 500)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, parts...)
		if len(_nextToken) == 0 {
			break
		}
		nextToken = _nextToken
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(interfaces); i++ {
		// 阿里云实例的弹性网卡已经在guestnetwork同步了
		if interfaces[i].ServiceManaged {
			continue
		}
		interfaces[i].region = region
		ret = append(ret, &interfaces[i])
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

func (region *SRegion) GetNetworkInterfaces(instanceId, status string, nextToken string, maxResults int) ([]SNetworkInterface, string, error) {
	if maxResults > 500 || maxResults <= 0 {
		maxResults = 500
	}

	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": fmt.Sprintf("%d", maxResults),
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	if len(status) > 0 {
		params["Status"] = status
	}

	body, err := region.ecsRequest("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, "", errors.Wrap(err, "DescribeNetworkInterfaces")
	}

	interfaces := []SNetworkInterface{}
	err = body.Unmarshal(&interfaces, "NetworkInterfaceSets", "NetworkInterfaceSet")
	if err != nil {
		return nil, "", errors.Wrap(err, "Unmarshal")
	}
	nextToken, _ = body.GetString("NextToken")
	return interfaces, nextToken, nil
}
