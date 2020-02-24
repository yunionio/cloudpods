package compute

import "yunion.io/x/onecloud/pkg/apis"

type ScalingPolicyDetails struct {
	apis.StandaloneResourceDetails
	SScalingPolicy
	// 定时方式触发
	Timer ScalingTimerDetails `json:"timer"`
	// 周期方式触发
	CycleTimer ScalingCycleTimerDetails `json:"cycle_timer"`
	//  告警方式触发
	Alarm ScalingAlarmDetails `json:"alarm"`
}

type ScalingPolicyCreateInput struct {
	apis.StandaloneResourceCreateInput

	// description: trigger type
	// enum: timing,cycle,alarm
	TriggerType string

	Timer      ScalingTimerCreateInput
	CycleTimer ScalingCycleTimerCreateInput
	Alarm      ScalingAlarmCreateInput

	// desciption: action of scaling activity
	// enum: add,remove,set
	// example: add
	Action string

	// desciption: number
	// example: 2
	Number int

	// desciption: Unit of Number
	// enum: s,%
	// example: s
	Unit string

	// desciption: Scaling activity triggered by alarms will be rejected during this period about CoolingTime
	// example: 300
	CoolingTime int
}

type ScalingPolicyListInput struct {
	apis.StatusStandaloneResourceListInput

	// description: scaling group
	// example: sg-test
	ScalingGroup string

	// description: trigger type
	// enum: timing,cycel,alarm
	// example: alarm
	TriggerType string
}
