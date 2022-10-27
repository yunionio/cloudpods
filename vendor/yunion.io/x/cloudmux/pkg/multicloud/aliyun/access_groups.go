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

	alierr "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"

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

func (self *SAccessGroup) IsDefault() bool {
	return self.AccessGroupName == "DEFAULT_CLASSIC_GROUP_NAME" || self.AccessGroupName == "DEFAULT_VPC_GROUP_NAME"
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

func (self *SAccessGroup) GetMaxPriority() int {
	return 1
}

func (self *SAccessGroup) GetMinPriority() int {
	return 100
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

func (self *SAccessGroup) GetRules() ([]cloudprovider.AccessGroupRule, error) {
	rules := []SAccessGroupRule{}
	num := 1
	for {
		part, total, err := self.region.GetAccessGroupRules(self.AccessGroupName, 50, num)
		if err != nil {
			return nil, errors.Wrapf(err, "GetAccessGroupRules")
		}
		rules = append(rules, part...)
		if len(rules) >= total {
			break
		}
	}
	ret := []cloudprovider.AccessGroupRule{}
	for i := range rules {
		rule := cloudprovider.AccessGroupRule{
			ExternalId: rules[i].AccessRuleId,
			Priority:   rules[i].Priority,
			Source:     rules[i].SourceCidrIp,
		}
		switch rules[i].RWAccess {
		case "RDWR":
			rule.RWAccessType = cloudprovider.RWAccessTypeRW
		case "RDONLY":
			rule.RWAccessType = cloudprovider.RWAccessTypeR
		}
		switch rules[i].UserAccess {
		case "no_squash":
			rule.UserAccessType = cloudprovider.UserAccessTypeNoRootSquash
		case "root_squash":
			rule.UserAccessType = cloudprovider.UserAccessTypeRootSquash
		case "all_squash":
			rule.UserAccessType = cloudprovider.UserAccessTypeAllSquash
		}
		ret = append(ret, rule)
	}
	return ret, nil
}

func (self *SRegion) getAccessGroups(fsType string, pageSize, pageNum int) ([]SAccessGroup, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if pageNum < 1 {
		pageNum = 1
	}
	params := map[string]string{
		"RegionId":       self.RegionId,
		"PageSize":       fmt.Sprintf("%d", pageSize),
		"PageNumber":     fmt.Sprintf("%d", pageNum),
		"FileSystemType": fsType,
	}
	resp, err := self.nasRequest("DescribeAccessGroups", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeAccessGroups")
	}

	ret := struct {
		TotalCount   int
		AccessGroups struct {
			AccessGroup []SAccessGroup
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret.AccessGroups.AccessGroup, ret.TotalCount, nil
}

type SAccessGroupRule struct {
	RWAccess     string
	UserAccess   string
	Priority     int
	SourceCidrIp string
	AccessRuleId string
}

func (self *SRegion) GetAccessGroupRules(groupName string, pageSize, pageNum int) ([]SAccessGroupRule, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if pageNum < 1 {
		pageNum = 1
	}
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessGroupName": groupName,
		"PageSize":        fmt.Sprintf("%d", pageSize),
		"PageNumber":      fmt.Sprintf("%d", pageNum),
	}
	resp, err := self.nasRequest("DescribeAccessRules", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeAccessRules")
	}
	ret := struct {
		TotalCount  int
		AccessRules struct {
			AccessRule []SAccessGroupRule
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret.AccessRules.AccessRule, ret.TotalCount, nil
}

func (self *SRegion) GetAccessGroups(fsType string) ([]SAccessGroup, error) {
	accessGroups := []SAccessGroup{}
	num := 1
	for {
		part, total, err := self.getAccessGroups(fsType, 50, num)
		if err != nil {
			if e, ok := errors.Cause(err).(*alierr.ServerError); ok && e.ErrorCode() == "Region.NotSupported" {
				return accessGroups, nil
			}
			return nil, errors.Wrapf(err, "GetAccessGroups")
		}
		for i := range part {
			part[i].FileSystemType = fsType
			accessGroups = append(accessGroups, part[i])
		}
		if len(accessGroups) >= total {
			break
		}
	}
	return accessGroups, nil
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
		"FileSystemType":  opts.FileSystemType,
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

func (self *SRegion) DeleteAccessGroupRule(fsType, groupName, ruleId string) error {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"AccessRuleId":    ruleId,
		"AccessGroupName": groupName,
		"FileSystemType":  fsType,
	}
	_, err := self.nasRequest("DeleteAccessRule", params)
	return errors.Wrapf(err, "DeleteAccessRule")
}

func (self *SRegion) CreateAccessGroupRule(source, fsType, groupName string, rwType cloudprovider.TRWAccessType, userType cloudprovider.TUserAccessType, priority int) error {
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
	_, err := self.nasRequest("CreateAccessRule", params)
	return errors.Wrapf(err, "CreateAccessRule")
}

func (self *SAccessGroup) SyncRules(common, added, removed cloudprovider.AccessGroupRuleSet) error {
	for _, rule := range removed {
		err := self.region.DeleteAccessGroupRule(self.FileSystemType, self.AccessGroupName, rule.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "DeleteAccessGroupRule")
		}
	}
	for _, rule := range added {
		err := self.region.CreateAccessGroupRule(rule.Source, self.FileSystemType, self.AccessGroupName, rule.RWAccessType, rule.UserAccessType, rule.Priority)
		if err != nil {
			return errors.Wrapf(err, "CreateAccessGroupRule")
		}
	}
	return nil
}
