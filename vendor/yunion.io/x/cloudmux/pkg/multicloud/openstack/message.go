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

package openstack

import "net/url"

type SMessage struct {
	Id           string
	MessageLevel string
	EventId      string
	ResourceType string
	UserMessage  string
}

func (region *SRegion) GetMessages(resourceId string) ([]SMessage, error) {
	messages := []SMessage{}
	resource := "messages"
	query := url.Values{}
	if len(resourceId) > 0 {
		query.Set("resource_uuid", resourceId)
	}
	resp, err := region.bsList(resource, query)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(&messages, "messages")
	if err != nil {
		return nil, err
	}
	return messages, nil
}
