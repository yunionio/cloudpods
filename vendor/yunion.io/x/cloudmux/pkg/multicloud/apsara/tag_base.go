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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ApsaraTags struct {
	Tags struct {
		Tag []multicloud.STag
	}
}

func (self *ApsaraTags) GetTags() (map[string]string, error) {
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

func (self *ApsaraTags) GetSysTags() map[string]string {
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

func (self *ApsaraTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
