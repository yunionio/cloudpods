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

package google

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDailySchedule struct {
	DaysInCycle int
	StartTime   string
	Duration    string
}

type SDayOfWeek struct {
	Day       string
	StartTime string
	Duration  string
}

type SHourlySchedule struct {
	HoursInCycle int
	StartTime    string
	Duration     string
}

type SWeeklySchedule struct {
	DayOfWeeks []SDayOfWeek
}

type SSchedule struct {
	WeeklySchedule SWeeklySchedule
	DailySchedule  SDailySchedule
	HourlySchedule SHourlySchedule
}

type SRetentionPolicy struct {
	MaxRetentionDays   int
	OnSourceDiskDelete string
}

type SSnapshotProperties struct {
	StorageLocations []string
	GuestFlush       bool
}

type SSnapshotSchedulePolicy struct {
	Schedule           SSchedule
	RetentionPolicy    SRetentionPolicy
	SnapshotProperties SSnapshotProperties
}

type SResourcePolicy struct {
	region *SRegion
	SResourceBase

	Id string

	CreationTimestamp      time.Time
	Region                 string
	Status                 string
	Kind                   string
	SnapshotSchedulePolicy SSnapshotSchedulePolicy `json:"snapshotSchedulePolicy"`
}

func (region *SRegion) GetResourcePolicies(maxResults int, pageToken string) ([]SResourcePolicy, error) {
	policies := []SResourcePolicy{}
	resource := fmt.Sprintf("regions/%s/resourcePolicies", region.Name)
	params := map[string]string{}
	return policies, region.List(resource, params, maxResults, pageToken, &policies)
}

func (region *SRegion) GetResourcePolicy(id string) (*SResourcePolicy, error) {
	policy := &SResourcePolicy{region: region}
	return policy, region.Get(id, policy)
}

func (policy *SResourcePolicy) GetStatus() string {
	switch policy.Status {
	case "READY":
		return api.SNAPSHOT_POLICY_READY
	default:
		log.Errorf("unknown policy status %s", policy.Status)
		return api.SNAPSHOT_POLICY_UNKNOWN
	}
}

func (policy *SResourcePolicy) Refresh() error {
	_policy, err := policy.region.GetResourcePolicy(policy.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(policy, _policy)
}

func (policy *SResourcePolicy) IsEmulated() bool {
	return false
}

func (policy *SResourcePolicy) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (policy *SResourcePolicy) GetProjectId() string {
	return policy.region.GetProjectId()
}

func (policy *SResourcePolicy) GetRetentionDays() int {
	return policy.SnapshotSchedulePolicy.RetentionPolicy.MaxRetentionDays
}

func (policy *SResourcePolicy) GetRepeatWeekdays() ([]int, error) {
	result := []int{1, 2, 3, 4, 5, 6, 7}
	if len(policy.SnapshotSchedulePolicy.Schedule.WeeklySchedule.DayOfWeeks) > 0 {
		return nil, fmt.Errorf("current not support dayOfWeeks")
	}
	if policy.SnapshotSchedulePolicy.Schedule.HourlySchedule.HoursInCycle != 0 {
		return nil, fmt.Errorf("current not support hourlySchedule")
	}
	return result, nil
}

func (policy *SResourcePolicy) GetTimePoints() ([]int, error) {
	result := []int{}
	if len(policy.SnapshotSchedulePolicy.Schedule.DailySchedule.StartTime) == 0 {
		return nil, fmt.Errorf("current only support dailySchedule")
	}
	if startInfo := strings.Split(policy.SnapshotSchedulePolicy.Schedule.DailySchedule.StartTime, ":"); len(startInfo) >= 2 {
		point, err := strconv.Atoi(startInfo[0])
		if err != nil {
			return nil, errors.Wrapf(err, "convert %s", policy.SnapshotSchedulePolicy.Schedule.DailySchedule.StartTime)
		}
		result = append(result, point)
		if startInfo[1] != "00" {
			result = append(result, point+1)
		}
	}
	return result, nil
}

func (policy *SResourcePolicy) IsActivated() bool {
	return true
}
