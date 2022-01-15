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
	"yunion.io/x/onecloud/pkg/multicloud/nutanix"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TaskListOptions struct {
	}
	shellutils.R(&TaskListOptions{}, "task-list", "list task", func(cli *nutanix.SRegion, args *TaskListOptions) error {
		tasks, err := cli.GetTasks()
		if err != nil {
			return err
		}
		printList(tasks, 0, 0, 0, []string{})
		return nil
	})

	type TaskIdOptions struct {
		ID string
	}

	shellutils.R(&TaskIdOptions{}, "task-show", "show task", func(cli *nutanix.SRegion, args *TaskIdOptions) error {
		task, err := cli.GetTask(args.ID)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})

}
