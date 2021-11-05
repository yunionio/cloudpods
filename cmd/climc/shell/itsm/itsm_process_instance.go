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
	 * 创建流程实例（发起流程）
	 */
	type ProcessInstanceCreateOptions struct {
		PROCESS_DEFINITION_KEY string `help:"Key of the process definition"`
		INITIATOR              string `help:"ID of user who launch the process instance"`
		SERVER_CREATE_PARAMTER string `help:"Onecloud server create spec json string"`
		UserId                 string `help:"ID of user"`
		RoleId                 string `help:"ID of role"`
		ProjectId              string `help:"ID of project"`
		Percent                string `help:"Great than Percent of nrOfCompletedInstances/nrOfInstances"`
	}
	R(&ProcessInstanceCreateOptions{}, "process-instance-create", "Create process instance", func(s *mcclient.ClientSession, args *ProcessInstanceCreateOptions) error {
		variables_params := jsonutils.NewDict()
		variables_params.Add(jsonutils.NewString(args.PROCESS_DEFINITION_KEY), "process_definition_key")
		variables_params.Add(jsonutils.NewString(args.INITIATOR), "initiator")
		variables_params.Add(jsonutils.NewString(args.SERVER_CREATE_PARAMTER), "server-create-paramter")
		if len(args.UserId) > 0 {
			variables_params.Add(jsonutils.NewString(args.UserId), "user-id")
		} else {
			if len(args.RoleId) > 0 {
				variables_params.Add(jsonutils.NewString(args.RoleId), "role-id")
			}
			if len(args.ProjectId) > 0 {
				variables_params.Add(jsonutils.NewString(args.ProjectId), "project-id")
			}
		}
		if len(args.Percent) > 0 {
			variables_params.Add(jsonutils.NewString(args.Percent), "percent")
		}
		params := jsonutils.NewDict()
		params.Add(variables_params, "variables")
		result, err := modules.ProcessInstance.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 删除指定ID的流程实例
	 */
	type ProcessInstanceDeleteOptions struct {
		ID           string `help:"ID of process process"`
		DeleteReason string `help:"Reason why delete the process instance" required:"true" choices:"true|false"`
	}
	R(&ProcessInstanceDeleteOptions{}, "process-instance-delete", "Delete process instance by ID", func(s *mcclient.ClientSession, args *ProcessInstanceDeleteOptions) error {
		params := jsonutils.NewDict()
		if len(args.DeleteReason) > 0 {
			params.Add(jsonutils.NewString(args.DeleteReason), "delete_reason")
		}
		result, e := modules.ProcessInstance.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	/**
	 * 列出流程实例
	 */
	type ProcessInstanceListOptions struct {
		ProcessDefinitionId string `help:"ID of process definition"`
		options.BaseListOptions
	}
	R(&ProcessInstanceListOptions{}, "process-instance-list", "List process instance", func(s *mcclient.ClientSession, args *ProcessInstanceListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		if len(args.ProcessDefinitionId) > 0 {
			params.Add(jsonutils.NewString(args.ProcessDefinitionId), "process_definition_id")
		}
		result, err := modules.ProcessInstance.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ProcessInstance.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的流程实例
	 */
	type ProcessInstanceShowOptions struct {
		ID string `help:"ID of the process instance"`
	}
	R(&ProcessInstanceShowOptions{}, "process-instance-show", "Show process instance", func(s *mcclient.ClientSession, args *ProcessInstanceShowOptions) error {
		result, err := modules.ProcessInstance.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
