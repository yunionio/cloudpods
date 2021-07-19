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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type KafkaListOptions struct {
		Ids []string
	}
	shellutils.R(&KafkaListOptions{}, "kafka-list", "List kafka", func(cli *aliyun.SRegion, args *KafkaListOptions) error {
		kafkas, err := cli.GetKafkas(args.Ids)
		if err != nil {
			return err
		}
		printList(kafkas, 0, 0, 0, []string{})
		return nil
	})

	type KafkaIdOptions struct {
		ID string
	}

	shellutils.R(&KafkaIdOptions{}, "kafka-delete", "Delete kafka", func(cli *aliyun.SRegion, args *KafkaIdOptions) error {
		return cli.DeleteKafka(args.ID)
	})

	shellutils.R(&KafkaIdOptions{}, "kafka-release", "Release kafka", func(cli *aliyun.SRegion, args *KafkaIdOptions) error {
		return cli.ReleaseKafka(args.ID)
	})

	type KafkaTopicListOptions struct {
		KafkaIdOptions
		Page     int
		PageSize int
	}

	shellutils.R(&KafkaTopicListOptions{}, "kafka-topic-list", "List kafka topic", func(cli *aliyun.SRegion, args *KafkaTopicListOptions) error {
		topics, _, err := cli.GetKafkaTopics(args.ID, args.Page, args.PageSize)
		if err != nil {
			return err
		}
		printList(topics, 0, 0, 0, nil)
		return nil
	})

}
