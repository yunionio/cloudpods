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

package multicloud

import (
	"strings"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type STagSet struct {
	TagSet []STag
	//Redis
	InstanceTags []STag
}

func (self STagSet) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.TagSet {
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.InstanceTags {
		ret[tag.TagKey] = tag.TagValue
	}
	return ret, nil
}

type STag struct {
	TagKey   string
	TagValue string

	Key   string
	Value string
}

type STags struct {
	Tags struct {
		Tag []STag
	}
}

func (self *STags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasSuffix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") {
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

func (self *STags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasPrefix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			} else if len(tag.Key) > 0 {
				ret[tag.Key] = tag.Value
			}
		}
	}
	return ret
}

type SResourceBase struct {
	// Qcloud
	STagSet

	// Aliyun
	STags
}

func (self *SResourceBase) IsEmulated() bool {
	return false
}

func (self *SResourceBase) Refresh() error {
	return nil
}

func (self *SResourceBase) GetSysTags() map[string]string {
	return nil
}

func (self *SResourceBase) GetTags() (map[string]string, error) {
	tags, _ := self.STagSet.GetTags()
	if len(tags) > 0 {
		return tags, nil
	}
	tags, _ = self.STags.GetTags()
	if len(tags) > 0 {
		return tags, nil
	}
	return map[string]string{}, nil
}

func (self *SResourceBase) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
