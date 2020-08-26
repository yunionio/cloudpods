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
	MonitorTemplates      modulebase.ResourceManager
	MonitorTemplateInputs modulebase.JointResourceManager
)

func init() {
	MonitorTemplates = NewServiceTreeManager("monitor_template", "monitor_templates",
		[]string{"ID", "monitor_template_name", "monitor_template_desc", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	MonitorTemplateInputs = NewJointMonitorManager(
		"monitorInfo",
		"monitorInfos",
		[]string{"ID", "monitor_template_id", "monitor_name", "monitor_conf_value", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{},
		&MonitorTemplates,
		&MonitorInputs)

	register(&MonitorTemplates)

	register(&MonitorTemplateInputs)
}
