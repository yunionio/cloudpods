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

package huawei

import (
	"fmt"
	"net/url"
)

type SLtsGroup struct {
	LogGroupId   string
	LogGroupName string
}

func (self *SRegion) ListLtsGroups() ([]SLtsGroup, error) {
	query := url.Values{}
	resp, err := self.list(SERVICE_LTS, "groups", query)
	if err != nil {
		return nil, err
	}
	ret := []SLtsGroup{}
	return ret, resp.Unmarshal(&ret, "log_groups")
}

type SLtsStream struct {
	LogStreamId   string
	LogStreamName string
}

func (self *SRegion) ListLtsStreams() ([]SLtsStream, error) {
	query := url.Values{}
	query.Set("limit", "100")
	ret := []SLtsStream{}
	for {
		resp, err := self.list(SERVICE_LTS, "log-streams", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			LogStreams []SLtsStream
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.LogStreams...)
		if len(part.LogStreams) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil

}

func (self *SRegion) ListLtsStreamsByGroup(groupId string) ([]SLtsStream, error) {
	query := url.Values{}
	res := fmt.Sprintf("groups/%s/streams", groupId)
	resp, err := self.list(SERVICE_LTS, res, query)
	if err != nil {
		return nil, err
	}
	ret := []SLtsStream{}
	return ret, resp.Unmarshal(&ret, "log_streams")
}
