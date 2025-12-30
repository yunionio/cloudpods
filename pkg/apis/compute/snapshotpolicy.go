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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	SNAPSHOT_POLICY_APPLY         = "applying"
	SNAPSHOT_POLICY_APPLY_FAILED  = "apply_failed"
	SNAPSHOT_POLICY_CANCEL        = "canceling"
	SNAPSHOT_POLICY_CANCEL_FAILED = "cancel_failed"

	SNAPSHOT_POLICY_TYPE_DISK   = "disk"
	SNAPSHOT_POLICY_TYPE_SERVER = "server"
)

type SnapshotPolicyDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SSnapshotPolicy

	// 绑定磁盘数量
	BindingDiskCount int `json:"binding_disk_count"`
	// 绑定主机数量
	BindingResourceCount int `json:"binding_resource_count"`
	// 快照数量
	SnapshotCount int `json:"snapshot_count"`
}

type SSnapshotPolicyCreateInput struct {
	apis.VirtualResourceCreateInput
	CloudproviderResourceInput
	CloudregionResourceInput

	// 自动快照保留时长 -1: 表示永久保留
	RetentionDays int `json:"retention_days"`
	// 自动快照保留数量
	// 此配置优先级高于 RetentionDays
	RetentionCount int `json:"retention_count"`
	// 快照类型, 目前支持 disk, server
	// disk: 自动磁盘快照策略， 只能关联磁盘
	// server: 自动主机快照策略, 只能关联主机
	Type string `json:"type"`
	// 创建自动快照策略的执行周期
	// 1. 周一到周日
	// 2. 1~7, 1: Monday, 7: Sunday
	RepeatWeekdays []int `json:"repeat_weekdays"`
	// 创建自动快照策略的执行时间
	// 0:00~23:59 每天, 0~23 每小时
	// 创建自动快照策略的时间必须与 RepeatWeekdays 对应的创建周期相一致
	TimePoints []int `json:"time_points"`
}

func (self *SSnapshotPolicyCreateInput) Validate() error {
	if len(self.RepeatWeekdays) == 0 {
		return httperrors.NewMissingParameterError("repeat_weekdays")
	}

	repeatDays := []int{}
	for _, day := range self.RepeatWeekdays {
		if day < 1 || day > 7 {
			return httperrors.NewOutOfRangeError("repeat_weekdays out of range 1-7")
		}
		if !utils.IsInArray(day, repeatDays) {
			repeatDays = append(repeatDays, day)
		}
	}
	self.RepeatWeekdays = repeatDays

	if len(self.TimePoints) == 0 {
		return httperrors.NewMissingParameterError("time_points")
	}

	points := []int{}
	for _, point := range self.TimePoints {
		if point < 0 || point > 23 {
			return httperrors.NewOutOfRangeError("time_points out of range 0-23")
		}
		if !utils.IsInArray(point, points) {
			points = append(points, point)
		}
	}
	self.TimePoints = points
	return nil
}

type SSnapshotPolicyUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	// 自动快照保留时长, -1: 表示永久保留
	RetentionDays *int
	// 自动快照保留数量
	// 此配置优先级高于 RetentionDays
	RetentionCount *int

	// 自动快照执行周期
	// 1. 周一到周日
	// 2. 1~7, 1: Monday, 7: Sunday
	RepeatWeekdays *[]int `json:"repeat_weekdays"`
	// 自动快照执行时间
	// 0:00~23:59 每天, 0~23 每小时
	// 创建自动快照策略的时间必须与 RepeatWeekdays 对应的创建周期相一致
	TimePoints *[]int `json:"time_points"`
}

func (self *SSnapshotPolicyUpdateInput) Validate() error {
	if self.RepeatWeekdays != nil {
		repeatDays := []int{}
		for _, day := range *self.RepeatWeekdays {
			if day < 1 || day > 7 {
				return httperrors.NewOutOfRangeError("repeat_weekdays out of range 1-7")
			}
			if !utils.IsInArray(day, repeatDays) {
				repeatDays = append(repeatDays, day)
			}
		}
		self.RepeatWeekdays = &repeatDays
	}

	if self.TimePoints != nil {
		points := []int{}
		for _, point := range *self.TimePoints {
			if point < 0 || point > 23 {
				return httperrors.NewOutOfRangeError("time_points out of range 0-23")
			}
			if !utils.IsInArray(point, points) {
				points = append(points, point)
			}
		}
		self.TimePoints = &points
	}
	return nil
}

type SnapshotPolicyDisksInput struct {
	// 磁盘ID列表
	// 快照策略类型需要时disk
	Disks []string `json:"disk"`
}

type SnapshotPolicyResourcesInput struct {
	// 绑定的资源列表
	// 目前仅支持绑定主机
	Resources []struct {
		// 绑定资源ID
		Id string `json:"id"`
		// server: 主机, disk: 磁盘
		Type string `json:"type"`
	} `json:"resources"`
}

type RepeatWeekdays []int

func (days RepeatWeekdays) String() string {
	return jsonutils.Marshal(days).String()
}

func (days RepeatWeekdays) IsZero() bool {
	return len(days) == 0
}

type TimePoints []int

func (points TimePoints) String() string {
	return jsonutils.Marshal(points).String()
}

func (points TimePoints) IsZero() bool {
	return len(points) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&RepeatWeekdays{}), func() gotypes.ISerializable {
		return &RepeatWeekdays{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&TimePoints{}), func() gotypes.ISerializable {
		return &TimePoints{}
	})

}
