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

package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SResourceGroup struct {
	multicloud.SResourceBase

	Status      string
	AccountId   string
	DisplayName string
	Id          string
	CreateDate  time.Time
	Name        string
}

func (self *SResourceGroup) GetGlobalId() string {
	return self.Id
}

func (self *SResourceGroup) GetId() string {
	return self.Id
}

func (self *SResourceGroup) GetName() string {
	return self.Name
}

func (self *SResourceGroup) GetStatus() string {
	return ""
}

func (self *SAliyunClient) GetResourceGroups(pageNumber int, pageSize int) ([]SResourceGroup, int, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	if pageNumber <= 0 {
		pageNumber = 1
	}
	params := map[string]string{
		"PageNumber": fmt.Sprintf("%d", pageNumber),
		"PageSize":   fmt.Sprintf("%d", pageSize),
	}
	resp, err := self.rmRequest("ListResourceGroups", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "rmRequest.ListResourceGroups")
	}
	groups := []SResourceGroup{}
	err = resp.Unmarshal(&groups, "ResourceGroups", "ResourceGroup")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalCount")
	return groups, int(total), nil
}
