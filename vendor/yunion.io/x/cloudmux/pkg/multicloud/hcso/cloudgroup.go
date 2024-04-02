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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCloudgroup struct {
	client      *SHuaweiClient
	Name        string
	Description string
	Id          string
	CreateTime  string
}

func (group *SCloudgroup) GetName() string {
	return group.Name
}

func (group *SCloudgroup) GetDescription() string {
	return group.Description
}

func (group *SCloudgroup) GetGlobalId() string {
	return group.Id
}

func (group *SCloudgroup) Delete() error {
	return group.client.DeleteGroup(group.Id)
}

func (group *SCloudgroup) AddUser(name string) error {
	user, err := group.client.GetIClouduserByName(name)
	if err != nil {
		return errors.Wrap(err, "GetIClouduserByName")
	}
	return group.client.AddUserToGroup(group.Id, user.GetGlobalId())
}

func (group *SCloudgroup) RemoveUser(name string) error {
	user, err := group.client.GetIClouduserByName(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetIClouduserByName(%s)", name)
	}
	return group.client.RemoveUserFromGroup(group.Id, user.GetGlobalId())
}

func (group *SCloudgroup) DetachPolicy(roleId string, policyType api.TPolicyType) error {
	if policyType == api.PolicyTypeCustom {
		return group.client.DetachGroupCustomRole(group.Id, roleId)
	}
	return group.client.DetachGroupRole(group.Id, roleId)
}

func (group *SCloudgroup) AttachPolicy(roleId string, policyType api.TPolicyType) error {
	if policyType == api.PolicyTypeCustom {
		return group.client.AttachGroupCustomRole(group.Id, roleId)
	}
	return group.client.AttachGroupRole(group.Id, roleId)
}

func (group *SCloudgroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := group.client.GetGroupRoles(group.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetGroupRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		_, err := group.client.GetRole(roles[i].GetName())
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				continue
			}
			return nil, errors.Wrapf(err, "GetRole(%s)", roles[i].GetName())
		}
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (group *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := group.client.GetGroupUsers(group.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = group.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SHuaweiClient) GetGroups(domainId, name string) ([]SCloudgroup, error) {
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

	groups := []SCloudgroup{}
	err = doListAllWithNextLink(client.Groups.List, params, &groups)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithOffset")
	}
	return groups, nil
}

func (self *SHuaweiClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := self.GetGroups("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetGroup")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		if groups[i].Name != "admin" {
			groups[i].client = self
			ret = append(ret, &groups[i])
		}
	}
	return ret, nil
}

func (self *SHuaweiClient) GetGroupUsers(groupId string) ([]SClouduser, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}

	resp, err := client.Groups.ListInContextWithSpec(nil, fmt.Sprintf("%s/users", groupId), nil, "users")
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	users := []SClouduser{}
	err = jsonutils.Update(&users, resp.Data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return users, nil
}

func (self *SHuaweiClient) GetGroupRoles(groupId string) ([]SRole, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	resp, err := client.Domains.ListRoles(self.ownerId, groupId)
	if err != nil {
		return nil, errors.Wrap(err, "ListRoles")
	}
	roles := []SRole{}
	err = jsonutils.Update(&roles, resp.Data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	return roles, nil
}

func (self *SHuaweiClient) CreateGroup(name, desc string) (*SCloudgroup, error) {
	params := map[string]string{
		"name": name,
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}

	group := SCloudgroup{client: self}
	err = DoCreate(client.Groups.Create, jsonutils.Marshal(map[string]interface{}{"group": params}), &group)
	if err != nil {
		return nil, errors.Wrap(err, "DoCreate")
	}
	return &group, nil
}

func (self *SHuaweiClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	return group, nil
}

func (self *SHuaweiClient) DeleteGroup(id string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	return DoDeleteWithSpec(client.Groups.DeleteInContextWithSpec, nil, id, "", nil, nil)
}

func (self *SHuaweiClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	groups, err := self.GetGroups(self.ownerId, name)
	if err != nil {
		return nil, errors.Wrap(err, "GetGroups")
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

func (self *SHuaweiClient) AddUserToGroup(groupId, userId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	_, err = client.Groups.UpdateInContextWithSpec(nil, groupId, fmt.Sprintf("users/%s", userId), nil, "")
	return err
}

func (self *SHuaweiClient) RemoveUserFromGroup(groupId, userId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	_, err = client.Groups.DeleteInContextWithSpec(nil, groupId, fmt.Sprintf("users/%s", userId), nil, nil, "")
	return err
}

func (self *SHuaweiClient) DetachGroupRole(groupId, roleId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	role, err := self.GetRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err = client.Domains.DeleteRole(self.ownerId, groupId, role.Id)
		if err != nil {
			return errors.Wrapf(err, "DeleteRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err = client.Projects.DeleteProjectRole(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "DeleteProjectRole")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err = client.Projects.DeleteProjectRole(project.ID, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "DeleteProjectRole")
			}
		}
	}
	return nil
}

func (self *SHuaweiClient) DetachGroupCustomRole(groupId, roleId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	role, err := self.GetCustomRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetCustomRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err = client.Domains.DeleteRole(self.ownerId, groupId, role.Id)
		if err != nil {
			return errors.Wrapf(err, "DeleteRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err = client.Projects.DeleteProjectRole(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "DeleteProjectRole")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err = client.Projects.DeleteProjectRole(project.ID, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "DeleteProjectRole")
			}
		}
	}
	return nil
}

func (self *SHuaweiClient) GetCustomRole(name string) (*SRole, error) {
	roles, err := self.GetCustomRoles()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCustomRoles(%s)", name)
	}
	for i := range roles {
		if roles[i].DisplayName == name {
			return &roles[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (self *SHuaweiClient) GetRole(name string) (*SRole, error) {
	roles, err := self.GetRoles("", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetRoles(%s)", name)
	}
	for i := range roles {
		if roles[i].DisplayName == name {
			return &roles[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (self *SHuaweiClient) AttachGroupRole(groupId, roleId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	role, err := self.GetRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err = client.Domains.AddRole(self.ownerId, groupId, role.Id)
		if err != nil {
			return errors.Wrapf(err, "AddRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err = client.Projects.AddProjectRole(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "AddProjectRole")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err = client.Projects.AddProjectRole(project.ID, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "AddProjectRole")
			}
		}
	}
	return nil
}

func (self *SHuaweiClient) AttachGroupCustomRole(groupId, roleId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	role, err := self.GetCustomRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err = client.Domains.AddRole(self.ownerId, groupId, role.Id)
		if err != nil {
			return errors.Wrapf(err, "AddRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err = client.Projects.AddProjectRole(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "AddProjectRole")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err = client.Projects.AddProjectRole(project.ID, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "AddProjectRole")
			}
		}
	}
	return nil
}
