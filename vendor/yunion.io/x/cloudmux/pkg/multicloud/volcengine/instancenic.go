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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SInstanceNic struct {
	instance *SInstance

	id      string
	ipAddr  string
	macAddr string
}

type SNetworkInterface struct {
	cloudprovider.DummyICloudNic
	multicloud.SNetworkInterfaceBase
	VolcEngineTags
	region *SRegion

	InstanceId           string
	NetworkInterfaceId   string
	VpcId                string
	SubnetId             string
	PrimaryIpAddress     string
	Type                 string
	MacAddress           string
	CreationTime         time.Time
	NetworkInterfaceName string
	PrivateIpSets        SPrivateIpSets
	ResourceGroupId      string
	SecurityGroupIds     SSecurityGroupIds
	Status               string
	ZoneId               string
	PrivateIpAddresses   []string
	AssociatedElasticIp  SAssociatedElasticIp
	IPv6Sets             []string
}

func (nic *SNetworkInterface) GetIP() string {
	return nic.PrimaryIpAddress
}

func (nic *SNetworkInterface) GetIP6() string {
	for _, ip := range nic.IPv6Sets {
		return ip
	}
	return ""
}

func (nic *SNetworkInterface) GetMAC() string {
	return nic.MacAddress
}

func (nic *SNetworkInterface) InClassicNetwork() bool {
	return false
}

func (nic *SNetworkInterface) GetDriver() string {
	return "virtio"
}

func (nic *SNetworkInterface) GetINetworkId() string {
	return nic.SubnetId
}

func (nic *SNetworkInterface) GetSubAddress() ([]string, error) {
	return nic.region.GetSubAddress(nic.NetworkInterfaceId)
}

func (nic *SNetworkInterface) AssignAddress(ipAddrs []string) error {
	return nic.region.AssignAddres(nic.NetworkInterfaceId, ipAddrs)
}

func (nic *SNetworkInterface) UnassignAddress(ipAddrs []string) error {
	return nic.region.UnassignAddress(nic.NetworkInterfaceId, ipAddrs)
}

func (region *SRegion) GetSubAddress(nicId string) ([]string, error) {
	nics, err := region.GetNetworkInterfaces(nicId, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworkInterfaces")
	}
	ipAddrs := []string{}
	for _, net := range nics {
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

func (region *SRegion) GetNetworkInterfaces(nicId, instanceId string) ([]SNetworkInterface, error) {
	params := map[string]string{
		"PageSize": "100",
	}
	if len(nicId) > 0 {
		params["NetworkInterfaceId.1"] = nicId
	}
	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	ret := []SNetworkInterface{}
	for {
		resp, err := region.vpcRequest("DescribeNetworkInterfaces", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeNetworkInterfaces")
		}
		part := struct {
			NetworkInterfaceSets []SNetworkInterface
			NextToken            string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.NetworkInterfaceSets...)
		if len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
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
