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
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SLink struct {
	Next     string
	Previous string
	Self     string
}

type SClouduser struct {
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneDeleteUser
func (self *SHuaweiClient) DeleteClouduser(id string) error {
	_, err := self.delete(SERVICE_IAM_V3, "", "users/"+id)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListGroupsForUser
func (self *SHuaweiClient) ListUserGroups(userId string) ([]SCloudgroup, error) {
	resp, err := self.list(SERVICE_IAM_V3, "", fmt.Sprintf("users/%s/groups", userId), nil)
	if err != nil {
		return nil, err
	}
	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "groups")
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListUsers
func (self *SHuaweiClient) GetCloudusers(name string) ([]SClouduser, error) {
	params := url.Values{}
	if len(name) > 0 {
		params.Set("name", name)
	}
	resp, err := self.list(SERVICE_IAM_V3, "", "users", params)
	if err != nil {
		return nil, err
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "users")
	if err != nil {
		return nil, err
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=CreateUser
func (self *SHuaweiClient) CreateClouduser(name, password, desc string) (*SClouduser, error) {
	params := map[string]interface{}{
		"name":      name,
		"domain_id": self.ownerId,
	}
	if len(password) > 0 {
		params["password"] = password
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	resp, err := self.post(SERVICE_IAM, "", "OS-USER/users", map[string]interface{}{"user": params})
	if err != nil {
		if strings.Contains(err.Error(), "1101") {
			return nil, errors.Wrap(err, `IAM user name. The length is between 5 and 32. The first digit is not a number. Special characters can only contain the '_' '-' or ' '`) //https://support.huaweicloud.com/api-iam/iam_08_0015.html
		}
		return nil, err
	}
	user := &SClouduser{client: self}
	err = resp.Unmarshal(user, "user")
	if err != nil {
		return nil, err
	}
	return user, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=UpdateUser
func (self *SHuaweiClient) ResetClouduserPassword(id, password string) error {
	_, err := self.put(SERVICE_IAM, "", "OS-USER/users/"+id, map[string]interface{}{
		"user": map[string]interface{}{
			"password": password,
		},
	})
	return err
}

type SAccessKey struct {
	client *SHuaweiClient

	AccessKey   string    `json:"access"`
	Secret      string    `json:"secret"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"create_time"`
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=ListPermanentAccessKeys
func (self *SHuaweiClient) GetAKSK(id string) ([]cloudprovider.SAccessKey, error) {
	query := url.Values{}
	query.Set("user_id", id)
	obj, err := self.list(SERVICE_IAM, "", "OS-CREDENTIAL/credentials", query)
	if err != nil {
		return nil, errors.Wrap(err, "list credential")
	}
	aks := make([]SAccessKey, 0)
	obj.Unmarshal(&aks, "credentials")
	res := make([]cloudprovider.SAccessKey, len(aks))
	for i := 0; i < len(aks); i++ {
		res[i].Name = aks[i].Description
		res[i].AccessKey = aks[i].AccessKey
		res[i].Secret = aks[i].Secret
		res[i].Status = aks[i].Status
		res[i].CreatedAt = aks[i].CreatedAt
	}
	return res, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=CreatePermanentAccessKey
func (self *SHuaweiClient) CreateAKSK(id, name string) (*cloudprovider.SAccessKey, error) {
	params := map[string]interface{}{
		"credential": map[string]interface{}{
			"user_id":     id,
			"description": name,
		},
	}
	obj, err := self.post(SERVICE_IAM, "", "OS-CREDENTIAL/credentials", params)
	if err != nil {
		return nil, errors.Wrap(err, "SHuaweiClient.createAKSK")
	}
	ak := SAccessKey{}
	err = obj.Unmarshal(&ak, "credential")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	res := cloudprovider.SAccessKey{
		Name:      ak.Description,
		AccessKey: ak.AccessKey,
		Secret:    ak.Secret,
	}
	return &res, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=DeletePermanentAccessKey
func (self *SHuaweiClient) DeleteAKSK(accessKey string) error {
	_, err := self.delete(SERVICE_IAM, "", "OS-CREDENTIAL/credentials/"+accessKey)
	return err
}

func (user *SClouduser) DeleteAccessKey(accessKey string) error {
	return user.client.DeleteAKSK(accessKey)
}

func (user *SClouduser) CreateAccessKey(name string) (*cloudprovider.SAccessKey, error) {
	return user.client.CreateAKSK(user.Id, name)
}

func (user *SClouduser) GetAccessKeys() ([]cloudprovider.SAccessKey, error) {
	return user.client.GetAKSK(user.Id)
}
