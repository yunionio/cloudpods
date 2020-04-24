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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-em/zh-cn_topic_0121230880.html
type SEnterpriceProject struct {
	multicloud.SResourceBase

	Id          string
	Name        string
	Description string
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (self *SHuaweiClient) GetEnterpriceProjects() ([]SEnterpriceProject, error) {
	projects := []SEnterpriceProject{}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	err = doListAllWithOffset(client.EnterpriceProjects.List, map[string]string{}, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithOffset")
	}
	return projects, nil
}
