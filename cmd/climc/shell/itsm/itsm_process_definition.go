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
	 * 删除指定ID的流程定义
	 */
	type ProcessDefinitionDeleteOptions struct {
		ID      string `help:"ID of process definition, deletes the process definition which belongs to the given process definition id"`
		Cascade bool   `help:"Whether to cascade delete process instance, if set to true, all process instances (including) history are deleted" required:"true" choices:"true|false"`
	}
	R(&ProcessDefinitionDeleteOptions{}, "process-definition-delete", "Delete process definition by ID", func(s *mcclient.ClientSession, args *ProcessDefinitionDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.Cascade {
			params.Add(jsonutils.JSONTrue, "cascade")
		} else {
			params.Add(jsonutils.JSONFalse, "cascade")
		}
		result, e := modules.ProcessDefinitions.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	/**
	 * 修改指定的流程定义的激活/挂起状态
	 */
	type NodealertUpdateOptions struct {
		ID    string `help:"ID of process definition"`
		state string `help:"State of the process definition" choices:"activate|suspend"`
	}
	R(&NodealertUpdateOptions{}, "process-definition-change-state", "Update the state of the process definition", func(s *mcclient.ClientSession, args *NodealertUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.state), "state")
		result, err := modules.ProcessDefinitions.Put(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 列出流程定义
	 */
	type ProcessDefinitionListOptions struct {
		ProcessDefinitionKey string `help:"Key of process definition"`
		UserId               string `help:"ID of user who start the process definition"`
		options.BaseListOptions
	}
	R(&ProcessDefinitionListOptions{}, "process-definition-list", "List process definition", func(s *mcclient.ClientSession, args *ProcessDefinitionListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		if len(args.ProcessDefinitionKey) > 0 {
			params.Add(jsonutils.NewString(args.ProcessDefinitionKey), "process_definition_key")
		}
		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}
		result, err := modules.ProcessDefinitions.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ProcessDefinitions.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的流程定义
	 */
	type ProcessDefinitionShowOptions struct {
		ID string `help:"ID of the process definition"`
	}
	R(&ProcessDefinitionShowOptions{}, "process-definition-show", "Show process definition", func(s *mcclient.ClientSession, args *ProcessDefinitionShowOptions) error {
		result, err := modules.ProcessDefinitions.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
