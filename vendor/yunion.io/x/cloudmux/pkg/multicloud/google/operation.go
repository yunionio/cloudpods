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

package google

import (
	"time"

	"yunion.io/x/log"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	OPERATION_STATUS_RUNNING = "RUNNING"
	OPERATION_STATUS_DONE    = "DONE"
)

type SOperation struct {
	Id            string
	Name          string
	OperationType string
	TargetLink    string
	TargetId      string
	Status        string
	User          string
	Progress      int
	InsertTime    time.Time
	StartTime     time.Time
	EndTime       time.Time
	SelfLink      string
	Region        string
	Kind          string
}

func (self *SGoogleClient) GetOperation(id string) (*SOperation, error) {
	operation := &SOperation{}
	err := self.GetBySelfId(id, &operation)
	if err != nil {
		return nil, err
	}
	return operation, nil
}

func (self *SGoogleClient) WaitOperation(id string, resource, action string) (string, error) {
	targetLink := ""
	err := cloudprovider.Wait(time.Second*5, time.Minute*5, func() (bool, error) {
		operation, err := self.GetOperation(id)
		if err != nil {
			return false, err
		}
		log.Debugf("%s %s operation status: %s expect %s", action, resource, operation.Status, OPERATION_STATUS_DONE)
		if operation.Status == OPERATION_STATUS_DONE {
			targetLink = operation.TargetLink
			return true, nil
		}
		return false, nil
	})
	return targetLink, err
}

func (region *SRegion) GetRdsOperation(id string) (*SOperation, error) {
	operation := &SOperation{}
	err := region.rdsGet(id, &operation)
	if err != nil {
		return nil, err
	}
	return operation, nil
}

func (region *SRegion) WaitRdsOperation(id string, resource, action string) (string, error) {
	targetLink := ""
	err := cloudprovider.Wait(time.Second*5, time.Minute*20, func() (bool, error) {
		operation, err := region.GetRdsOperation(id)
		if err != nil {
			return false, err
		}
		log.Debugf("%s %s operation status: %s expect %s", action, resource, operation.Status, OPERATION_STATUS_DONE)
		if operation.Status == OPERATION_STATUS_DONE {
			targetLink = operation.TargetLink
			return true, nil
		}
		return false, nil
	})
	return targetLink, err
}
