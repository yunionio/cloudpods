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

package aliyun

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type AliyunTags struct {
	Tags struct {
		Tag []multicloud.STag

		// Kafka
		TagVO []multicloud.STag `json:"TagVO" yunion-deprecated-by:"Tag"`
	}
}

var sysTags = []string{"aliyun", "acs:", "ack.aliyun.com", "k8s.io"}

func (self *AliyunTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if tag.IsSysTagPrefix(sysTags) {
			continue
		}
		if len(tag.TagKey) > 0 {
			ret[tag.TagKey] = tag.TagValue
		} else if len(tag.Key) > 0 {
			ret[tag.Key] = tag.Value
		}
	}

	return ret, nil
}

func (self *AliyunTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if tag.IsSysTagPrefix(sysTags) {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			} else if len(tag.Key) > 0 {
				ret[tag.Key] = tag.Value
			}
		}
	}
	return ret
}

func (self *AliyunTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type SAliyunTag struct {
	ResourceId   string
	ResourceType string
	TagKey       string
	TagValue     string
}
