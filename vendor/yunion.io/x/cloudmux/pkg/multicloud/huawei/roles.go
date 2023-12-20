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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SRole struct {
	DomainId      string
	Flag          string
	DescriptionCn string
	Catelog       string
	Description   string
	Id            string
	DisplayName   string
	Type          string
	UpdatedTime   string
	CreatedTime   string
	Links         SLink
	Policy        jsonutils.JSONDict
}

func (role *SRole) GetName() string {
	return role.DisplayName
}

func (role *SRole) GetDescription() string {
	return role.DescriptionCn
}

func (role *SRole) GetPolicyType() string {
	return "System"
}

func (role *SRole) GetGlobalId() string {
	return role.Id
}

func (role *SRole) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (role *SRole) GetDocument() (*jsonutils.JSONDict, error) {
	return &role.Policy, nil
}

func (role *SRole) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SHuaweiClient) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := self.GetRoles("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListPermissions
func (self *SHuaweiClient) GetRoles(domainId, name string) ([]SRole, error) {
	params := url.Values{}
	if len(domainId) > 0 {
		params.Set("domain_id", self.ownerId)
	}
	if len(name) > 0 {
		params.Set("name", name)
	}

	query := url.Values{}
	query.Set("per_page", "300")
	page := 1
	query.Set("page", fmt.Sprintf("%d", page))
	ret := []SRole{}
	for {
		resp, err := self.list(SERVICE_IAM_V3, "", "roles", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Roles       []SRole
			TotalNumber int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Roles...)
		if len(ret) >= part.TotalNumber || len(part.Roles) == 0 {
			break
		}
		page++
		query.Set("page", fmt.Sprintf("%d", page))
	}
	return ret, nil
}
