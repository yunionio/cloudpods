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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BaremetalAgentListOptions struct {
		options.BaseListOptions
	}
	R(&BaremetalAgentListOptions{}, "agent-list", "List all agent", func(s *mcclient.ClientSession, args *BaremetalAgentListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Baremetalagents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Baremetalagents.GetColumns(s))
		return nil
	})

	type BaremetalAgentOpsOperations struct {
		ID string `help:"ID or name of agent"`
	}
	R(&BaremetalAgentOpsOperations{}, "agent-show", "Show details of an agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&BaremetalAgentOpsOperations{}, "agent-enable", "Enable agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&BaremetalAgentOpsOperations{}, "agent-disable", "Disable agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&BaremetalAgentOpsOperations{}, "agent-delete", "Delete agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
