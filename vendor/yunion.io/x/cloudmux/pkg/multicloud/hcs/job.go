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

package hcs

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	TASK_SUCCESS = "SUCCESS"
	TASK_FAIL    = "FAIL"
)

type SJob struct {
	Status   string `json:"status"`
	Entities struct {
		SubJobsJotal int
		ImageId      string
		ServerId     string
		SubJobs      []SJob
	} `json:"entities"`
	JobId      string `json:"job_id"`
	JobType    string `json:"job_type"`
	BeginTime  string `json:"begin_time"`
	EndTime    string `json:"end_time"`
	ErrorCode  string `json:"error_code"`
	FailReason string `json:"fail_reason"`
}

func (self *SJob) GetIds() []string {
	ret := []string{}
	if len(self.Entities.ImageId) > 0 {
		ret = append(ret, self.Entities.ImageId)
	}
	if len(self.Entities.ServerId) > 0 {
		ret = append(ret, self.Entities.ServerId)
	}
	for _, sub := range self.Entities.SubJobs {
		if len(sub.Entities.ServerId) > 0 {
			ret = append(ret, sub.Entities.ServerId)
		}
		if len(sub.Entities.ImageId) > 0 {
			ret = append(ret, sub.Entities.ImageId)
		}
	}
	return ret
}

func (self *SHcsClient) waitJobSuccess(serviceType, regionId string, jobId string, interval time.Duration, timeout time.Duration) (*SJob, error) {
	start := time.Now()
	var job *SJob
	var err error
	for time.Now().Sub(start) < timeout {
		job, err = self.GetJob(serviceType, regionId, jobId)
		if err != nil {
			return nil, err
		}
		log.Infof("wait %s job %s(%s) status: %s", serviceType, job.JobId, job.JobType, job.Status)
		if job.Status == TASK_SUCCESS {
			return job, nil
		}
		reason := []string{job.FailReason}
		for _, subJob := range job.Entities.SubJobs {
			if len(subJob.FailReason) > 0 {
				reason = append(reason, subJob.FailReason)
			}
		}
		if job.Status == TASK_FAIL {
			return nil, fmt.Errorf("job %s failed reason %s", jobId, strings.Join(reason, ";"))
		}
		time.Sleep(interval)
	}
	return nil, errors.Wrapf(cloudprovider.ErrTimeout, "wait job %s", jsonutils.Marshal(job))
}

func (self *SHcsClient) GetJob(service, regionId string, jobId string) (*SJob, error) {
	resp, err := self._getJob(service, regionId, jobId)
	if err != nil {
		return nil, err
	}
	ret := &SJob{}
	return ret, resp.Unmarshal(ret)

}

func (self *SRegion) GetJob(service string, jobId string) (*SJob, error) {
	return self.client.GetJob(service, self.Id, jobId)
}
