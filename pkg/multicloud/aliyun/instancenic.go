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

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	id       string
	ipAddr   string
	macAddr  string

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.id
}

func (self *SInstanceNic) mustGetId() string {
	if self.id == "" {
		panic("empty network interface id")
	}
	return self.id
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetMAC() string {
	return self.macAddr
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	vswitchId := self.instance.VpcAttributes.VSwitchId
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wire := wires[i].(*SWire)
		net := wire.getNetworkById(vswitchId)
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	selfId := self.mustGetId()
	region := self.instance.host.zone.region
	params := map[string]string{
		"RegionId":             region.GetId(),
		"NetworkInterfaceId.1": selfId,
	}
	body, err := region.ecsRequest("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, err
	}

	type DescribeNetworkInterfacesResponse struct {
		TotalCount           int    `json:"TotalCount"`
		RequestID            string `json:"RequestId"`
		PageSize             int    `json:"PageSize"`
		NextToken            string `json:"NextToken"`
		PageNumber           int    `json:"PageNumber"`
		NetworkInterfaceSets struct {
			NetworkInterfaceSet []struct {
				Status             string `json:"Status"`
				PrivateIPAddress   string `json:"PrivateIpAddress"`
				ZoneID             string `json:"ZoneId"`
				ResourceGroupID    string `json:"ResourceGroupId"`
				InstanceID         string `json:"InstanceId"`
				VSwitchID          string `json:"VSwitchId"`
				NetworkInterfaceID string `json:"NetworkInterfaceId"`
				MacAddress         string `json:"MacAddress"`
				SecurityGroupIds   struct {
					SecurityGroupID []string `json:"SecurityGroupId"`
				} `json:"SecurityGroupIds"`
				Type     string `json:"Type"`
				Ipv6Sets struct {
					Ipv6Set []struct {
						Ipv6Address string `json:"Ipv6Address"`
					} `json:"Ipv6Set"`
				} `json:"Ipv6Sets"`
				VpcID              string `json:"VpcId"`
				OwnerID            string `json:"OwnerId"`
				AssociatedPublicIP struct {
				} `json:"AssociatedPublicIp"`
				CreationTime time.Time `json:"CreationTime"`
				Tags         struct {
					Tag []struct {
						TagKey   string `json:"TagKey"`
						TagValue string `json:"TagValue"`
					} `json:"Tag"`
				} `json:"Tags"`
				PrivateIPSets struct {
					PrivateIPSet []struct {
						PrivateIPAddress   string `json:"PrivateIpAddress"`
						AssociatedPublicIP struct {
						} `json:"AssociatedPublicIp"`
						Primary bool `json:"Primary"`
					} `json:"PrivateIpSet"`
				} `json:"PrivateIpSets"`
			} `json:"NetworkInterfaceSet"`
		} `json:"NetworkInterfaceSets"`
	}
	var resp DescribeNetworkInterfacesResponse
	if err := body.Unmarshal(&resp); err != nil {
		return nil, errors.Wrapf(err, "unmarshal DescribeNetworkInterfacesResponse: %s", body)
	}
	if got := len(resp.NetworkInterfaceSets.NetworkInterfaceSet); got != 1 {
		return nil, errors.Errorf("got %d element(s) in interface set, expect 1", got)
	}
	var (
		ipAddrs          []string
		networkInterface = resp.NetworkInterfaceSets.NetworkInterfaceSet[0]
	)
	if got := networkInterface.NetworkInterfaceID; got != selfId {
		return nil, errors.Errorf("got interface data for %s, expect %s", got, selfId)
	}
	for _, privateIP := range networkInterface.PrivateIPSets.PrivateIPSet {
		if !privateIP.Primary {
			ipAddrs = append(ipAddrs, privateIP.PrivateIPAddress)
		}
	}
	return ipAddrs, nil
}

func (self *SInstanceNic) ipAddrsParams(ipAddrs []string) map[string]string {
	region := self.instance.host.zone.region
	params := map[string]string{
		"RegionId":           region.GetId(),
		"NetworkInterfaceId": self.mustGetId(),
	}
	for i, ipAddr := range ipAddrs {
		k := fmt.Sprintf("PrivateIpAddress.%d", i+1)
		params[k] = ipAddr
	}
	return params
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	var (
		selfId   = self.mustGetId()
		instance = self.instance
		zone     = instance.host.zone
		region   = zone.region
	)
	ecsClient, err := region.getEcsClient()
	if err != nil {
		return err
	}
	request := ecs.CreateAssignPrivateIpAddressesRequest()
	request.Scheme = "https"

	request.NetworkInterfaceId = selfId
	request.PrivateIpAddress = &ipAddrs
	resp, err := ecsClient.AssignPrivateIpAddresses(request)
	if err != nil {
		return errors.Wrapf(err, "AssignPrivateIpAddresses")
	}

	allocated := resp.AssignedPrivateIpAddressesSet.PrivateIpSet.PrivateIpAddress
	if len(allocated) != len(ipAddrs) {
		return errors.Errorf("AssignAddress want %d addresses, got %d", len(ipAddrs), len(allocated))
	}
	for i := 0; i < len(ipAddrs); i++ {
		ip0 := ipAddrs[i]
		ip1 := allocated[i]
		if ip0 != ip1 {
			return errors.Errorf("AssignAddress address %d does not match: want %s, got %s", i, ip0, ip1)
		}
	}
	return nil
}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	var (
		selfId   = self.mustGetId()
		instance = self.instance
		zone     = instance.host.zone
		region   = zone.region
	)
	ecsClient, err := region.getEcsClient()
	if err != nil {
		return err
	}
	request := ecs.CreateUnassignPrivateIpAddressesRequest()
	request.Scheme = "https"

	request.NetworkInterfaceId = selfId
	request.PrivateIpAddress = &ipAddrs
	resp, err := ecsClient.UnassignPrivateIpAddresses(request)
	if err != nil {
		if resp.GetHttpStatus() == 404 {
			return nil
		}
		return err
	}
	return nil
}
