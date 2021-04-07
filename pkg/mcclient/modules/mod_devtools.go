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

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	DevToolCronjobs           modulebase.ResourceManager
	DevToolTemplates          modulebase.ResourceManager
	DevToolScripts            modulebase.ResourceManager
	DevToolScriptApplyRecords modulebase.ResourceManager
)

func init() {

	DevToolCronjobs = NewDevtoolManager(
		"devtool_cronjob",
		"devtool_cronjobs",
		[]string{"id", "ansible_playbook_id", "template_id", "server_id", "name", "day", "hour", "min", "sec", "interval", "start", "enabled", "created_at"},
		[]string{},
	)
	registerCompute(&DevToolCronjobs)

	DevToolTemplates = NewDevtoolManager(
		"devtool_template",
		"devtool_templates",
		[]string{"id", "name", "domain_id", "tenant_id", "day", "hour", "min", "sec", "interval", "start", "enabled", "description"},
		[]string{"is_system"},
	)
	registerCompute(&DevToolTemplates)

	DevToolScripts = NewDevtoolManager(
		"script",
		"scripts",
		[]string{"Id", "Name", "Type", "Playbook_Reference", "Max_Try_Times"},
		[]string{},
	)
	registerCompute(&DevToolScripts)
	DevToolScriptApplyRecords = NewDevtoolManager(
		"scriptapplyrecord",
		"scriptapplyrecords",
		[]string{"Script_Id", "Server_Id", "Start_Time", "End_Time", "Reason", "Status"},
		[]string{},
	)
	registerCompute(&DevToolScriptApplyRecords)
}
