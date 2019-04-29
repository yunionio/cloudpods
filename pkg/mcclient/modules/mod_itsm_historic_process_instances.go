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

var (
	HistoricProcessInstance ResourceManager
)

func init() {
	HistoricProcessInstance = NewITSMManager("historic-process-instance", "historic-process-instances",
		[]string{"id", "process_definition_key", "start_activity_id", "end_time", "duration_in_millis", "removal_time", "business_key", "end_activity_id", "process_definition_version", "delete_reason", "process_definition_id", "start_time", "start_user_id", "case_instance_id", "root_process_instance_id", "super_case_instance_id", "state", "process_definition_name", "super_process_instance_id", "tenant_id"},
		[]string{},
	)
	register(&HistoricProcessInstance)
}
