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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	ScalingGroup    modulebase.ResourceManager
	ScalingPolicy   modulebase.ResourceManager
	ScalingActivity modulebase.ResourceManager
)

func init() {
	ScalingGroup = NewComputeManager("scalinggroup", "scalinggroups",
		[]string{"ID", "Name", "Hypervisor", "Cloudregion_ID", "Network_ID", "Min_Instance_Number",
			"Max_Instance_Number", "Desire_Instance_Number", "Guest_Template_ID", "Loadbalancer_ID", "Group_ID", "Enabled",
			"Expansion_Principle", "Shrink_Principle"},
		[]string{},
	)
	ScalingPolicy = NewComputeManager("scalingpolicy", "scalingpolicies",
		[]string{"ID", "Name", "Timer", "Cycle_Timer", "Alarm", "Action", "Number", "Unit", "Cooling_Time"},
		[]string{},
	)
	ScalingActivity = NewComputeManager("scalingactivity", "scalingactivities",
		[]string{"ID", "Name", "Instance_Number", "Trigger_Desc", "Action_Desc", "Status", "Start_Time",
			"End_Time", "Reason"},
		[]string{},
	)
	registerCompute(&ScalingGroup)
	registerCompute(&ScalingPolicy)
	registerCompute(&ScalingActivity)
}
