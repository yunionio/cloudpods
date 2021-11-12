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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/itsm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type TaskListOptions struct {
		options.BaseListOptions
	}
	R(&TaskListOptions{}, "process-list", "List processes", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Processes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Processes.GetColumns(s))
		return nil
	})

	type ProcessShowOptions struct {
		ID string `help:"ID or Name of the process to show"`
	}
	R(&ProcessShowOptions{}, "process-show", "Show process details", func(s *mcclient.ClientSession, args *ProcessShowOptions) error {
		result, err := modules.Processes.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
