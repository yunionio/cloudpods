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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type KafkaListOptions struct {
		Id     string
		Offset int
		Limit  int
	}
	shellutils.R(&KafkaListOptions{}, "kafka-list", "List kafka", func(cli *qcloud.SRegion, args *KafkaListOptions) error {
		kafkas, _, err := cli.GetKafkas(args.Id, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(kafkas, 0, 0, 0, []string{})
		return nil
	})

	type KafkaIdOptions struct {
		ID string
	}

	type KafkaTopicListOptions struct {
		KafkaIdOptions
		Offset int
		Limit  int
	}

	shellutils.R(&KafkaTopicListOptions{}, "kafka-topic-list", "List kafka topic", func(cli *qcloud.SRegion, args *KafkaTopicListOptions) error {
		topics, _, err := cli.GetKafkaTopics(args.ID, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(topics, 0, 0, 0, []string{})
		return nil
	})

}
