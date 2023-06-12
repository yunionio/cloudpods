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

package compute

import (
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/bitmap"
)

type SnapshotPolicyDetails struct {
	apis.VirtualResourceDetails

	SSnapshotPolicy

	RetentionDays         int   `json:"retention_days"`
	RepeatWeekdaysDisplay []int `json:"repeat_weekdays_display"`
	TimePointsDisplay     []int `json:"time_points_display"`
	IsActivated           *bool `json:"is_activated,omitempty"`

	BindingDiskCount int `json:"binding_disk_count"`
}

type SnapshotPolicyResourceInfo struct {
	// 快照策略名称
	Snapshotpolicy string `json:"snapshotpolicy"`
}

type SSnapshotPolicyCreateInput struct {
	apis.VirtualResourceCreateInput

	RetentionDays  int    `json:"retention_days"`
	RepeatWeekdays string `json:"repeat_weekdays"`
	TimePoints     string `json:"time_points"`
}

func (self SSnapshotPolicyCreateInput) GetRepeatWeekdays(limit int) (uint32, error) {
	days, err := jsonutils.ParseString(self.RepeatWeekdays)
	if err != nil {
		return 0, errors.Wrapf(err, "ParseString %s", self.RepeatWeekdays)
	}
	repeatWeekdays := []int{}
	days.Unmarshal(&repeatWeekdays)
	err = daysCheck(repeatWeekdays, 1, 7)
	if err != nil {
		return 0, err
	}
	if len(repeatWeekdays) > limit {
		return 0, httperrors.NewInputParameterError("repeat_weekdays only contains %d days at most", limit)
	}
	return bitmap.IntArray2Uint(repeatWeekdays), nil
}

func (self SSnapshotPolicyCreateInput) GetTimePoints(limit int) (uint32, error) {
	points, err := jsonutils.ParseString(self.TimePoints)
	if err != nil {
		return 0, errors.Wrapf(err, "ParseString %s", self.TimePoints)
	}
	timepoints := []int{}
	points.Unmarshal(&timepoints)
	err = daysCheck(timepoints, 0, 23)
	if err != nil {
		return 0, err
	}
	if len(timepoints) > limit {
		return 0, httperrors.NewInputParameterError("time_points only contains %d points at most", limit)
	}
	return bitmap.IntArray2Uint(timepoints), nil
}

func daysCheck(days []int, min, max int) error {
	if len(days) == 0 {
		return nil
	}
	sort.Ints(days)

	if days[0] < min || days[len(days)-1] > max {
		return fmt.Errorf("Out of range")
	}

	for i := 1; i < len(days); i++ {
		if days[i] == days[i-1] {
			return fmt.Errorf("Has repeat day %v", days)
		}
	}
	return nil
}

type SSnapshotPolicyUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	RetentionDays *int

	RepeatWeekdays *string `json:"repeat_weekdays"`
	TimePoints     *string `json:"time_points"`
}

func (self SSnapshotPolicyUpdateInput) GetRepeatWeekdays(limit int) (uint32, error) {
	if self.RepeatWeekdays != nil {
		days, err := jsonutils.ParseString(*self.RepeatWeekdays)
		if err != nil {
			return 0, errors.Wrapf(err, "ParseString %s", *self.RepeatWeekdays)
		}
		repeatWeekdays := []int{}
		days.Unmarshal(&repeatWeekdays)
		err = daysCheck(repeatWeekdays, 1, 7)
		if err != nil {
			return 0, err
		}
		if len(repeatWeekdays) > limit {
			return 0, httperrors.NewInputParameterError("repeat_weekdays only contains %d days at most", limit)
		}
		return bitmap.IntArray2Uint(repeatWeekdays), nil
	}
	return 0, nil
}

func (self SSnapshotPolicyUpdateInput) GetTimePoints(limit int) (uint32, error) {
	if self.TimePoints != nil {
		points, err := jsonutils.ParseString(*self.TimePoints)
		if err != nil {
			return 0, errors.Wrapf(err, "ParseString %s", *self.TimePoints)
		}
		timepoints := []int{}
		points.Unmarshal(&timepoints)
		err = daysCheck(timepoints, 0, 23)
		if err != nil {
			return 0, err
		}
		if len(timepoints) > limit {
			return 0, httperrors.NewInputParameterError("time_points only contains %d points at most", limit)
		}
		return bitmap.IntArray2Uint(timepoints), nil
	}
	return 0, nil
}
