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

import "fmt"

type SGtmInstanceConfigs struct {
	InstanceId         string
	ScheduleDomainName string
	ScheduleZoneName   string
	AddressPools       struct {
		AddressPool []struct {
			AddressPoolId   string
			AddressPoolName string
		}
	}
}

func (self *SAliyunClient) ListCloudGtmInstanceConfigs() ([]SGtmInstanceConfigs, error) {
	params := map[string]string{"PageSize": "100"}
	ret := []SGtmInstanceConfigs{}
	pageNumber := 1
	for {
		params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
		resp, err := self.alidnsRequest("ListCloudGtmInstanceConfigs", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			InstanceConfigs struct {
				InstanceConfig []SGtmInstanceConfigs
			}
			TotalItems int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.InstanceConfigs.InstanceConfig...)
		if len(ret) >= part.TotalItems || len(part.InstanceConfigs.InstanceConfig) == 0 {
			break
		}
		pageNumber++
	}
	return ret, nil
}

type SGtmAddressPool struct {
	AddressPoolId string
	Addresses     struct {
		Address []struct {
			Address string
		}
	}
}

func (self *SAliyunClient) DescribeCloudGtmAddressPool(id string) (*SGtmAddressPool, error) {
	params := map[string]string{"AddressPoolId": id}
	resp, err := self.alidnsRequest("DescribeCloudGtmAddressPool", params)
	if err != nil {
		return nil, err
	}
	ret := &SGtmAddressPool{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
