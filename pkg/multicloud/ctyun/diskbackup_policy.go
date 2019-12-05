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

package ctyun

import (
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDiskBacupPolicy struct {
	region *SRegion

	PolicyResourceCount int64           `json:"policy_resource_count"`
	BackupPolicyName    string          `json:"backup_policy_name"`
	ScheduledPolicy     ScheduledPolicy `json:"scheduled_policy"`
	BackupPolicyID      string          `json:"backup_policy_id"`
}

type ScheduledPolicy struct {
	RententionNum               int    `json:"rentention_num"`
	StartTime                   string `json:"start_time"`
	Status                      string `json:"status"`
	RemainFirstBackupOfCurMonth string `json:"remain_first_backup_of_curMonth"`
	Frequency                   int    `json:"frequency"`
}

func (self *SDiskBacupPolicy) GetId() string {
	return self.BackupPolicyID
}

func (self *SDiskBacupPolicy) GetName() string {
	return self.BackupPolicyName
}

func (self *SDiskBacupPolicy) GetGlobalId() string {
	return self.GetId()
}

func (self *SDiskBacupPolicy) GetStatus() string {
	if self.ScheduledPolicy.Status == "ON" || self.ScheduledPolicy.Status == "OFF" {
		return api.SNAPSHOT_POLICY_READY
	}
	return api.SNAPSHOT_POLICY_UNKNOWN
}

func (self *SDiskBacupPolicy) Refresh() error {
	policy, err := self.region.GetDiskBackupPolicy(self.GetId())
	if err != nil {
		return errors.Wrap(err, "SDiskBacupPolicy.Refresh")
	}

	err = jsonutils.Update(self, policy)
	if err != nil {
		return errors.Wrap(err, "SDiskBacupPolicy.Refresh.Update")
	}

	return nil
}

func (self *SDiskBacupPolicy) IsEmulated() bool {
	return false
}

func (self *SDiskBacupPolicy) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDiskBacupPolicy) GetProjectId() string {
	return ""
}

func (self *SDiskBacupPolicy) IsActivated() bool {
	if self.ScheduledPolicy.Status == "ON" {
		return true
	}

	return false
}

func (self *SDiskBacupPolicy) GetRetentionDays() int {
	return self.ScheduledPolicy.RententionNum
}

func (self *SDiskBacupPolicy) GetRepeatWeekdays() ([]int, error) {
	return []int{self.ScheduledPolicy.Frequency}, nil
}

func (self *SDiskBacupPolicy) GetTimePoints() ([]int, error) {
	ret, err := strconv.Atoi(self.ScheduledPolicy.StartTime[0:2])
	return []int{ret}, err
}

func (self *SRegion) GetDiskBackupPolices() ([]SDiskBacupPolicy, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryDiskBackupPolicys", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetDiskBackupPolices.DoGet")
	}

	ret := make([]SDiskBacupPolicy, 0)
	err = resp.Unmarshal(&ret, "returnObj", "backup_policies")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetDiskBackupPolices.Unmarshal")
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

func (self *SRegion) GetDiskBackupPolicy(policyId string) (*SDiskBacupPolicy, error) {
	polices, err := self.GetDiskBackupPolices()
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetDiskBackupPolicy.GetDiskBackupPolices")
	}

	for i := range polices {
		policy := polices[i]
		if policy.BackupPolicyID == policyId {
			return &policy, nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetDiskBackupPolicy")
}

func (self *SRegion) CreateDiskBackupPolicy(name, startTime, frequency, rententionNum, firstBackup, status string) error {
	policyParams := jsonutils.NewDict()
	policyParams.Set("policyName", jsonutils.NewString(name))
	scheduleParams := jsonutils.NewDict()
	scheduleParams.Set("startTime", jsonutils.NewString(startTime))
	scheduleParams.Set("frequency", jsonutils.NewString(frequency))
	scheduleParams.Set("rententionNum", jsonutils.NewString(rententionNum))
	scheduleParams.Set("firstBackup", jsonutils.NewString(firstBackup))
	scheduleParams.Set("status", jsonutils.NewString(status))
	policyParams.Set("scheduledPolicy", scheduleParams)

	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"jsonStr":  policyParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/createDiskBackupPolicy", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.CreateDiskBackupPolicy.DoPost")
	}

	return nil
}

func (self *SRegion) BindingDiskBackupPolicy(policyId, resourceId, resourceType string) error {
	policyParams := jsonutils.NewDict()
	policyParams.Set("policyId", jsonutils.NewString(policyId))
	resourcesParams := jsonutils.NewArray()
	resourcesParam := jsonutils.NewDict()
	resourcesParam.Set("resourceId", jsonutils.NewString(resourceId))
	resourcesParam.Set("resourceType", jsonutils.NewString(resourceType))
	resourcesParams.Add(resourcesParam)
	policyParams.Set("resources", resourcesParams)

	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"jsonStr":  policyParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/bindResourceToPolicy", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.BindingDiskBackupPolicy.DoPost")
	}

	return nil
}

func (self *SRegion) UnBindDiskBackupPolicy(policyId, resourceId string) error {
	resourcesParams := jsonutils.NewArray()
	resourcesParam := jsonutils.NewDict()
	resourcesParam.Set("resource_id", jsonutils.NewString(resourceId))
	resourcesParams.Add(resourcesParam)

	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"jsonStr":  resourcesParams,
		"policyId": jsonutils.NewString(policyId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/unBindResourceToPolicy", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.UnBindDiskBackupPolicy.DoPost")
	}

	return nil
}
