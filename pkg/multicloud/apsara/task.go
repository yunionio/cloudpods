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
	"strings"
	"time"

	"yunion.io/x/log"
)

type TaskStatusType string

type TaskActionType string

const (
	ImportImageTask = TaskActionType("ImportImage")
	ExportImageTask = TaskActionType("ExportImage")

	// Finished：已完成
	// Processing：运行中
	// Waiting：多任务排队中
	// Deleted：已取消
	// Paused：暂停
	// Failed：失败
	TaskStatusFinished   = TaskStatusType("Finished")
	TaskStatusProcessing = TaskStatusType("Processing")
	TaskStatusWaiting    = TaskStatusType("Waiting")
	TaskStatusDeleted    = TaskStatusType("Deleted")
	TaskStatusPaused     = TaskStatusType("Paused")
	TaskStatusFailed     = TaskStatusType("Failed")
)

type STask struct {
	TaskId        string
	TaskStatus    TaskStatusType
	TaskAction    string
	SupportCancel bool
	FinishedTime  time.Time
	CreationTime  time.Time
}

func (self *SRegion) waitTaskStatus(action TaskActionType, taskId string, targetStatus TaskStatusType, interval time.Duration, timeout time.Duration) error {
	start := time.Now()
	for time.Now().Sub(start) < timeout {
		status, err := self.GetTaskStatus(action, taskId)
		if err != nil {
			return err
		}
		if status == targetStatus {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timeout for waitting task %s(%s) after %f minutes", taskId, action, timeout.Minutes())
}

func (self *SRegion) GetTaskStatus(action TaskActionType, taskId string) (TaskStatusType, error) {
	task, err := self.GetTask(taskId)
	if err != nil {
		return "", err
	}
	return task.TaskStatus, nil
}

func (self *SRegion) GetTasks(action TaskActionType, taskId []string, taskStatus TaskStatusType, offset int, limit int) ([]STask, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}

	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	params["TaskAction"] = string(action)
	if taskId != nil && len(taskId) > 0 {
		params["TaskIds"] = strings.Join(taskId, ",")
	}
	if len(taskStatus) > 0 {
		params["TaskStatus"] = string(taskStatus)
	}

	body, err := self.ecsRequest("DescribeTasks", params)
	if err != nil {
		log.Errorf("GetTasks fail %s", err)
		return nil, 0, err
	}

	log.Infof("%s", body)
	tasks := make([]STask, 0)
	err = body.Unmarshal(&tasks, "TaskSet", "Task")
	if err != nil {
		log.Errorf("Unmarshal task fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return tasks, int(total), nil
}

type STaskError struct {
	ErrorCode       string
	ErrorMsg        string
	OperationStatus string
}

type STaskDetail struct {
	CreationTime         time.Time
	FailedCount          int
	FinishedTime         time.Time
	RegionId             string
	OperationProgressSet map[string][]STaskError
	RequestId            string
	SuccessCount         int
	SupportCancel        bool
	TaskAction           string
	TaskId               string
	TaskProcess          string
	TaskStatus           TaskStatusType
	TotalCount           int
}

func (self *SRegion) GetTask(taskId string) (*STaskDetail, error) {
	params := map[string]string{
		"RegionId": self.RegionId,
		"TaskId":   taskId,
	}
	body, err := self.ecsRequest("DescribeTaskAttribute", params)
	if err != nil {
		return nil, err
	}
	log.Infof("%s", body)
	detail := &STaskDetail{}
	return detail, body.Unmarshal(detail)
}

func (self *SRegion) CancelTask(taskId string) error {
	params := map[string]string{
		"RegionId": self.RegionId,
		"TaskId":   taskId,
	}
	_, err := self.ecsRequest("CancelTask", params)
	return err
}

func (region *SRegion) CancelImageImportTasks() error {
	tasks, _, _ := region.GetTasks(ImportImageTask, []string{}, TaskStatusProcessing, 0, 50)
	for i := 0; i < len(tasks); i++ {
		task, err := region.GetTask(tasks[i].TaskId)
		if err != nil {
			log.Errorf("failed get task %s %s error: %v", tasks[i].CreationTime, tasks[i].TaskId, err)
		}
		if task != nil {
			log.Debugf("task info: %s(%s) cancelable %t process %s", task.TaskId, task.CreationTime, task.SupportCancel, task.TaskProcess)
		} else {
			log.Debugf("task info: %s(%s) cancelable %t", tasks[i].TaskId, tasks[i].CreationTime, tasks[i].SupportCancel)
		}
		if tasks[i].SupportCancel {
			err := region.CancelTask(tasks[i].TaskId)
			if err != nil {
				return fmt.Errorf("failed to cancel task %s error: %v", tasks[i].TaskId, err)
			}
		}
	}
	return nil
}
