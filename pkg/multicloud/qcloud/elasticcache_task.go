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

package qcloud

import "yunion.io/x/pkg/errors"

type SElasticcacheTask struct {
	Status      string `json:"Status"`
	StartTime   string `json:"StartTime"`
	TaskType    string `json:"TaskType"`
	InstanceID  string `json:"InstanceId"`
	TaskMessage string `json:"TaskMessage"`
	RequestID   string `json:"RequestId"`
}

// https://cloud.tencent.com/document/product/239/30601
func (self *SRegion) DescribeTaskInfo(taskId string) (*SElasticcacheTask, error) {
	params := map[string]string{}
	params["TaskId"] = taskId
	resp, err := self.redisRequest("DescribeTaskInfo", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeTaskInfo")
	}

	ret := &SElasticcacheTask{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	return ret, nil
}
