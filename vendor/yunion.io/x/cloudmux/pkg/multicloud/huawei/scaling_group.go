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

package huawei

import (
	"fmt"
	"net/url"
)

type SScalingGroup struct {
	ScalingGroupId   string
	ScalingGroupName string
}

func (self *SRegion) ListScalingGroups() ([]SScalingGroup, error) {
	query := url.Values{}
	query.Set("limit", "100")
	ret := []SScalingGroup{}
	for {
		resp, err := self.list(SERVICE_AS, "scaling_group", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			ScalingGroups []SScalingGroup
			TotalNubmer   int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ScalingGroups...)
		if len(ret) >= part.TotalNubmer || len(part.ScalingGroups) == 0 {
			break
		}
		query.Set("start_number", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

type SScalingInstance struct {
	InstanceId   string
	InstanceName string
}

func (self *SRegion) ListScalingInstances(groupId string) ([]SScalingInstance, error) {
	query := url.Values{}
	query.Set("limit", "100")
	res := fmt.Sprintf("scaling_group_instance/%s/list", groupId)
	ret := []SScalingInstance{}
	for {
		resp, err := self.list(SERVICE_AS, res, query)
		if err != nil {
			return nil, err
		}
		part := struct {
			ScalingGroupInstances []SScalingInstance
			TotalNubmer           int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ScalingGroupInstances...)
		if len(ret) >= part.TotalNubmer || len(part.ScalingGroupInstances) == 0 {
			break
		}
		query.Set("start_number", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}
