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

package devtool

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	DevToolCronjobs           modulebase.ResourceManager
	DevToolTemplates          modulebase.ResourceManager
	DevToolScripts            modulebase.ResourceManager
	DevToolScriptApplyRecords modulebase.ResourceManager
	DevToolSshInfos           modulebase.ResourceManager
	DevToolServiceUrls        modulebase.ResourceManager
)

func init() {

	DevToolCronjobs = modules.NewDevtoolManager(
		"devtool_cronjob",
		"devtool_cronjobs",
		[]string{"id", "ansible_playbook_id", "template_id", "server_id", "name", "day", "hour", "min", "sec", "interval", "start", "enabled", "created_at"},
		[]string{},
	)
	modules.Register(&DevToolCronjobs)

	DevToolTemplates = modules.NewDevtoolManager(
		"devtool_template",
		"devtool_templates",
		[]string{"id", "name", "domain_id", "tenant_id", "day", "hour", "min", "sec", "interval", "start", "enabled", "description"},
		[]string{"is_system"},
	)
	modules.Register(&DevToolTemplates)

	DevToolScripts = modules.NewDevtoolManager(
		"script",
		"scripts",
		[]string{"Id", "Name", "Type", "Playbook_Reference", "Max_Try_Times"},
		[]string{},
	)
	modules.Register(&DevToolScripts)

	DevToolScriptApplyRecords = modules.NewDevtoolManager(
		"scriptapplyrecord",
		"scriptapplyrecords",
		[]string{"Script_Id", "Server_Id", "Start_Time", "End_Time", "Reason", "Status"},
		[]string{},
	)
	modules.Register(&DevToolScriptApplyRecords)

	DevToolSshInfos = modules.NewDevtoolManager(
		"sshinfo",
		"sshinfos",
		[]string{"Id", "Server_Id", "Server_Name", "Server_Hypervisor", "Forward_Id", "User", "Host", "Port", "Need_Clean", "Failed_Reason"},
		[]string{},
	)
	modules.Register(&DevToolSshInfos)

	DevToolServiceUrls = modules.NewDevtoolManager(
		"serviceurl",
		"serviceurls",
		[]string{"Id", "Service", "Server_Id", "Url", "Server_Ansible_Info", "Failed_Reason"},
		[]string{},
	)
	modules.Register(&DevToolServiceUrls)
}
