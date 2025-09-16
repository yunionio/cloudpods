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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SAccessGroup struct {
	region *SRegion

	RuleCount        int
	AccessGroupType  string
	Description      string
	AccessGroupName  string
	MountTargetCount int
	FileSystemType   string
}

func (self *SAccessGroup) GetName() string {
	return self.AccessGroupName
}

func (self *SAccessGroup) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.GetFileSystemType(), self.AccessGroupName)
}

func (self *SAccessGroup) GetFileSystemType() string {
	return self.FileSystemType
}

func (self *SAccessGroup) GetDesc() string {
	return self.Description
}

func (self *SAccessGroup) GetMountTargetCount() int {
	return self.MountTargetCount
}

func (self *SAccessGroup) GetNetworkType() string {
	return strings.ToLower(self.AccessGroupType)
}

func (self *SAccessGroup) Delete() error {
	return self.region.DeleteAccessGroup(self.FileSystemType, self.AccessGroupName)
}

func (self *SAccessGroup) GetSupporedUserAccessTypes() []cloudprovider.TUserAccessType {
	return []cloudprovider.TUserAccessType{
		cloudprovider.UserAccessTypeAllSquash,
		cloudprovider.UserAccessTypeRootSquash,
		cloudprovider.UserAccessTypeNoRootSquash,
	}
}

