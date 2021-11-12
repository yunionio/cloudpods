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
	 * 列出历史流程实例
	 */
	type HistoricProcessInstanceListOptions struct {
		ProcessDefinitionKey string `help:"Process definition key"`
		UserId               string `help:"ID of user"`
		options.BaseListOptions
	}
	R(&HistoricProcessInstanceListOptions{}, "historic-process-instance-list", "List historic process definition", func(s *mcclient.ClientSession, args *HistoricProcessInstanceListOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		if len(args.ProcessDefinitionKey) > 0 {
			params.Add(jsonutils.NewString(args.ProcessDefinitionKey), "process_definition_key")
		}
		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}
		result, err := modules.HistoricProcessInstance.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.HistoricProcessInstance.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的历史流程实例
	 */
	type HistoricProcessInstanceShowOptions struct {
		ID string `help:"ID of the historic process instance"`
	}
	R(&HistoricProcessInstanceShowOptions{}, "historic-process-instance-show", "Show historic process definition", func(s *mcclient.ClientSession, args *HistoricProcessInstanceShowOptions) error {
		result, err := modules.HistoricProcessInstance.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HistoricProcessInstanceStaticticsOptions struct {
		ID string `help:"ID of user"`
	}
	R(&HistoricProcessInstanceStaticticsOptions{}, "historic-process-instance-statistics", "Show statistics for historic process instance", func(s *mcclient.ClientSession, args *HistoricProcessInstanceStaticticsOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.ID), "user_id")

		rst, err := modules.HistoricProcessInstance.GetStatistics(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})
}
