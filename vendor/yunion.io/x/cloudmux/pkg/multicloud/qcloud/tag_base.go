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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type QcloudTags struct {
	TagSet []multicloud.STag

	// Redis
	InstanceTags []multicloud.STag
	// Elasticsearch
	TagList []multicloud.STag
	// Kafka
	Tags []multicloud.STag
	// Cdn
	Tag []multicloud.STag
	// TDSQL
	ResourceTags []multicloud.STag
}

func (self *QcloudTags) getTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.TagSet {
		if tag.Value == "null" {
			tag.Value = ""
		}
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.InstanceTags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.TagList {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.Tags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.Tag {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.ResourceTags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	return ret, nil
}

func (self *QcloudTags) GetTags() (map[string]string, error) {
	tags, _ := self.getTags()
	for k := range tags {
		if strings.HasPrefix(k, "tencentcloud:") {
			delete(tags, k)
		}
	}
	return tags, nil
}

func (self *QcloudTags) GetSysTags() map[string]string {
	tags, _ := self.getTags()
	ret := map[string]string{}
	for k, v := range tags {
		if strings.HasPrefix(k, "tencentcloud:") {
			ret[k] = v
		}
	}
	return ret
}

func (self *QcloudTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
