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

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SKafka) GetTopics() ([]cloudprovider.SKafkaTopic, error) {
	result := []cloudprovider.SKafkaTopic{}
	for {
		part, total, err := self.region.GetKafkaTopics(self.InstanceId, 50, len(result))
		if err != nil {
			return nil, errors.Wrapf(err, "GetKafkaTopics")
		}
		result = append(result, part...)
		if len(result) >= total {
			break
		}
	}
	return result, nil
}

func (self *SRegion) GetKafkaTopics(insId string, limit, offset int) ([]cloudprovider.SKafkaTopic, int, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"InstanceId": insId,
		"Limit":      fmt.Sprintf("%d", limit),
		"Offset":     fmt.Sprintf("%d", offset),
	}
	resp, err := self.kafkaRequest("DescribeTopic", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeTopic")
	}
	result := struct {
		Result struct {
			TopicList []struct {
				TopicId   string
				TopicName string
				Note      string
			}
			TotalCount float64
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	ret := []cloudprovider.SKafkaTopic{}
	for _, t := range result.Result.TopicList {
		ret = append(ret, cloudprovider.SKafkaTopic{
			Id:          t.TopicId,
			Name:        t.TopicName,
			Description: t.Note,
		})
	}
	return ret, int(result.Result.TotalCount), nil
}
