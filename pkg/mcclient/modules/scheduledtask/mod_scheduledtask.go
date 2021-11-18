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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	ScheduledTask         modulebase.ResourceManager
	ScheduledTaskActivity modulebase.ResourceManager
)

func init() {
	ScheduledTask = modules.NewScheduledtaskManager("scheduledtask", "scheduledtasks",
		[]string{"ID", "Name", "Scheduled_Type", "Timer", "Cycle_Timer", "Resource_Type", "Operation", "Label_Type", "Labels", "Timer_Desc"}, []string{},
	)
	ScheduledTaskActivity = modules.NewScheduledtaskManager("scheudledtaskactivity", "scheduledtaskactivities",
		[]string{"ID", "Status", "Scheduled_Task_Id", "Start_Time", "End_Time", "Reason"}, []string{},
	)
	modules.Register(&ScheduledTask)
	modules.Register(&ScheduledTaskActivity)
}