func (self *SAccessGroup) GetRules() ([]cloudprovider.IAccessGroupRule, error) {
	rules, err := self.region.GetAccessGroupRules(self.AccessGroupName)
	if err != nil {
		return nil, errors.Wrapf(err, "GetAccessGroupRules")
	}
	ret := []cloudprovider.IAccessGroupRule{}
	for i := range rules {
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (self *SRegion) GetAccessGroups(fsType string) ([]SAccessGroup, error) {
	pageNum := 1
	params := map[string]string{
		"RegionId":       self.RegionId,
		"PageSize":       "100",
		"PageNumber":     fmt.Sprintf("%d", pageNum),
		"FileSystemType": fsType,
	}
	ret := []SAccessGroup{}
	for {
		resp, err := self.nasRequest("DescribeAccessGroups", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeAccessGroups")
		}

		part := struct {
			TotalCount   int
			AccessGroups struct {
				AccessGroup []SAccessGroup
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		for i := range part.AccessGroups.AccessGroup {
			part.AccessGroups.AccessGroup[i].FileSystemType = fsType
			part.AccessGroups.AccessGroup[i].region = self
		}
		ret = append(ret, part.AccessGroups.AccessGroup...)
		if len(ret) >= part.TotalCount || len(part.AccessGroups.AccessGroup) == 0 {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	return ret, nil
}

func (self *SRegion) GetAccessGroupRules(groupName string) ([]SAccessGroupRule, error) {
	pageNum := 1
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessGroupName": groupName,
		"PageSize":        "100",
		"PageNumber":      "1",
	}
	ret := []SAccessGroupRule{}
	for {
		resp, err := self.nasRequest("DescribeAccessRules", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeAccessRules")
		}
		part := struct {
			TotalCount  int
			AccessRules struct {
				AccessRule []SAccessGroupRule
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		for i := range part.AccessRules.AccessRule {
			part.AccessRules.AccessRule[i].AccessGroupName = groupName
			part.AccessRules.AccessRule[i].region = self
		}
		ret = append(ret, part.AccessRules.AccessRule...)
		if len(ret) >= part.TotalCount || len(part.AccessRules.AccessRule) == 0 {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	return ret, nil
}

func (self *SRegion) GetICloudAccessGroups() ([]cloudprovider.ICloudAccessGroup, error) {
	standardAccessGroups, err := self.GetAccessGroups("standard")
	if err != nil {
		return nil, errors.Wrapf(err, "GetAccessGroups")
	}
	extremeAccessGroups, err := self.GetAccessGroups("extreme")
	if err != nil {
		return nil, errors.Wrapf(err, "GetAccessGroups")
	}
	ret := []cloudprovider.ICloudAccessGroup{}
	for _, accessGroups := range [][]SAccessGroup{standardAccessGroups, extremeAccessGroups} {
		for i := range accessGroups {
			accessGroups[i].region = self
			ret = append(ret, &accessGroups[i])
		}
	}
	return ret, nil
}

func (self *SRegion) GetICloudAccessGroupById(id string) (cloudprovider.ICloudAccessGroup, error) {
	groups, err := self.GetICloudAccessGroups()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetICloudAccessGroups")
	}
	for i := range groups {
		if groups[i].GetGlobalId() == id {
			return groups[i], nil
		}
	}
	return nil, errors.Wrapf(err, "GetICloudAccessGroupById(%s)", id)
}

func (self *SRegion) CreateICloudAccessGroup(opts *cloudprovider.SAccessGroup) (cloudprovider.ICloudAccessGroup, error) {
	if len(opts.FileSystemType) == 0 {
		opts.FileSystemType = "standard"
	}
	err := self.CreateAccessGroup(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateAccessGroup")
	}
	return self.GetICloudAccessGroupById(fmt.Sprintf("%s/%s", opts.FileSystemType, opts.Name))
}

func (self *SRegion) CreateAccessGroup(opts *cloudprovider.SAccessGroup) error {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessGroupName": opts.Name,
		"AccessGroupType": opts.NetworkType,
	}
	if len(opts.FileSystemType) > 0 {
		params["FileSystemType"] = opts.FileSystemType
	}
	_, err := self.nasRequest("CreateAccessGroup", params)
	return errors.Wrapf(err, "CreateAccessGroup")
}

func (self *SRegion) DeleteAccessGroup(fsType, name string) error {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessGroupName": name,
		"FileSystemType":  fsType,
	}
	_, err := self.nasRequest("DeleteAccessGroup", params)
	return errors.Wrapf(err, "DeleteAccessGroup")
}

func (self *SRegion) DeleteAccessGroupRule(groupName, ruleId string) error {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessRuleId":    ruleId,
		"AccessGroupName": groupName,
	}
	_, err := self.nasRequest("DeleteAccessRule", params)
	return errors.Wrapf(err, "DeleteAccessRule")
}

func (self *SRegion) CreateAccessGroupRule(source, fsType, groupName string, rwType cloudprovider.TRWAccessType, userType cloudprovider.TUserAccessType, priority int) (string, error) {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"SourceCidrIp":    source,
		"AccessGroupName": groupName,
		"Priority":        fmt.Sprintf("%d", priority),
		"FileSystemType":  fsType,
	}
	switch rwType {
	case cloudprovider.RWAccessTypeR:
		params["RWAccessType"] = "RDONLY"
	case cloudprovider.RWAccessTypeRW:
		params["RWAccessType"] = "RDWR"
	}
	switch userType {
	case cloudprovider.UserAccessTypeAllSquash:
		params["UserAccessType"] = "all_squash"
	case cloudprovider.UserAccessTypeRootSquash:
		params["UserAccessType"] = "root_squash"
	case cloudprovider.UserAccessTypeNoRootSquash:
		params["UserAccessType"] = "no_squash"
	}
	resp, err := self.nasRequest("CreateAccessRule", params)
	if err != nil {
		return "", errors.Wrapf(err, "CreateAccessRule")
	}
	return resp.GetString("AccessRuleId")
}

func (self *SAccessGroup) CreateRule(opts *cloudprovider.AccessGroupRule) (cloudprovider.IAccessGroupRule, error) {
	ruleId, err := self.region.CreateAccessGroupRule(opts.Source, self.FileSystemType, self.AccessGroupName, opts.RWAccessType, opts.UserAccessType, opts.Priority)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateAccessGroupRule")
	}
	rules, err := self.region.GetAccessGroupRules(self.AccessGroupName)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].AccessRuleId == ruleId {
			return &rules[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, ruleId)
}
