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

package qcloud

import (
	"fmt"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	NORMAL = "NORMAL"
	UNKOWN = "ISOLATED"
)

type SPolicy struct {
	DayOfWeek []int
	Hour      []int
}

type SSnapshotPolicy struct {
	region *SRegion

	AutoSnapshotPolicyName  string
	AutoSnapshotPolicyId    string
	AutoSnapshotPolicyState string
	RetentionDays           int
	Policy                  []SPolicy
	Activated               bool `json:"IsActivated"`
	IsPermanent             bool
}

func (self *SSnapshotPolicy) GetId() string {
	return self.AutoSnapshotPolicyId
}

func (self *SSnapshotPolicy) GetName() string {
	return self.AutoSnapshotPolicyName
}

func (self *SSnapshotPolicy) GetGlobalId() string {
	return self.GetId()
}

func (self *SSnapshotPolicy) GetStatus() string {
	if self.AutoSnapshotPolicyState == NORMAL {
		return api.SNAPSHOT_POLICY_READY
	}
	return api.SNAPSHOT_POLICY_UNKNOWN
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

func (self *SSnapshotPolicy) GetProjectId() string {
	return ""
}

func (self *SSnapshotPolicy) GetRetentionDays() int {
	if self.IsPermanent {
		return -1
	}
	return self.RetentionDays
}

func (self *SSnapshotPolicy) GetRepeatWeekdays() ([]int, error) {
	if len(self.Policy) == 0 {
		return nil, errors.Error("Policy Set Empty")
	}
	repeatWeekdays := self.Policy[0].DayOfWeek
	if len(repeatWeekdays) > 0 {
		if repeatWeekdays[0] == 0 {
			repeatWeekdays = append(repeatWeekdays, 7)[1:]
		}
	}
	return repeatWeekdays, nil
}

func (self *SSnapshotPolicy) GetTimePoints() ([]int, error) {
	if len(self.Policy) == 0 {
		return nil, errors.Error("Policy Set Empty")
	}
	return self.Policy[0].Hour, nil
}

func (self *SSnapshotPolicy) IsActivated() bool {
	return self.Activated
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
	if len(policyId) > 0 {
		params["AutoSnapshotPolicyIds.0"] = policyId
	}
	if limit != 0 {
		params["Limit"] = strconv.Itoa(limit)
		params["Offset"] = strconv.Itoa(offset)
	}
	body, err := self.cbsRequest("DescribeAutoSnapshotPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "Get Snapshot Policies failed")
	}
	snapshotPolicies := make([]SSnapshotPolicy, 0, 1)
	if err := body.Unmarshal(&snapshotPolicies, "AutoSnapshotPolicySet"); err != nil {
		return nil, 0, errors.Wrap(err, "Unmarshal snapshot policies detail failed")
	}
	for i := range snapshotPolicies {
		snapshotPolicies[i].region = self
	}
	return snapshotPolicies, len(snapshotPolicies), nil
}

func (self *SSnapshotPolicy) Delete() error {
	if self.region == nil {
		return fmt.Errorf("Not init region for snapshotPolicy %s", self.GetId())
	}
	return self.region.DeleteSnapshotPolicy(self.GetId())
}

func (self *SRegion) DeleteSnapshotPolicy(snapshotPolicyId string) error {
	params := make(map[string]string)
	params["AutoSnapshotPolicyIds.0"] = snapshotPolicyId
	_, err := self.cbsRequest("DeleteAutoSnapshotPolicies", params)
	if err != nil {
		return errors.Wrapf(err, "delete auto snapshot policy %s failed", snapshotPolicyId)
	}
	return nil
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

	// In Qcloud, that IsPermanent is true means that keep snapshot forever,
	// In OneCloud, that RetentionDays is -1 means that keep snapshot forever.
	if input.RetentionDays == -1 {
		params["IsPermanent"] = strconv.FormatBool(true)
	} else {
		params["RetentionDays"] = strconv.Itoa(input.RetentionDays)
	}
	dayOfWeekPrefix, hourPrefix := "Policy.0.DayOfWeek.", "Policy.0.Hour."
	for index, day := range input.RepeatWeekdays {
		if day == 0 {
			day = 7
		}
		params[dayOfWeekPrefix+strconv.Itoa(index)] = strconv.Itoa(day)
	}
	if len(input.PolicyName) > 0 {
		params["AutoSnapshotPolicyName"] = input.PolicyName
	}
	for index, hour := range input.TimePoints {
		params[hourPrefix+strconv.Itoa(index)] = strconv.Itoa(hour)
	}
	body, err := self.cbsRequest("CreateAutoSnapshotPolicy", params)
	if err != nil {
		return "", errors.Wrap(err, "create auto snapshot policy failed")
	}
	id, _ := body.GetString("AutoSnapshotPolicyId")
	return id, nil
}

func (self *SRegion) UpdateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput, snapshotPolicyId string) error {
	// not implement
	return nil
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	params := make(map[string]string)
	params["AutoSnapshotPolicyId"] = snapshotPolicyId
	params["DiskIds.0"] = diskId
	_, err := self.cbsRequest("BindAutoSnapshotPolicy", params)
	if err != nil {
		return errors.Wrapf(err, "Bind AutoSnapshotPolicy %s to Disk failed", snapshotPolicyId)
	}
	return nil
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	params := make(map[string]string)
	params["AutoSnapshotPolicyId"] = snapshotPolicyId
	params["DiskIds.0"] = diskId
	_, err := self.cbsRequest("UnbindAutoSnapshotPolicy", params)
	if err != nil {
		return errors.Wrapf(err, "Unbind AutoSnapshotPolicy %s of Disk failed", snapshotPolicyId)
	}
	return nil
}

func (self *SRegion) GetSnapshotIdByDiskId(diskID string) ([]string, error) {
	params := make(map[string]string)
	params["DiskId"] = diskID

	rps, err := self.cbsRequest("DescribeDiskAssociatedAutoSnapshotPolicy", params)
	if err != nil {
		return nil, errors.Wrapf(err, "Get All SnapshotpolicyIDs of Disk %s failed", diskID)
	}

	snapshotpolicies := make([]SSnapshotPolicy, 0)
	if err := rps.Unmarshal(&snapshotpolicies, "AutoSnapshotPolicySet"); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal snapshot policies details failed")
	}
	ret := make([]string, len(snapshotpolicies))
	for i := range snapshotpolicies {
		ret[i] = snapshotpolicies[i].GetId()
	}
	return ret, nil
}
