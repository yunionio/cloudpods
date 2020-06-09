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

import "time"

type TimerCreateInput struct {

	// description: 执行时间
	ExecTime time.Time
}

type CycleTimerCreateInput struct {

	// description: 周期类型
	// enum: day,week,month
	CycleType string `json:"cycle_type"`

	// description: 分(0-59)
	// example: 13
	Minute int `json:"minute"`

	// description: 时(0-23)
	// example: 13
	Hour int `json:"hour"`

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

type ScalingAlarmCreateInput struct {

	// description: 累计次数(报警次数达到此累计次数就会触发伸缩活动)
	// example: 1
	Cumulate int `json:"cumulate"`

	// description: 监控周期，单位s
	// example: 300
	Cycle int `json:"cycle"`

	// description: 监控指标
	// example: cpu
	// enum: cpu,disk_read,disk_write,flow_into,flow_out
	Indicator string `json:"indicator"`

	// description: 监控指标的取值方式(比如最大值，最小值，平均值)
	// example: max
	// enum: max,min,average
	Wrapper string `json:"wrapper"`

	// descripion: 监控指标的比较符
	// example: ge
	// enum: ge,le
	Operator string `json:"operator"`

	// description: 监控指标的取值
	// example: 3
	Value float64 `json:"value"`
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

type ScalingAlarmDetails struct {
	// description: 累计次数，只有到达累计次数的报警才会触发伸缩活动
	Cumulate int `json:"cumulate"`
	// description: 监控周期
	Cycle int `json:"cycle"`
	// description: 指标
	Indicator string `json:"indicator"`
	// description: 指标的取值方式，最大值/最小值/平均值
	Wrapper string `json:"wrapper"`
	// description: 指标和阈值之间的比较关系，>=/<=
	Operator string `json:"operator"`
	// description: 阈值
	Value float64 `json:"value"`
}
