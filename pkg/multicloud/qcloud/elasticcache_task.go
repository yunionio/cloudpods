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
