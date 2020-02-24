package compute

import "time"

type ScalingTimerCreateInput struct {

	// description: execute time
	ExecTime time.Time
}

type ScalingCycleTimerCreateInput struct {

	// description: cycle type
	// enum: day,week,month
	CycleType string

	// description: minute(0-59)
	// example: 13
	Minute int

	// description: hour(0-23)
	// example: 13
	Hour int

	// description: repeat weekdays; 1-7, 1: Monday, 7: Sunday
	// example: [1,3,5,7]
	WeekDays []int

	// description: repeat monthdays; 1-31
	// example: [1,4,31]
	MonthDays []int

	// description: end time
	EndTime time.Time
}

type ScalingAlarmCreateInput struct {

	// description: The scaling activity will be triggered after cumulate consecutive alarms
	// example: 1
	Cumulate int

	// description: Monitoring indicators
	// example: cpu
	Indicator string

	// description: Wrapper instruct how to calculate collective data based on individual data
	// example: max
	Wrapper string

	// descritpion: Operator represent the relation of Indicator and value
	// example: ge
	Operator string

	// descritption: Threshold value(percentage)
	// example: 3
	Value int
}

type ScalingTimerDetails struct {
	// 执行时间
	ExecTime time.Time `json:"exec_time"`
}

type ScalingCycleTimerDetails struct {
	// 周期类型：按天/周/月
	CycleType string `json:"cycle_type"`
	// 分钟
	Minute int `json:"minute"`
	// 小时
	Hour int `json:"hour"`
	// 每周的几天
	WeekDays []int `json:"week_day"`
	// 每月的几天
	MonthDays []int `json:"month_day"`
	// 此周期任务的截止时间
	EndTime time.Time `json:"end_time"`
}

type ScalingAlarmDetails struct {
	// 累计次数，只有到达累计次数的报警才会触发伸缩活动
	Cumulate int `json:"cumulate"`
	// 指标
	Indicator string `json:"indicator"`
	// 指标的取值方式，最大值/最小值/平均值
	Wrapper string `json:"wrapper"`
	// 指标和阈值之间的比较关系，>=/<=
	Operator string `json:"operator"`
	// 阈值
	Value int `json:"value"`
}
