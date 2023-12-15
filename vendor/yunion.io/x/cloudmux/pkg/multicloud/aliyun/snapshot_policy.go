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
	"sort"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSnapshotPolicyType string

const (
	Creating  SSnapshotPolicyType = "Creating"
	Available SSnapshotPolicyType = "Available"
	Normal    SSnapshotPolicyType = "Normal"
)

type SSnapshotPolicy struct {
	multicloud.SResourceBase
	AliyunTags
	region *SRegion

	AutoSnapshotPolicyName string
	AutoSnapshotPolicyId   string
	RepeatWeekdays         string
	TimePoints             string
	RetentionDays          int
	Status                 SSnapshotPolicyType
}

func (self *SSnapshotPolicy) GetId() string {
	return self.AutoSnapshotPolicyId
}

func (self *SSnapshotPolicy) GetName() string {
	return self.AutoSnapshotPolicyName
}

func (self *SSnapshotPolicy) GetStatus() string {
	switch self.Status {
	case Normal, Available:
		return apis.STATUS_AVAILABLE
	case Creating:
		return apis.STATUS_CREATING
	default:
		return apis.STATUS_UNKNOWN
	}
}

func (self *SSnapshotPolicy) Refresh() error {
	policies, err := self.region.GetSnapshotPolicies(self.AutoSnapshotPolicyId)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshotPolicies")
	}
	for i := range policies {
		if policies[i].AutoSnapshotPolicyId == self.AutoSnapshotPolicyId {
			return jsonutils.Update(self, policies[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.AutoSnapshotPolicyId)
}

func (self *SSnapshotPolicy) IsEmulated() bool {
	return false
}

func (self *SSnapshotPolicy) GetGlobalId() string {
	return self.AutoSnapshotPolicyId
}

func (self *SSnapshotPolicy) GetProjectId() string {
	return ""
}

func (self *SSnapshotPolicy) GetRetentionDays() int {
	return self.RetentionDays
}

func sliceAtoi(sa []string) ([]int, error) {
	si := make([]int, 0, len(sa))
	for _, a := range sa {
		i, err := strconv.Atoi(a)
		if err != nil {
			return si, err
		}
		si = append(si, i)
	}
	return si, nil
}

func stringToIntDays(days []string) ([]int, error) {
	idays, err := sliceAtoi(days)
	if err != nil {
		return nil, err
	}
	sort.Ints(idays)
	return idays, nil
}

func parsePolicy(policy string) ([]int, error) {
	tp, err := jsonutils.ParseString(policy)
	if err != nil {
		return nil, fmt.Errorf("Parse policy %s error %s", policy, err)
	}
	atp, ok := tp.(*jsonutils.JSONArray)
	if !ok {
		return nil, fmt.Errorf("Policy %s Wrong format", tp)
	}
	return stringToIntDays(atp.GetStringArray())
}

func (self *SSnapshotPolicy) GetRepeatWeekdays() ([]int, error) {
	return parsePolicy(self.RepeatWeekdays)
}

func (self *SSnapshotPolicy) GetTimePoints() ([]int, error) {
	return parsePolicy(self.TimePoints)
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	policies, err := self.GetSnapshotPolicies("")
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudSnapshotPolicy, len(policies))
	for i := 0; i < len(policies); i += 1 {
		ret[i] = &policies[i]
	}
	return ret, nil
}

func (self *SRegion) GetSnapshotPolicies(policyId string) ([]SSnapshotPolicy, error) {
	params := make(map[string]string)

	params["RegionId"] = self.RegionId
	params["PageSize"] = "100"
	pageNum := 1
	params["PageNumber"] = fmt.Sprintf("%d", pageNum)

	if len(policyId) > 0 {
		params["AutoSnapshotPolicyId"] = policyId
	}

	ret := make([]SSnapshotPolicy, 0)
	for {
		resp, err := self.ecsRequest("DescribeAutoSnapshotPolicyEx", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeAutoSnapshotPolicyEx")
		}
		part := struct {
			TotalCount           int
			AutoSnapshotPolicies struct {
				AutoSnapshotPolicy []SSnapshotPolicy
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.AutoSnapshotPolicies.AutoSnapshotPolicy...)
		if len(ret) >= part.TotalCount || len(part.AutoSnapshotPolicies.AutoSnapshotPolicy) == 0 {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	for i := 0; i < len(ret); i += 1 {
		ret[i].region = self
	}
	return ret, nil
}

func (self *SSnapshotPolicy) Delete() error {
	if self.region == nil {
		return fmt.Errorf("Not init region for snapshotPolicy %s", self.AutoSnapshotPolicyId)
	}
	return self.region.DeleteSnapshotPolicy(self.AutoSnapshotPolicyId)
}

func (self *SRegion) DeleteSnapshotPolicy(snapshotPolicyId string) error {
	params := make(map[string]string)
	params["autoSnapshotPolicyId"] = snapshotPolicyId
	params["regionId"] = self.RegionId
	_, err := self.ecsRequest("DeleteAutoSnapshotPolicy", params)
	return err
}

func (self *SRegion) GetISnapshotPolicyById(id string) (cloudprovider.ICloudSnapshotPolicy, error) {
	policies, err := self.GetSnapshotPolicies(id)
	if err != nil {
		return nil, err
	}
	for i := range policies {
		if policies[i].AutoSnapshotPolicyId == id {
			return &policies[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	if input.RepeatWeekdays == nil {
		return "", fmt.Errorf("Can't create snapshot policy with nil repeatWeekdays")
	}
	if input.TimePoints == nil {
		return "", fmt.Errorf("Can't create snapshot policy with nil timePoints")
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["repeatWeekdays"] = jsonutils.Marshal(input.GetStringArrayRepeatWeekdays()).String()
	params["timePoints"] = jsonutils.Marshal(input.GetStringArrayTimePoints()).String()
	params["retentionDays"] = strconv.Itoa(input.RetentionDays)
	params["autoSnapshotPolicyName"] = input.Name
	body, err := self.ecsRequest("CreateAutoSnapshotPolicy", params)
	if err != nil {
		return "", errors.Wrapf(err, "CreateAutoSnapshotPolicy")
	}
	return body.GetString("AutoSnapshotPolicyId")
}

func (self *SRegion) UpdateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput, snapshotPolicyId string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if input.RetentionDays != 0 {
		params["retentionDays"] = strconv.Itoa(input.RetentionDays)
	}
	if input.RepeatWeekdays != nil && len(input.RepeatWeekdays) != 0 {
		params["repeatWeekdays"] = jsonutils.Marshal(input.GetStringArrayRepeatWeekdays()).String()
	}
	if input.TimePoints != nil && len(input.TimePoints) != 0 {
		params["timePoints"] = jsonutils.Marshal(input.GetStringArrayTimePoints()).String()
	}
	_, err := self.ecsRequest("ModifyAutoSnapshotPolicyEx", params)
	if err != nil {
		return fmt.Errorf("ModifyAutoSnapshotPolicyEx Fail %s", err)
	}
	return nil
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskIds []string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["autoSnapshotPolicyId"] = snapshotPolicyId
	params["diskIds"] = jsonutils.Marshal(diskIds).String()
	_, err := self.ecsRequest("ApplyAutoSnapshotPolicy", params)
	if err != nil {
		return errors.Wrapf(err, "ApplyAutoSnapshotPolicy")
	}
	return nil
}

func (self *SSnapshotPolicy) ApplyDisks(ids []string) error {
	return self.region.ApplySnapshotPolicyToDisks(self.AutoSnapshotPolicyId, ids)
}

func (self *SSnapshotPolicy) GetApplyDiskIds() ([]string, error) {
	disks, err := self.region.GetDisks("", "", "", nil, self.AutoSnapshotPolicyId)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for _, disk := range disks {
		ret = append(ret, disk.DiskId)
	}
	return ret, nil
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskIds []string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["diskIds"] = jsonutils.Marshal(diskIds).String()
	_, err := self.ecsRequest("CancelAutoSnapshotPolicy", params)
	if err != nil {
		return errors.Wrapf(err, "CancelAutoSnapshotPolicy")
	}
	return nil
}

func (self *SSnapshotPolicy) CancelDisks(ids []string) error {
	return self.region.CancelSnapshotPolicyToDisks(self.AutoSnapshotPolicyId, ids)
}
