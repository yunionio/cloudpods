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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

func (self *SRegion) DescribeWafInstance() (*SWafInstance, error) {
	params := map[string]string{
		"RegionId": self.RegionId,
	}
	resp, err := self.wafv2Request("DescribeInstance", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstance")
	}
	ret := &SWafInstance{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if len(ret.InstanceId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return ret, nil
}

func (self *SRegion) GetICloudWafInstancesV2() ([]cloudprovider.ICloudWafInstance, error) {
	ins, err := self.DescribeWafInstance()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return []cloudprovider.ICloudWafInstance{}, nil
		}
		return nil, errors.Wrapf(err, "DescribeInstanceSpecInfo")
	}
	domains, err := self.DescribeWafDomains(ins.InstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomainNames")
	}
	ret := []cloudprovider.ICloudWafInstance{}
	for i := range domains {
		domain, err := self.DescribeDomainV2(ins.InstanceId, domains[i].Domain)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeDomain %s", domains[i].Domain)
		}
		domain.region = self
		domain.insId = ins.InstanceId
		ret = append(ret, domain)
	}
	return ret, nil
}
