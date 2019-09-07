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
	Instances modulebase.ResourceManager
)

func init() {
	Instances = NewITSMManager("instance", "instances",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "apply_date", "instance_id", "business_id", "procdef_key", "busi_type", "apply_unit", "client_info", "emergency", "impact", "title", "content", "start_time", "end_time", "starter", "starter_name", "instance_status", "current_approver", "approver_name", "task_name", "task_type", "task_id", "result", "login_name", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "apply_date", "instance_id", "business_id", "procdef_key", "busi_type", "apply_unit", "client_info", "emergency", "impact", "title", "content", "start_time", "end_time", "starter", "starter_name", "instance_status", "current_approver", "approver_name", "task_name", "task_type", "task_id", "result", "login_name", "common_start_string"},
	)
	register(&Instances)
}
