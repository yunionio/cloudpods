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
	"time"

	"yunion.io/x/pkg/errors"
)

func (self *SRegion) waitTaskStatus(serviceType string, taskId string, targetStatus string, interval time.Duration, timeout time.Duration) error {
	start := time.Now()
	for time.Now().Sub(start) < timeout {
		status, err := self.GetTaskStatus(serviceType, taskId)
		if err != nil {
			return err
		}
		if status == targetStatus {
			break
		} else if status == TASK_FAIL {
			return fmt.Errorf("task %s failed", taskId)
		} else {
			time.Sleep(interval)
		}
	}
	return nil
}

type SJob struct {
	Status   string
	Entities struct {
		SubJobs []struct {
			VolumeId string
			ServerId string
		}
		VolumeId string
		ImageId  string
	}
}

func (self *SRegion) GetJob(serviceType string, jobId string) (*SJob, error) {
	resp, err := self.list(serviceType, "jobs/"+jobId, nil)
	if err != nil {
		return nil, err
	}
	ret := &SJob{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) GetTaskStatus(serviceType string, taskId string) (string, error) {
	job, err := self.GetJob(serviceType, taskId)
	if err != nil {
		return "", err
	}
	return job.Status, nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0022225398.html
// 数据结构  entities -> []job
func (self *SRegion) GetAllSubTaskEntityIDs(serviceType string, taskId string) ([]string, error) {
	err := self.waitTaskStatus(serviceType, taskId, TASK_SUCCESS, 10*time.Second, 600*time.Second)
	if err != nil {
		return nil, err
	}
	job, err := self.GetJob(serviceType, taskId)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for _, entity := range job.Entities.SubJobs {
		if len(entity.VolumeId) > 0 {
			ret = append(ret, entity.VolumeId)
		}
		if len(entity.ServerId) > 0 {
			ret = append(ret, entity.ServerId)
		}
	}
	return ret, nil
}

// 数据结构  entities -> job
func (self *SRegion) GetTaskEntityID(serviceType string, taskId string, key string) (string, error) {
	err := self.waitTaskStatus(serviceType, taskId, TASK_SUCCESS, 10*time.Second, 600*time.Second)
	if err != nil {
		return "", err
	}

	job, err := self.GetJob(serviceType, taskId)
	if err != nil {
		return "", err
	}
	switch key {
	case "volume_id":
		return job.Entities.VolumeId, nil
	case "image_id":
		return job.Entities.ImageId, nil
	default:
		return "", fmt.Errorf("unknown %s", key)
	}
}
