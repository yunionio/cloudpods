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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceSpecs struct {
	Code  int
	Value bool
}

type SWafInstance struct {
	Version           string
	InstanceSpecInfos []SInstanceSpecs
	InstanceId        string
	ExpireTime        uint64
}

func (self *SRegion) DescribeInstanceSpecInfo() (*SWafInstance, error) {
	params := map[string]string{
		"RegionId": self.RegionId,
	}
	resp, err := self.wafRequest("DescribeInstanceSpecInfo", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstanceSpecInfo")
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

func (self *SRegion) DeleteInstance(id string) error {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
	}
	_, err := self.wafRequest("DeleteInstance", params)
	return errors.Wrapf(err, "DeleteInstance")
}
