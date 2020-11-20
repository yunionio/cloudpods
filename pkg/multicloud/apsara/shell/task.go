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
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TaskListOptions struct {
		TYPE   string   `help:"Task types, either ImportImage or ExportImage" choices:"ImportImage|ExportImage"`
		Task   []string `help:"Task ID"`
		Status string   `help:"Task status" choices:"Finished|Processing|Waiting|Deleted|Paused|Failed"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&TaskListOptions{}, "task-list", "List tasks", func(cli *apsara.SRegion, args *TaskListOptions) error {
		tasks, total, err := cli.GetTasks(apsara.TaskActionType(args.TYPE), args.Task, apsara.TaskStatusType(args.Status), args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(tasks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type TaskIDOptions struct {
		ID string `help:"Task ID"`
	}

	shellutils.R(&TaskIDOptions{}, "task-show", "Show task", func(cli *apsara.SRegion, args *TaskIDOptions) error {
		task, err := cli.GetTask(args.ID)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})

	shellutils.R(&TaskIDOptions{}, "cancel-task", "Cancel task", func(cli *apsara.SRegion, args *TaskIDOptions) error {
		return cli.CancelTask(args.ID)
	})

}
