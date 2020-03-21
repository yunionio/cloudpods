package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	ScalingGroup    modulebase.ResourceManager
	ScalingPolicy   modulebase.ResourceManager
	ScalingActivity modulebase.ResourceManager
)

func init() {
	ScalingGroup = NewComputeManager("scalinggroup", "scalinggroups",
		[]string{"ID", "Name", "Hypervisor", "CloudregionID", "NetworkID", "MinInstanceNumber", "MaxInstanceNumber",
			"DesireInstanceNumber", "GuestTemplateID", "LoadbalancerID", "GroupID", "Enabled", "ExpansionPrinciple", "ShrinkPrinciple"},
		[]string{},
	)
	ScalingPolicy = NewComputeManager("scalingpolicy", "scalingpolicies",
		[]string{"ID", "Name", "Timer", "CycleTimer", "Alarm", "Action", "Number", "Unit", "CoolingTime"},
		[]string{},
	)
	ScalingActivity = NewComputeManager("scalingactivity", "scalingactivities",
		[]string{"ID", "Name", "InstanceNumber", "TriggerDesc", "ActionDesc", "Status", "StartTime", "EndTime", "Reason"},
		[]string{},
	)
	registerCompute(&ScalingGroup)
	registerCompute(&ScalingPolicy)
	registerCompute(&ScalingActivity)
}
