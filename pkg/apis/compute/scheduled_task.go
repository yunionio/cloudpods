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

type LabelDetail struct {
	Label        string    `json:"label"`
	IsolatedTime time.Time `json:"isolated_time"`
}

type ScheduledTaskListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	// description: resource type
	// example: server
	// enum: server
	ResourceType string `json:"resource_type"`

	// description: label type
	// example: tag
	LabelType string `json:"label_type"`

	// description: operation
	// example: stop
	// enum: start,stop,restart
	Operation string `json:"operation"`
}

type ScheduledTaskCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	// description: scheduled type
	// enum: cycle,timing
	// example: timing
	ScheduledType string                `json:"scheduled_type"`
	Timer         TimerCreateInput      `json:"timer"`
	CycleTimer    CycleTimerCreateInput `json:"cycle_timer"`

	// description: resource type
	// enum: server
	// example: server
	ResourceType string `json:"resource_type"`
	// description: operation
	// enum: start,stop,restart
	// example: stop
	Operation string `json:"operation"`
	// description: label type
	// enum: tag,id
	// example: id
	LabelType string `json:"label_type"`
	// description: labels
	// example: {g-12345}
	Labels []string
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
