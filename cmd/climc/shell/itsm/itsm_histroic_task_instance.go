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
	/**
	 * 列出历史任务实例
	 */
	type HistoricProcessInstanceListOptions struct {
		ProcessInstanceId string `help:"ID of process instance"`
		UserId            string `help:"ID of user"`
		options.BaseListOptions
	}
	R(&HistoricProcessInstanceListOptions{}, "historic-task-instance-list", "List historic process instance", func(s *mcclient.ClientSession, args *HistoricProcessInstanceListOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		if len(args.ProcessInstanceId) > 0 {
			params.Add(jsonutils.NewString(args.ProcessInstanceId), "process_instance_id")
		}
		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}
		result, err := modules.HistoricTaskInstances.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.HistoricTaskInstances.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的历史任务实例
	 */
	type HistoricProcessInstanceShowOptions struct {
		ID string `help:"ID of the historic task instance"`
	}
	R(&HistoricProcessInstanceShowOptions{}, "historic-task-instance-show", "Show historic task instance", func(s *mcclient.ClientSession, args *HistoricProcessInstanceShowOptions) error {
		result, err := modules.HistoricTaskInstances.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
