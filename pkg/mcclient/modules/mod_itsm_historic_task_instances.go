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
	HistoricTaskInstances modulebase.ResourceManager
)

func init() {
	HistoricTaskInstances = NewITSMManager("historic-task-instance", "historic-task-instances",
		[]string{"id", "name", "parent_task_id", "duration_in_millis", "description", "case_execution_id", "removal_time", "delete_reason", "follow_up_date", "execution_id", "activity_instance_id", "root_process_instance_id", "owner", "process_definition_key", "end_time", "due_date", "super_case_instance_id", "priority", "process_definition_id", "start_time", "case_definition_key", "case_instance_id", "process_instance_id", "case_definition_id", "assignee", "task_definition_key"},
		[]string{},
	)
	register(&HistoricTaskInstances)
}
