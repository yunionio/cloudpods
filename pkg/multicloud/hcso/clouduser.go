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

package hcso

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/multicloud/hcso/client/modules"
)

type SLink struct {
	Next     string
	Previous string
	Self     string
}

type SClouduser struct {
	multicloud.SBaseClouduser

	client *SHuaweiClient

	Description       string
	DomainId          string
	Enabled           bool
	ForceResetPwd     bool
	Id                string
	LastProjectId     string
	Links             SLink
	Name              string
	PasswordExpiresAt string
	PwdStatus         bool
}

func (user *SClouduser) GetGlobalId() string {
	return user.Id
}

func (user *SClouduser) GetName() string {
	return user.Name
}

func (user *SClouduser) GetEmailAddr() string {
	return ""
}

func (user *SClouduser) GetInviteUrl() string {
	return ""
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return []cloudprovider.ICloudpolicy{}, nil
}

func (user *SClouduser) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	return []cloudprovider.ICloudpolicy{}, nil
}

func (user *SClouduser) AttachSystemPolicy(policyType string) error {
	return cloudprovider.ErrNotSupported
}

func (user *SClouduser) AttachCustomPolicy(policyType string) error {
	return cloudprovider.ErrNotSupported
}

func (user *SClouduser) DetachSystemPolicy(policyId string) error {
	return cloudprovider.ErrNotSupported
}

func (user *SClouduser) DetachCustomPolicy(policyId string) error {
	return cloudprovider.ErrNotSupported
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.ListUserGroups(user.Id)
	if err != nil {
		return nil, errors.Wrap(err, "Users.ListGroups")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (user *SClouduser) Delete() error {
	return user.client.DeleteClouduser(user.Id)
}

func (user *SClouduser) IsConsoleLogin() bool {
	return user.Enabled == true
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.Id, password)
}

func (self *SHuaweiClient) DeleteClouduser(id string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	_, err = client.Users.Delete(id)
	return err
}

func (self *SHuaweiClient) ListUserGroups(userId string) ([]SCloudgroup, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	result, err := client.Users.ListGroups(userId)
	if err != nil {
		return nil, errors.Wrap(err, "Users.ListGroups")
	}
	groups := []SCloudgroup{}
	err = jsonutils.Update(&groups, result.Data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return groups, nil
}

func (self *SHuaweiClient) GetCloudusers(name string) ([]SClouduser, error) {
	params := map[string]string{}
	if len(name) > 0 {
		params["name"] = name
	}
	users := []SClouduser{}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	err = doListAllWithOffset(client.Users.List, params, &users)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithOffset")
	}
	return users, nil
}

func (self *SHuaweiClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.GetCloudusers("")
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudusers")
	}
	iUsers := []cloudprovider.IClouduser{}
	for i := range users {
		if users[i].Name != self.ownerName {
			users[i].client = self
			iUsers = append(iUsers, &users[i])
		}
	}
	return iUsers, nil
}

func (self *SHuaweiClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	users, err := self.GetCloudusers(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudusers(%s)", name)
	}
	if len(users) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(users) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	users[0].client = self
	return &users[0], nil
}

func (self *SHuaweiClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.CreateClouduser(conf.Name, conf.Password, conf.Desc)
}

func (self *SHuaweiClient) CreateClouduser(name, password, desc string) (*SClouduser, error) {
	params := map[string]string{
		"name":      name,
		"domain_id": self.ownerId,
	}
	if len(password) > 0 {
		params["password"] = password
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	user := SClouduser{client: self}
	err = DoCreate(client.Users.Create, jsonutils.Marshal(map[string]interface{}{"user": params}), &user)
	if err != nil {
		ce, ok := err.(*modules.HuaweiClientError)
		if ok && len(ce.Errorcode) > 0 && ce.Errorcode[0] == "1101" {
			return nil, errors.Wrap(err, `IAM user name. The length is between 5 and 32. The first digit is not a number. Special characters can only contain the '_' '-' or ' '`) //https://support.huaweicloud.com/api-iam/iam_08_0015.html
		}
		return nil, errors.Wrap(err, "DoCreate")
	}
	return &user, nil
}

func (self *SHuaweiClient) ResetClouduserPassword(id, password string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	return client.Users.ResetPassword(id, password)
}
