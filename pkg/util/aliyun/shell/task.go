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
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TaskListOptions struct {
		TYPE   string   `help:"Task types, either ImportImage or ExportImage" choices:"ImportImage|ExportImage"`
		Task   []string `help:"Task ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&TaskListOptions{}, "task-list", "List tasks", func(cli *aliyun.SRegion, args *TaskListOptions) error {
		tasks, total, err := cli.GetTasks(aliyun.TaskActionType(args.TYPE), args.Task, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(tasks, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
