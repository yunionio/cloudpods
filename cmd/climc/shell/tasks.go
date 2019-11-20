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
	type TaskListOptions struct {
		options.BaseListOptions
	}
	R(&TaskListOptions{}, "task-list", "List taskman", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Tasks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Tasks.GetColumns(s))
		return nil
	})

	type RegionTaskListOptions struct {
		ObjName  string `help:"object name"`
		ObjId    string `help:"object id"`
		TaskName string `help:"task name"`
	}
	R(&RegionTaskListOptions{}, "region-task-list", "List tasks on region server", func(s *mcclient.ClientSession, args *RegionTaskListOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.ComputeTasks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ComputeTasks.GetColumns(s))
		return nil
	})

	type TaskShowOptions struct {
		ID string `help:"ID or name of the task"`
	}
	R(&TaskShowOptions{}, "region-task-show", "Show details of a region task", func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		result, err := modules.ComputeTasks.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
