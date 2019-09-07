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
	Processlogs modulebase.ResourceManager
)

func init() {
	Processlogs = NewITSMManager("processlog", "processlogs",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "receive_time", "task_receiver", "task_operator", "task_status", "task_name", "task_type", "task_id", "handle_time", "operate_result", "operate_advice", "log_order", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "receive_time", "task_receiver", "task_operator", "task_status", "task_name", "task_type", "task_id", "handle_time", "operate_result", "operate_advice", "log_order", "common_start_string"},
	)
	register(&Processlogs)
}
