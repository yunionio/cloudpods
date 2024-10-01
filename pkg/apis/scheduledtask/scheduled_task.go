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

package scheduledtask

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type ScheduledTaskDetails struct {
	apis.VirtualResourceDetails
	SScheduledTask

	// 描述
	TimerDesc string `json:"timer_desc"`
	// 定时方式触发
	Timer TimerDetails `json:"timer"`
	// 周期方式触发
	CycleTimer CycleTimerDetails `json:"cycle_timer"`
	// 绑定的所有标示
	Labels       []string      `json:"labels,allowempty"`
	LabelDetails []LabelDetail `json:"label_details,allowempty"`
}

type TimerDetails struct {
	// description: 执行时间
	ExecTime time.Time `json:"exec_time"`
}

type CycleTimerDetails struct {
	// description: 周期类型：按天/周/月
	CycleType string `json:"cycle_type"`
	// description: 分钟
	Minute int `json:"minute"`
	// description: 小时
	Hour int `json:"hour"`
	// description: 每周的几天
	WeekDays []int `json:"week_days"`
	// description: 每月的几天
	MonthDays []int `json:"month_days"`
	// description: 此周期任务的开始时间
	StartTime time.Time `json:"start_time"`
	// description: 此周期任务的截止时间
	EndTime time.Time `json:"end_time"`
}

type LabelDetail struct {
	Label        string    `json:"label"`
	IsolatedTime time.Time `json:"isolated_time"`
}

type ScheduledTaskListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	// description: resource type
	// example: server
	// enum: ["server"]
	ResourceType string `json:"resource_type"`

	// description: label type
	// example: tag
	LabelType string `json:"label_type"`

	// description: filter scheduledtask binded with label
	// example: g-12345
	Label string `json:"label"`

	// description: operation
	// example: stop
	// enum: ["start","stop","restart"]
	Operation string `json:"operation"`
}

type ScheduledTaskCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	// description: scheduled type
	// enum: ["cycle","timing"]
	// example: timing
	ScheduledType string                `json:"scheduled_type"`
	Timer         TimerCreateInput      `json:"timer"`
	CycleTimer    CycleTimerCreateInput `json:"cycle_timer"`

	// description: resource type
	// enum: ["server"]
	// example: server
	ResourceType string `json:"resource_type"`
	// description: operation
	// enum: ["start","stop","restart"]
	// example: stop
	Operation string `json:"operation"`
	// description: label type
	// enum: ["tag","id"]
	// example: id
	LabelType string `json:"label_type"`
	// description: labels
	// example: {g-12345}
	Labels []string
}

type TimerCreateInput struct {

	// description: 执行时间
	ExecTime time.Time
}

type CycleTimerCreateInput struct {

	// description: 周期类型
	// enum: ["hour","day","week","month"]
	CycleType string `json:"cycle_type"`

	// description: 分(0-59)
	// example: 13
	Minute int `json:"minute"`

	// description: 时(0-23)
	// example: 13
	Hour int `json:"hour"`

	// 频率为小时或天时启用，泛指间隔单位
	// example: 2
	CycleNum int `json:"cycle_num"`

	// description: 每周的周几; 1-7, 1: Monday, 7: Sunday
	// example: [1,3,5,7]
	WeekDays []int `json:"week_days"`

	// description: 每月的哪几天; 1-31
	// example: [1,4,31]
	MonthDays []int `json:"month_days"`

	// description: 开始时间
	StartTime time.Time `json:"start_time"`

	// description: 截止时间
	EndTime time.Time `json:"end_time"`
}

type ScheduledTaskResourceInfo struct {
	// description: 定时任务名称
	// example: st-nihao
	ScheduledTask string `json:"scheduled_task"`

	// description: 定时任务ID
	// example: 1234
	ScheduledTaskId string `json:"scheduled_task_id"`
}

type ScheduledTaskFilterListInput struct {
	// description: 定时任务 Id or Name
	// example: st-1234
	ScheduledTask string `json:"scheduled_task"`
}

type ScheduledTaskActivityDetails struct {
	apis.StatusStandaloneResourceDetails
	SScheduledTaskActivity
}

type ScheduledTaskActivityListInput struct {
	apis.StatusStandaloneResourceListInput
	// description: 定时任务 ID or Name
	// example: st-11212
	ScheduledTask string `json:"scheduled_task"`
}

type ScheduledTaskSetLabelsInput struct {
	Labels []string `json:"labels"`
}

type ScheduledTaskTriggerInput struct {
}
