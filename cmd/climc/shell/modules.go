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

package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func init() {
	type ModuleListOptions struct {
	}
	R(&ModuleListOptions{}, "module-list", "List all modules", func(s *mcclient.ClientSession, args *ModuleListOptions) error {
		modules, jointModules := modulebase.GetRegisterdModules()
		json := jsonutils.Marshal(modules)
		fmt.Println("Modules")
		fmt.Println(json.PrettyString())
		json2 := jsonutils.Marshal(jointModules)
		fmt.Println("Joint modules")
		fmt.Println(json2.PrettyString())
		return nil
	})
}
