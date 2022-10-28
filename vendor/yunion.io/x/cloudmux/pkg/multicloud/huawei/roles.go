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

func (self *SHuaweiClient) GetRoles(domainId, name string) ([]SRole, error) {
	params := map[string]string{}
	if len(domainId) > 0 {
		params["domain_id"] = self.ownerId
	}
	if len(name) > 0 {
		params["name"] = name
	}

	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}

	roles := []SRole{}
	err = doListAllWithNextLink(client.Roles.List, params, &roles)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithOffset")
	}
	return roles, nil
}
