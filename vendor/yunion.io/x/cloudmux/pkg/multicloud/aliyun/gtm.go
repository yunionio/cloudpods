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

	"yunion.io/x/pkg/errors"
)

type SDnsGtmInstance struct {
	InstanceId string
	Config     struct {
		PublicZoneName string
	}
}

func (self *SAliyunClient) DescribeDnsGtmInstances() ([]SDnsGtmInstance, error) {
	params := map[string]string{
		"PageSize": "100",
	}
	ret := []SDnsGtmInstance{}
	pageNumber := 1
	for {
		params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
		resp, err := self.alidnsRequest("DescribeDnsGtmInstances", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			GtmInstances []SDnsGtmInstance
			TotalItems   int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.GtmInstances...)
		if len(ret) >= part.TotalItems || len(part.GtmInstances) == 0 {
			break
		}
		pageNumber++
	}
	return ret, nil
}

type SDnsGtmInstanceAddressPool struct {
	Name       string
	AddrPoolId string
	AddrCount  int
	Addrs      struct {
		Addr []struct {
			Addr string
		}
	}
}

func (self *SAliyunClient) DescribeDnsGtmInstanceAddressPools(id string) ([]SDnsGtmInstanceAddressPool, error) {
	params := map[string]string{"InstanceId": id, "PageSize": "100"}
	ret := []SDnsGtmInstanceAddressPool{}
	pageNumber := 1
	for {
		params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
		resp, err := self.alidnsRequest("DescribeDnsGtmInstanceAddressPools", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			AddrPools struct {
				AddrPool []SDnsGtmInstanceAddressPool
			}
			TotalItems int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.AddrPools.AddrPool...)
		if len(ret) >= part.TotalItems || len(part.AddrPools.AddrPool) == 0 {
			break
		}
		pageNumber++
	}
	return ret, nil
}

func (self *SAliyunClient) DescribeDnsGtmInstanceAddressPool(id string) (*SDnsGtmInstanceAddressPool, error) {
	params := map[string]string{"AddrPoolId": id}
	resp, err := self.alidnsRequest("DescribeDnsGtmInstanceAddressPool", params)
	if err != nil {
		return nil, err
	}
	ret := &SDnsGtmInstanceAddressPool{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}
