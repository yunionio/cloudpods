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

package itsm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Operations modulebase.ResourceManager
)

func init() {
	Operations = modules.NewITSMManager("operation", "operations",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "operate_type", "template_id", "device_code", "ip_address", "cpu", "memery", "disk", "resource_sum", "network", "schedule", "login_name", "display_name", "resource_ids", "instance", "task", "log_list", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "operate_type", "template_id", "device_code", "ip_address", "cpu", "memery", "disk", "resource_sum", "network", "schedule", "login_name", "display_name", "resource_ids", "instance", "task", "log_list", "common_start_string"},
	)
	modules.Register(&Operations)
}
