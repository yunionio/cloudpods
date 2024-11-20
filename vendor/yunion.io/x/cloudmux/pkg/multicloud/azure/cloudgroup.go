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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/pinyinutils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCloudgroup struct {
	client *SAzureClient

	Id                string
	DeletionTimestamp string
	Description       string
	DirSyncEnabled    string
	DisplayName       string
	LastDirSyncTime   string
	Mail              string
	MailNickname      string
	MailEnabled       bool
	ProxyAddresses    []string
}

func (group *SCloudgroup) GetName() string {
	return group.DisplayName
}

func (group *SCloudgroup) GetGlobalId() string {
	return group.Id
}

func (group *SCloudgroup) GetDescription() string {
	return group.Description
}

func (group *SCloudgroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := group.client.GetPrincipalPolicy(group.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudpolicies(%s)", group.Id)
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &SCloudpolicy{Id: policies[i].RoleDefinitionId})
	}
	return ret, nil
}

func (group *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := group.client.ListGroupMemebers(group.Id)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupMemebers")
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = group.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (group *SCloudgroup) AddUser(name string) error {
	return group.client.AddGroupUser(group.Id, name)
}

func (group *SCloudgroup) RemoveUser(name string) error {
	return group.client.RemoveGroupUser(group.Id, name)
}

func (group *SCloudgroup) AttachPolicy(policyId string, policyType api.TPolicyType) error {
	return group.client.AssignPolicy(group.Id, policyId)
}

func (group *SCloudgroup) DetachPolicy(policyId string, policyType api.TPolicyType) error {
	policys, err := group.client.GetPrincipalPolicy(group.Id)
	if err != nil {
		return err
	}
	for _, policy := range policys {
		if policy.RoleDefinitionId == policyId {
			return group.client.DeletePrincipalPolicy(policy.Id)
		}
	}
	return nil
}

func (group *SCloudgroup) Delete() error {
	return group.client.DeleteGroup(group.Id)
}

func (self *SAzureClient) GetCloudgroups(name string) ([]SCloudgroup, error) {
	params := url.Values{}
	if len(name) > 0 {
		params.Set("$filter", fmt.Sprintf("displayName eq '%s'", name))
	}
	resp, err := self._list_v2(SERVICE_GRAPH, "groups", "", params)
	if err != nil {
		return nil, err
	}
	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "value")
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (self *SAzureClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := self.GetCloudgroups("")
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudgroups")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SAzureClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	groups, err := self.GetCloudgroups(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudgroups(%s)", name)
	}
	if len(groups) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(groups) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	groups[0].client = self
	return &groups[0], nil
}

func (self *SAzureClient) ListGroupMemebers(id string) ([]SClouduser, error) {
	resource := fmt.Sprintf("groups/%s/members", id)
	resp, err := self._list_v2(SERVICE_GRAPH, resource, "", nil)
	if err != nil {
		return nil, err
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "value")
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (self *SAzureClient) DeleteGroup(id string) error {
	_, err := self._delete_v2(SERVICE_GRAPH, "groups/"+id, "")
	return err
}

func (self *SAzureClient) CreateGroup(name, desc string) (*SCloudgroup, error) {
	params := map[string]interface{}{
		"displayName":     name,
		"mailNickname":    pinyinutils.Text2Pinyin(name),
		"mailEnabled":     false,
		"securityEnabled": true,
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	resp, err := self._post_v2(SERVICE_GRAPH, "groups", "", params)
	if err != nil {
		return nil, err
	}
	group := &SCloudgroup{client: self}
	err = resp.Unmarshal(group)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (self *SAzureClient) RemoveGroupUser(id, userName string) error {
	user, err := self.GetClouduser(userName)
	if err != nil {
		return errors.Wrapf(err, "GetCloudusers(%s)", userName)
	}
	resource := fmt.Sprintf("/groups/%s/members/%s/$ref", id, user.Id)
	_, err = self._delete_v2(SERVICE_GRAPH, resource, "")
	return err
}

func (self *SAzureClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	group.client = self
	return group, nil
}

func (self *SAzureClient) AddGroupUser(id, userName string) error {
	user, err := self.GetClouduser(userName)
	if err != nil {
		return errors.Wrapf(err, "GetCloudusers(%s)", userName)
	}
	params := map[string]interface{}{
		"@odata.id": fmt.Sprintf("https://graph.microsoft.com/v1.0/directoryObjects/%s", user.Id),
	}
	if self.envName == "AzureChinaCloud" {
		params["@odata.id"] = fmt.Sprintf("https://microsoftgraph.chinacloudapi.cn/v1.0/directoryObjects/%s", user.Id)
	}
	resource := fmt.Sprintf("groups/%s/members/$ref", id)
	_, err = self._post_v2(SERVICE_GRAPH, resource, "", params)
	if err != nil && !strings.Contains(err.Error(), "One or more added object references already exist for the following modified properties") {
		return err
	}
	return nil
}
