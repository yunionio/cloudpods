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

package servicetree

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建新的服务树
	 */
	type ServiceTreeCreateOptions struct {
		NAME   string `help:"Name of the new service tree"`
		STRUCT string `help:"Struct of the service tree"`
		Remark string `help:"Remark or description of the new service tree"`
	}
	R(&ServiceTreeCreateOptions{}, "servicetree-create", "Create a service tree", func(s *mcclient.ClientSession, args *ServiceTreeCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "service_tree_name")
		params.Add(jsonutils.NewString(args.STRUCT), "service_tree_struct")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}
		serviceTree, err := modules.ServiceTrees.Create(s, params)
		if err != nil {
			return err
		}
		printObject(serviceTree)
		return nil
	})

	/**
	 * 删除一个服务树
	 */
	type ServiceTreeDeleteOptions struct {
		ID string `help:"ID of alarm template"`
	}
	R(&ServiceTreeDeleteOptions{}, "servicetree-delete", "Delete a service tree", func(s *mcclient.ClientSession, args *ServiceTreeDeleteOptions) error {
		serviceTree, e := modules.ServiceTrees.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(serviceTree)
		return nil
	})

	/**
	 * 修改服务树的配置信息
	 */
	type ServiceTreeUpdateOptions struct {
		ID     string `help:"ID of the service tree"`
		Name   string `help:"Name of the service-tree"`
		Struct string `help:"Struct of the service-tree"`
		Remark string `help:"Remark or description of the service-tree"`
	}
	R(&ServiceTreeUpdateOptions{}, "servicetree-update", "Update a service tree", func(s *mcclient.ClientSession, args *ServiceTreeUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "service_tree_name")
		}
		if len(args.Struct) > 0 {
			params.Add(jsonutils.NewString(args.Struct), "service_tree_struct")
		}
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		serviceTree, err := modules.ServiceTrees.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(serviceTree)
		return nil
	})

	/**
	 * 列出服务树
	 */
	type ServiceTreeListOptions struct {
		options.BaseListOptions
	}
	R(&ServiceTreeListOptions{}, "servicetree-list", "List all service tree", func(s *mcclient.ClientSession, args *ServiceTreeListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.ServiceTrees.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ServiceTrees.GetColumns(s))
		return nil
	})

	/**
	 * 查看服务树配置详情
	 */
	type ServiceTreeShowOptions struct {
		ID string `help:"ID of the service-tree"`
	}
	R(&ServiceTreeShowOptions{}, "servicetree-show", "Show service tree details", func(s *mcclient.ClientSession, args *ServiceTreeShowOptions) error {
		result, err := modules.ServiceTrees.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
