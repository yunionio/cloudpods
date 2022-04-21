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

package apsara

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SIPv6Gateway struct {
	multicloud.SResourceBase
	multicloud.ApsaraTags
	region *SRegion

	Status             string    `json:"Status"`
	Description        string    `json:"Description"`
	InstanceChargeType string    `json:"InstanceChargeType"`
	Ipv6GatewayId      string    `json:"Ipv6GatewayId"`
	BusinessStatus     string    `json:"BusinessStatus"`
	Name               string    `json:"Name"`
	VpcID              string    `json:"VpcId"`
	ExpiredTime        string    `json:"ExpiredTime"`
	CreationTime       time.Time `json:"CreationTime"`
	RegionId           string    `json:"RegionId"`
	Spec               string    `json:"Spec"`

	DepartmentInfo
}

func (self *SIPv6Gateway) GetGlobalId() string {
	return self.Ipv6GatewayId
}

func (self *SIPv6Gateway) GetId() string {
	return self.Ipv6GatewayId
}

func (self *SIPv6Gateway) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.Ipv6GatewayId
}

func (self *SIPv6Gateway) GetStatus() string {
	return self.Status
}

func (self *SIPv6Gateway) GetInstanceType() string {
	return self.Spec
}

func (self *SIPv6Gateway) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SRegion) GetIPv6Gateways(vpcId string, pageNumber, pageSize int) ([]SIPv6Gateway, int, error) {
	if pageSize < 1 {
		pageSize = 50
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	params := map[string]string{
		"PageSize":   fmt.Sprintf("%d", pageSize),
		"PageNumber": fmt.Sprintf("%d", 1),
	}
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	resp, err := self.vpcRequest("DescribeIpv6Gateways", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeIpv6Gateways")
	}
	ret := []SIPv6Gateway{}
	err = resp.Unmarshal(&ret, "Ipv6Gateways", "Ipv6Gateway")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SRegion) GetIPv6Gateway(id string) (*SIPv6Gateway, error) {
	params := map[string]string{
		"Ipv6GatewayId": id,
	}
	resp, err := self.vpcRequest("DescribeIpv6GatewayAttribute", params)
	if err != nil {
		return nil, err
	}
	ret := &SIPv6Gateway{region: self}
	return ret, resp.Unmarshal(ret)
}

func (self *SVpc) GetICloudIPv6Gateways() ([]cloudprovider.ICloudIPv6Gateway, error) {
	res, pageNumber := []SIPv6Gateway{}, 1
	for {
		part, total, err := self.region.GetIPv6Gateways(self.VpcId, pageNumber, 50)
		if err != nil {
			return nil, err
		}
		res = append(res, part...)
		if len(res) >= total {
			break
		}
		pageNumber++
	}
	ret := []cloudprovider.ICloudIPv6Gateway{}
	for i := range res {
		res[i].region = self.region
		ret = append(ret, &res[i])
	}
	return ret, nil
}
