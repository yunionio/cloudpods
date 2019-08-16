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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SSnapshotPolicyType string

const (
	Creating  SSnapshotPolicyType = "Creating"
	Available SSnapshotPolicyType = "Available"
	Normal    SSnapshotPolicyType = "Normal"
)

type SSnapshotPolicy struct {
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
	// XXX: aliyun文档与实际返回值不符
	if self.Status == Normal || self.Status == Available {
		return api.SNAPSHOT_POLICY_READY
	} else if self.Status == Creating {
		return api.SNAPSHOT_POLICY_CREATING
	} else {
		return api.SNAPSHOT_POLICY_UNKNOWN
	}
}

func (self *SSnapshotPolicy) Refresh() error {
	if snapshotPolicies, total, err := self.region.GetSnapshotPolicies(self.AutoSnapshotPolicyId, 0, 1); err != nil {
		return err
	} else if total != 1 {
		return cloudprovider.ErrNotFound
	} else if err := jsonutils.Update(self, snapshotPolicies[0]); err != nil {
		return err
	}
	return nil
}

func (self *SSnapshotPolicy) IsEmulated() bool {
	return false
}

func (self *SSnapshotPolicy) GetMetadata() *jsonutils.JSONDict {
	return nil
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

func (self *SSnapshotPolicy) IsActivated() bool {
	return true
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	snapshotPolicies, total, err := self.GetSnapshotPolicies("", 0, 50)
	if err != nil {
		return nil, err
	}
	for len(snapshotPolicies) < total {
		var parts []SSnapshotPolicy
		parts, total, err = self.GetSnapshotPolicies("", len(snapshotPolicies), 50)
		if err != nil {
			return nil, err
		}
		snapshotPolicies = append(snapshotPolicies, parts...)
	}
	ret := make([]cloudprovider.ICloudSnapshotPolicy, len(snapshotPolicies))
	for i := 0; i < len(snapshotPolicies); i += 1 {
		ret[i] = &snapshotPolicies[i]
	}
	return ret, nil
}

func (self *SRegion) GetSnapshotPolicies(policyId string, offset int, limit int) ([]SSnapshotPolicy, int, error) {
	params := make(map[string]string)

	params["RegionId"] = self.RegionId
	if limit != 0 {
		params["PageSize"] = fmt.Sprintf("%d", limit)
		params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	}

	if len(policyId) > 0 {
		params["AutoSnapshotPolicyId"] = policyId
	}

	body, err := self.ecsRequest("DescribeAutoSnapshotPolicyEx", params)
	if err != nil {
		return nil, 0, fmt.Errorf("GetSnapshotPolicys fail %s", err)
	}

	snapshotPolicies := make([]SSnapshotPolicy, 0)
	if err := body.Unmarshal(&snapshotPolicies, "AutoSnapshotPolicies", "AutoSnapshotPolicy"); err != nil {
		return nil, 0, fmt.Errorf("Unmarshal snapshot policies details fail %s", err)
	}
	total, _ := body.Int("TotalCount")
	for i := 0; i < len(snapshotPolicies); i += 1 {
		snapshotPolicies[i].region = self
	}
	return snapshotPolicies, int(total), nil
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

func (self *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	policies, _, err := self.GetSnapshotPolicies(snapshotPolicyId, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &policies[0], nil
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
	params["autoSnapshotPolicyName"] = input.PolicyName
	if body, err := self.ecsRequest("CreateAutoSnapshotPolicy", params); err != nil {
		return "", fmt.Errorf("CreateAutoSnapshotPolicy fail %s", err)
	} else {
		return body.GetString("AutoSnapshotPolicyId")
	}
}

func (self *SRegion) UpdateSnapshotPolicy(
	snapshotPolicyId string, retentionDays *int,
	repeatWeekdays, timePoints *jsonutils.JSONArray, policyName string,
) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if len(policyName) > 0 {
		params["autoSnapshotPolicyName"] = policyName
	}
	if retentionDays != nil {
		params["retentionDays"] = strconv.Itoa(*retentionDays)
	}
	if repeatWeekdays != nil {
		params["repeatWeekdays"] = repeatWeekdays.String()
	}
	if timePoints != nil {
		params["timePoints"] = timePoints.String()
	}
	_, err := self.ecsRequest("ModifyAutoSnapshotPolicyEx", params)
	if err != nil {
		return fmt.Errorf("ModifyAutoSnapshotPolicyEx Fail %s", err)
	}
	return nil
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["autoSnapshotPolicyId"] = snapshotPolicyId
	diskIds := []string{diskId}
	params["diskIds"] = jsonutils.Marshal(diskIds).String()
	_, err := self.ecsRequest("ApplyAutoSnapshotPolicy", params)
	if err != nil {
		return fmt.Errorf("ApplyAutoSnapshotPolicy Fail %s", err)
	}
	return nil
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	diskIds := []string{diskId}
	params["diskIds"] = jsonutils.Marshal(diskIds).String()
	_, err := self.ecsRequest("CancelAutoSnapshotPolicy", params)
	if err != nil {
		return fmt.Errorf("CancelAutoSnapshotPolicy Fail %s", err)
	}
	return nil
}
