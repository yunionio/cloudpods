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
	 * 向指定的服务树节点添加机器
	 */
	type ServiceHostCreateOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to add to"`
	}
	R(&ServiceHostCreateOptions{}, "servicehost-create", "Add host to tree-node", func(s *mcclient.ClientSession, args *ServiceHostCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.HOST_NAME) > 0 {
			for _, f := range args.HOST_NAME {
				tmpObj := jsonutils.NewDict()
				tmpObj.Add(jsonutils.NewString(f), "host_name")
				arr.Add(tmpObj)
			}
		}
		params.Add(arr, "service_hosts")

		rst, err := modules.ServiceHosts.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 从服务树节点删除机器
	 */
	type ServiceHostDeleteOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to remove from"`
	}
	R(&ServiceHostDeleteOptions{}, "servicehost-delete", "Remove host from tree-node", func(s *mcclient.ClientSession, args *ServiceHostDeleteOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.HOST_NAME) > 0 {
			for _, f := range args.HOST_NAME {
				tmpObj := jsonutils.NewDict()
				tmpObj.Add(jsonutils.NewString(f), "host_name")
				arr.Add(tmpObj)
			}
		}
		params.Add(arr, "service_hosts")

		rst, err := modules.ServiceHosts.DoDeleteServiceHost(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 查看指定服务树节点的机器
	 */
	type ServiceHostListOptions struct {
		options.BaseListOptions
		Labels string `help:"Labels for tree-node(split by comma)"`
	}
	R(&ServiceHostListOptions{}, "servicehost-list", "List all hosts for the tree-node", func(s *mcclient.ClientSession, args *ServiceHostListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Labels) > 0 {
			params.Add(jsonutils.NewString(args.Labels), "node_labels")
		}

		result, err := modules.ServiceHosts.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ServiceHosts.GetColumns(s))
		return nil
	})

}
