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
	/**
	 * 处理任务
	 */
	type TaskOperOperateOptions struct {
		ID                string `help:"ID of task"`
		APPROVAL_DECISION bool   `help:"Approval decision"`
		ApprovalProposal  string `help:"Approval proposal"`
	}
	R(&TaskOperOperateOptions{}, "process-task-operate", "Operate task", func(s *mcclient.ClientSession, args *TaskOperOperateOptions) error {
		variables_params := jsonutils.NewDict()
		if args.APPROVAL_DECISION {
			variables_params.Add(jsonutils.JSONTrue, "approved")
		} else {
			variables_params.Add(jsonutils.JSONFalse, "approved")
		}
		if len(args.ApprovalProposal) > 0 {
			variables_params.Add(jsonutils.NewString(args.ApprovalProposal), "comment")
		}
		params := jsonutils.NewDict()
		params.Add(variables_params, "variables")
		result, err := modules.ProcessTasks.Put(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 列出任务
	 */
	type TaskListOptions struct {
		UserId  string `help:"Id of user"`
		GroupId string `help:"ID of group"`
		options.BaseListOptions
	}
	R(&TaskListOptions{}, "process-task-list", "List task", func(s *mcclient.ClientSession, args *TaskListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}
		if len(args.GroupId) > 0 {
			params.Add(jsonutils.NewString(args.GroupId), "group_id")
		}
		result, err := modules.ProcessTasks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ProcessTasks.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的任务
	 */
	type TaskShowOptions struct {
		ID string `help:"ID of the task"`
	}
	R(&TaskShowOptions{}, "process-task-show", "Show task details", func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		result, err := modules.ProcessTasks.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
