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

package ksyun

import (
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGroup struct {
	client *SKsyunClient

	Description string
	UserGroupId string
	GroupName   string
	CreateDate  time.Time
	Krn         string
	UserCount   int
	PolicyCount int
}

func (self *SGroup) GetName() string {
	return self.GroupName
}

func (self *SGroup) GetGlobalId() string {
	return self.GroupName
}

func (self *SGroup) GetDescription() string {
	return self.Description
}

func (self *SGroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SGroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.client.ListGroupPolicies(self.GroupName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SGroup) AddUser(name string) error {
	return self.client.AddUserToGroup(name, self.GroupName)
}

func (self *SGroup) RemoveUser(name string) error {
	return self.client.RemoveUserFromGroup(name, self.GroupName)
}

func (self *SGroup) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.AttachGroupPolicy(self.GroupName, policyName)
}

func (self *SGroup) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.DetachGroupPolicy(self.GroupName, policyName)
}

func (self *SGroup) Delete() error {
	return self.client.DeleteGroup(self.GroupName)
}

func (self *SKsyunClient) CreateICloudgroup(name string, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (self *SKsyunClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := self.ListGroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (client *SKsyunClient) ListGroups() ([]SGroup, error) {
	params := map[string]string{
		"MaxItems": "100",
	}
	ret := []SGroup{}
	for {
		resp, err := client.iamRequest("", "ListGroups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Groups struct {
				Member []SGroup
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Groups.Member...)
		if len(part.Marker) == 0 || len(part.Groups.Member) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) ListGroupPolicies(name string) ([]SPolicy, error) {
	params := map[string]string{
		"GroupName": name,
		"MaxItems":  "100",
	}

	ret := []SPolicy{}
	for {
		resp, err := client.iamRequest("", "ListGroupPolicies", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			AttachedPolicies struct {
				Member []SPolicy
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.AttachedPolicies.Member...)
		if len(part.Marker) == 0 || len(part.AttachedPolicies.Member) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) AttachGroupPolicy(name, policy string) error {
	params := map[string]string{
		"GroupName": name,
		"PolicyKrn": policy,
	}
	_, err := client.iamRequest("", "AttachGroupPolicy", params)
	return err
}

func (client *SKsyunClient) DetachGroupPolicy(name, policy string) error {
	params := map[string]string{
		"GroupName": name,
		"PolicyKrn": policy,
	}
	_, err := client.iamRequest("", "DetachGroupPolicy", params)
	return err
}

func (client *SKsyunClient) DeleteGroup(name string) error {
	params := map[string]string{
		"GroupName": name,
	}
	_, err := client.iamRequest("", "DeleteGroup", params)
	return err
}

func (client *SKsyunClient) AddUserToGroup(user, group string) error {
	params := map[string]string{
		"GroupName": group,
		"UserName":  user,
	}
	_, err := client.iamRequest("", "AddUserToGroup", params)
	return err
}

func (client *SKsyunClient) RemoveUserFromGroup(user, group string) error {
	params := map[string]string{
		"GroupName": group,
		"UserName":  user,
	}
	_, err := client.iamRequest("", "RemoveUserFromGroup", params)
	return err
}

func (client *SKsyunClient) CreateGroup(name, desc string) (*SGroup, error) {
	params := map[string]string{
		"GroupName":   name,
		"Description": desc,
	}
	resp, err := client.iamRequest("", "CreateGroup", params)
	if err != nil {
		return nil, err
	}
	ret := &SGroup{client: client}
	err = resp.Unmarshal(ret, "Group")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *SKsyunClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	group, err := client.GetGroup(name)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (client *SKsyunClient) GetGroup(name string) (*SGroup, error) {
	params := map[string]string{
		"GroupName": name,
	}
	resp, err := client.iamRequest("", "GetGroup", params)
	if err != nil {
		return nil, err
	}
	ret := &SGroup{client: client}
	err = resp.Unmarshal(ret, "UserGroup")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
