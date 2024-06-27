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

type SApigateway struct {
	Id           string
	InstanceName string
}

func (self *SRegion) ListApigateway() ([]SApigateway, error) {
	query := url.Values{}
	query.Set("limit", "500")
	ret := []SApigateway{}
	for {
		resp, err := self.list(SERVICE_APIG, "apigw/instances", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Instances []SApigateway
			Total     int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(ret) >= part.Total || len(part.Instances) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}
