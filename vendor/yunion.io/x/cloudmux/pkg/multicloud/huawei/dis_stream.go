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

type SDisStream struct {
	StreamId   string
	StreamName string
}

func (self *SRegion) ListDisStreams() ([]SDisStream, error) {
	query := url.Values{}
	query.Set("limit", "100")
	ret := []SDisStream{}
	for {
		resp, err := self.list(SERVICE_DIS, "streams", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			StreamInfoList []SDisStream
			TotalNumber    int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.StreamInfoList...)
		if len(ret) >= part.TotalNumber || len(part.StreamInfoList) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}
