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
	AnsiblePlaybookReference modulebase.ResourceManager
	AnsiblePlaybookInstance  modulebase.ResourceManager
)

func init() {
	AnsiblePlaybookReference = NewAnsibleManager(
		"ansibleplaybookreference",
		"ansibleplaybookreferences",
		[]string{
			"Id",
			"Name",
			"Playbook_Path",
			"Default_Params",
			"Method",
		},
		[]string{},
	)
	AnsiblePlaybookInstance = NewAnsibleManager(
		"ansibleplaybookinstance",
		"ansibleplaybookinstances",
		[]string{
			"Id",
			"Status",
			"Start_Time",
			"End_Time",
			"Output",
		},
		[]string{},
	)
	registerV2(&AnsiblePlaybookReference)
	registerV2(&AnsiblePlaybookInstance)
}
