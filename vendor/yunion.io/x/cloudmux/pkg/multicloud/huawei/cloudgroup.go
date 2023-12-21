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

	"yunion.io/x/pkg/errors"

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

func (group *SCloudgroup) DetachSystemPolicy(roleId string) error {
	return group.client.DetachGroupRole(group.Id, roleId)
}

func (group *SCloudgroup) DetachCustomPolicy(roleId string) error {
	return group.client.DetachGroupCustomRole(group.Id, roleId)
}

func (group *SCloudgroup) AttachSystemPolicy(roleId string) error {
	return group.client.AttachGroupRole(group.Id, roleId)
}

func (group *SCloudgroup) AttachCustomPolicy(roleId string) error {
	return group.client.AttachGroupCustomRole(group.Id, roleId)
}

func (group *SCloudgroup) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
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

func (group *SCloudgroup) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := group.client.GetGroupRoles(group.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetGroupRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		_, err := group.client.GetCustomRole(roles[i].GetName())
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListGroups
func (self *SHuaweiClient) GetGroups(domainId, name string) ([]SCloudgroup, error) {
	query := url.Values{}
	if len(domainId) > 0 {
		query.Set("domain_id", self.ownerId)
	}
	if len(name) > 0 {
		query.Set("name", name)
	}

	groups := []SCloudgroup{}
	resp, err := self.list(SERVICE_IAM_V3, "", "groups", query)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(&groups, "groups")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListUsersForGroupByAdmin
func (self *SHuaweiClient) GetGroupUsers(groupId string) ([]SClouduser, error) {
	resp, err := self.list(SERVICE_IAM_V3, "", fmt.Sprintf("groups/%s/users", groupId), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list group users")
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "users")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return users, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneCheckDomainPermissionForGroup
func (self *SHuaweiClient) GetGroupRoles(groupId string) ([]SRole, error) {
	res := fmt.Sprintf("domains/%s/groups/%s/roles", self.ownerId, groupId)
	resp, err := self.list(SERVICE_IAM_V3, "", res, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ListRoles")
	}
	roles := []SRole{}
	err = resp.Unmarshal(&roles, "roles")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return roles, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneCreateGroup
func (self *SHuaweiClient) CreateGroup(name, desc string) (*SCloudgroup, error) {
	params := map[string]interface{}{
		"name": name,
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	resp, err := self.post(SERVICE_IAM_V3, "", "groups", map[string]interface{}{"group": params})
	if err != nil {
		return nil, err
	}
	group := &SCloudgroup{client: self}
	err = resp.Unmarshal(group, "group")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return group, nil
}

func (self *SHuaweiClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	return group, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneDeleteGroup
func (self *SHuaweiClient) DeleteGroup(id string) error {
	_, err := self.delete(SERVICE_IAM_V3, "", "groups/"+id)
	return err
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneAddUserToGroup
func (self *SHuaweiClient) AddUserToGroup(groupId, userId string) error {
	_, err := self.put(SERVICE_IAM_V3, "", fmt.Sprintf("groups/%s/users/%s", groupId, userId), nil)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneRemoveUserFromGroup
func (self *SHuaweiClient) RemoveUserFromGroup(groupId, userId string) error {
	_, err := self.delete(SERVICE_IAM_V3, "", fmt.Sprintf("groups/%s/users/%s", groupId, userId))
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=ListCustomPolicies
func (self *SHuaweiClient) GetCustomRoles() ([]SRole, error) {
	query := url.Values{}
	query.Set("per_page", "300")
	page := 1
	query.Set("page", fmt.Sprintf("%d", page))
	ret := []SRole{}
	for {
		resp, err := self.list(SERVICE_IAM, "", "OS-ROLE/roles", query)
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

func (self *SHuaweiClient) DetachGroupRole(groupId, roleId string) error {
	role, err := self.GetRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err := self.KeystoneRemoveDomainPermissionFromGroup(self.ownerId, groupId, roleId)
		if err != nil {
			return errors.Wrapf(err, "remove domain role")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err := self.KeystoneRemoveProjectPermissionFromGroup(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "remove project role ")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err := self.KeystoneRemoveProjectPermissionFromGroup(project.Id, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "remove project role ")
			}
		}
	}
	return nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneRemoveProjectPermissionFromGroup
func (self *SHuaweiClient) KeystoneRemoveProjectPermissionFromGroup(projectId, groupId, roleId string) error {
	res := fmt.Sprintf("projects/%s/groups/%s/roles/%s", projectId, groupId, roleId)
	_, err := self.delete(SERVICE_IAM_V3, "", res)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneAssociateGroupWithProjectPermission
func (self *SHuaweiClient) KeystoneAssociateGroupWithProjectPermission(projectId, groupId, roleId string) error {
	res := fmt.Sprintf("projects/%s/groups/%s/roles/%s", self.GetMosProjectId(), groupId, roleId)
	_, err := self.put(SERVICE_IAM_V3, "", res, nil)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneAssociateGroupWithDomainPermission
func (self *SHuaweiClient) KeystoneAssociateGroupWithDomainPermission(domainId, groupId, roleId string) error {
	res := fmt.Sprintf("domains/%s/groups/%s/roles/%s", domainId, groupId, roleId)
	_, err := self.put(SERVICE_IAM_V3, "", res, nil)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneRemoveDomainPermissionFromGroup
func (self *SHuaweiClient) KeystoneRemoveDomainPermissionFromGroup(domainId, groupId, roleId string) error {
	res := fmt.Sprintf("domains/%s/groups/%s/roles/%s", domainId, groupId, roleId)
	_, err := self.delete(SERVICE_IAM_V3, "", res)
	return err
}

func (self *SHuaweiClient) AttachGroupRole(groupId, roleId string) error {
	role, err := self.GetRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err := self.KeystoneAssociateGroupWithDomainPermission(self.ownerId, groupId, roleId)
		if err != nil {
			return errors.Wrapf(err, "AddRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err := self.KeystoneAssociateGroupWithProjectPermission(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "add project role ")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err := self.KeystoneAssociateGroupWithProjectPermission(project.Id, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "add project role ")
			}
		}
	}
	return nil
}

func (self *SHuaweiClient) AttachGroupCustomRole(groupId, roleId string) error {
	role, err := self.GetCustomRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err := self.KeystoneAssociateGroupWithDomainPermission(self.ownerId, groupId, roleId)
		if err != nil {
			return errors.Wrapf(err, "AddRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err = self.KeystoneAssociateGroupWithProjectPermission(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "add project role ")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err := self.KeystoneAssociateGroupWithProjectPermission(project.Id, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "add project role ")
			}
		}
	}
	return nil
}

func (self *SHuaweiClient) DetachGroupCustomRole(groupId, roleId string) error {
	role, err := self.GetCustomRole(roleId)
	if err != nil {
		return errors.Wrapf(err, "GetCustomRole(%s)", roleId)
	}
	if role.Type == "AX" || role.Type == "AA" {
		err := self.KeystoneRemoveDomainPermissionFromGroup(self.ownerId, groupId, roleId)
		if err != nil {
			return errors.Wrapf(err, "DeleteRole")
		}
		if strings.Contains(strings.ToLower(role.Policy.String()), "obs") {
			err := self.KeystoneRemoveProjectPermissionFromGroup(self.GetMosProjectId(), groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "remove project role ")
			}
		}
	}
	if role.Type == "XA" || role.Type == "AA" {
		projects, err := self.GetProjects()
		if err != nil {
			return errors.Wrapf(err, "GetProjects")
		}
		for _, project := range projects {
			err := self.KeystoneRemoveProjectPermissionFromGroup(project.Id, groupId, role.Id)
			if err != nil {
				return errors.Wrapf(err, "remove project role ")
			}
		}
	}
	return nil
}
