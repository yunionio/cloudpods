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

package zstack

import "net/url"

type SSysTag struct {
	ZStackTime
	Inherent     bool   `json:"inherent"`
	ResourceType string `json:"resourceType"`
	ResourceUUID string `json:"resourceUuid"`
	Tag          string `json:"tag"`
	Type         string `json:"type"`
	UUID         string `json:"uuid"`
}

func (region *SRegion) GetSysTags(tagId string, resourceType string, resourceId string, tag string) ([]SSysTag, error) {
	tags := []SSysTag{}
	params := url.Values{}
	if len(tagId) > 0 {
		params.Add("q", "uuid="+tagId)
	}
	if len(resourceType) > 0 {
		params.Add("q", "resourceType="+resourceType)
	}
	if len(resourceId) > 0 {
		params.Add("q", "resourceUuid="+resourceId)
	}
	if len(tag) > 0 {
		params.Add("q", "tag="+tag)
	}
	return tags, region.client.listAll("system-tags", params, &tags)
}
