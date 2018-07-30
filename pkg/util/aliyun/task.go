package aliyun

import (
	"fmt"
	"strings"
	"time"
	"github.com/yunionio/log"
)

type TaskActionType string

const (
	ImportImageTask = TaskActionType("ImportImage")
	ExportImageTask = TaskActionType("ExportImage")
)

type STask struct {
	TaskId        string
	TaskStatus    string
	TaskAction    string
	SupportCancel bool
	FinishedTime  time.Time
	CreationTime  time.Time
}

func (self *SRegion) waitTaskStatus(action TaskActionType, taskId string, targetStatus string, interval time.Duration, timeout time.Duration) error {
	start := time.Now()
	for time.Now().Sub(start) < timeout {
		status, err := self.GetTaskStatus(action, taskId)
		if err != nil {
			return err
		}
		if status == targetStatus {
			break
		} else {
			time.Sleep(interval)
		}
	}
	return nil
}

func (self *SRegion) GetTaskStatus(action TaskActionType, taskId string) (string, error) {
	tasks, _, err := self.GetTasks(action, []string{taskId}, 0, 1)
	if err != nil {
		return "", err
	}
	return tasks[0].TaskStatus, nil
}

func (self *SRegion) GetTasks(action TaskActionType, taskId []string, offset int, limit int) ([]STask, int, error) {
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
