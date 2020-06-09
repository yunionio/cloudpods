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

type ScalingPolicyDetails struct {
	apis.VirtualResourceDetails
	ScalingGroupResourceInfo
	SScalingPolicy
	// 定时方式触发
	Timer TimerDetails `json:"timer"`
	// 周期方式触发
	CycleTimer CycleTimerDetails `json:"cycle_timer"`
	//  告警方式触发
	Alarm ScalingAlarmDetails `json:"alarm"`
}

type ScalingPolicyCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	// description: scaling_group ID or Name
	// example: sg-test-one
	ScalingGroup string `json:"scaling_group"`

	// swagger: ignore
	ScalingGroupId string `json:"scaling_group_id"`

	// description: trigger type
	// enum: timing,cycle,alarm
	TriggerType string `json:"trigger_type"`

	Timer      TimerCreateInput        `json:"timer"`
	CycleTimer CycleTimerCreateInput   `json:"cycle_timer"`
	Alarm      ScalingAlarmCreateInput `json:"alarm"`

	// desciption: 伸缩策略的行为(增加还是删除或者调整为)
	// enum: add,remove,set
	// example: add
	Action string `json:"action"`

	// desciption: 实例的数量
	// example: 2
	Number int `json:"number"`

	// desciption: 实例数量的单位
	// enum: s,%
	// example: s
	Unit string `json:"unit"`

	// desciption: Scaling activity triggered by alarms will be rejected during this period about CoolingTime
	// example: 300
	CoolingTime int `json:"cooling_time"`
}

type ScalingPolicyListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	// description: scaling group
	// example: sg-test
	ScalingGroupFilterListInput

	// description: trigger type
	// enum: timing,cycel,alarm
	// example: alarm
	TriggerType string `json:"trigger_type"`
}
