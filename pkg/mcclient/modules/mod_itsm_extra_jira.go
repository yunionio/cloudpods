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
	ExtraUsers             modulebase.ResourceManager
	ExtraProcessDefinition modulebase.ResourceManager
	ExtraProcessInstance   modulebase.ResourceManager
	ExtraJira              modulebase.ResourceManager
)

func init() {
	ExtraUsers = NewITSMManager("extra-user", "extra-users",
		[]string{},
		[]string{},
	)

	ExtraProcessDefinition = NewITSMManager("extra-process-definition", "extra-process-definitions",
		[]string{},
		[]string{},
	)

	ExtraProcessInstance = NewITSMManager("extra-process-instance", "extra-process-instances",
		[]string{},
		[]string{},
	)

	ExtraJira = NewITSMManager("extra-jira", "extra-jiras",
		[]string{},
		[]string{},
	)

	mods := []modulebase.ResourceManager{
		ExtraJira,
		ExtraUsers,
		ExtraProcessDefinition,
		ExtraProcessInstance,
	}
	for i := range mods {

		register(&mods[i])
		registerV2(&mods[i])
	}
}
