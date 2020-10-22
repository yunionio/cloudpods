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

import "yunion.io/x/onecloud/pkg/apis"

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
