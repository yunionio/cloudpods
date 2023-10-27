// Copyright 2023 Yunion
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

package volcengine

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type VolcEngineTags struct {
	Tags []multicloud.STag
}

func (itag *VolcEngineTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range itag.Tags {
		if len(tag.TagKey) > 0 {
			ret[tag.TagKey] = tag.TagValue
		} else if len(tag.Key) > 0 {
			ret[tag.Key] = tag.Value
		}
	}
	return ret, nil
}

func (itag *VolcEngineTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	prefix := "volc:"
	for _, tag := range itag.Tags {
		if len(tag.TagKey) > 0 {
			if strings.HasPrefix(tag.TagKey, prefix) {
				ret[tag.TagKey] = tag.TagValue
			}
		}
		if len(tag.Key) > 0 {
			if strings.HasPrefix(tag.Key, prefix) {
				ret[tag.Key] = tag.Value
			}
		}
	}
	return ret
}

func (itag *VolcEngineTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
