package huawei

import (
	"fmt"
	"time"

	"yunion.io/x/log"
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

func (self *SRegion) GetTaskStatus(serviceType string, taskId string) (string, error) {
	querys := map[string]string{"service_type": serviceType}
	task, err := self.ecsClient.Jobs.Get(taskId, querys)
	if err != nil {
		return "", err
	}

	status, err := task.GetString("status")
	if status == TASK_FAIL {
		log.Debugf("task %s failed: %s", taskId, task.String())
	}

	return status, err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0022225398.html
// 数据结构  entities -> []job
func (self *SRegion) GetAllSubTaskEntityIDs(serviceType string, taskId string, entityKeyName string) ([]string, error) {
	err := self.waitTaskStatus(serviceType, taskId, TASK_SUCCESS, 10*time.Second, 600*time.Second)
	if err != nil {
		return nil, err
	}

	querys := map[string]string{"service_type": serviceType}
	ret, err := self.ecsClient.Jobs.Get(taskId, querys)
	if err != nil {
		return nil, err
	}

	entities, err := ret.GetArray("entities", "sub_jobs")
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	for i := range entities {
		entity := entities[i]
		rid, err := entity.GetString("entities", entityKeyName)
		if err != nil {
			return nil, err
		}

		ids = append(ids, rid)
	}

	return ids, nil
}

// 数据结构  entities -> job
func (self *SRegion) GetTaskEntityID(serviceType string, taskId string, entityKeyName string) (string, error) {
	err := self.waitTaskStatus(serviceType, taskId, TASK_SUCCESS, 10*time.Second, 600*time.Second)
	if err != nil {
		return "", err
	}

	querys := map[string]string{"service_type": serviceType}
	ret, err := self.ecsClient.Jobs.Get(taskId, querys)
	if err != nil {
		return "", err
	}

	return ret.GetString("entities", entityKeyName)
}
