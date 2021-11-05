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
	 * 使用文件创建部署
	 */
	type DeploymentCreateOptions struct {
		FILE     string `help:"The local bpmn filename to Upload"`
		TenantID string `help:"ID of tenant"`
	}
	R(&DeploymentCreateOptions{}, "process-deployment-create", "Create deployment by upload BPMN file", func(s *mcclient.ClientSession, args *DeploymentCreateOptions) error {
		// TODO
		return nil
	})

	/**
	 * 删除指定ID的部署
	 */
	type DeploymentDeleteOptions struct {
		ID      string `help:"ID of process deployment"`
		Cascade bool   `help:"Whether to cascade delete process instance, history process instance and jobs" required:"true" choices:"true|false"`
	}
	R(&DeploymentDeleteOptions{}, "process-deployment-delete", "Delete process deployment by ID", func(s *mcclient.ClientSession, args *DeploymentDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.Cascade {
			params.Add(jsonutils.JSONTrue, "cascade")
		} else {
			params.Add(jsonutils.JSONFalse, "cascade")
		}
		result, e := modules.ProcessDeployments.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	/**
	 * 列出部署
	 */
	type DeploymentListOptions struct {
		options.BaseListOptions
	}
	R(&DeploymentListOptions{}, "process-deployment-list", "List process deployment", func(s *mcclient.ClientSession, args *DeploymentListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.ProcessDeployments.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ProcessDeployments.GetColumns(s))
		return nil
	})

	/**
	 * 查看指定ID的部署
	 */
	type DeploymentShowOptions struct {
		ID string `help:"ID of the process deployment"`
	}
	R(&DeploymentShowOptions{}, "process-deployment-show", "Show deployment", func(s *mcclient.ClientSession, args *DeploymentShowOptions) error {
		result, err := modules.ProcessDeployments.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
