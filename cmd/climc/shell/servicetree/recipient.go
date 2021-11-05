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
	 * 向服务树节点添加报警接收人
	 */
	type TreeNodeRecipientsCreateOptions struct {
		NODE_LABELS    string `help:"Labels for tree-node(split by comma)"`
		USER_TYPE      string `help:"User type, user|group "`
		RECIPIENT_ID   string `help:"User id or group id"`
		RECIPIENT_TYPE string `help:"junior(slight) or senior(serious)"`
	}
	R(&TreeNodeRecipientsCreateOptions{}, "treenode-recipient-create", "Add recipients to tree-node", func(s *mcclient.ClientSession, args *TreeNodeRecipientsCreateOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.RECIPIENT_ID), "recipient_id")
		tmpObj.Add(jsonutils.NewString(args.USER_TYPE), "type")
		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NODE_LABELS), "node_labels")
		params.Add(jsonutils.NewString(args.RECIPIENT_TYPE), "recipient_type")
		params.Add(arr, "recipients")

		rst, err := modules.Recipients.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 从服务树节点移除报警接收人
	 */
	type TreeNodeRecipientsDeleteOptions struct {
		NODE_LABELS    string `help:"Labels for tree-node(split by comma)"`
		USER_TYPE      string `help:"User type, user|group "`
		RECIPIENT_ID   string `help:"User id or group id"`
		RECIPIENT_TYPE string `help:"junior(slight) or senior(serious)"`
	}
	R(&TreeNodeRecipientsDeleteOptions{}, "treenode-recipient-delete", "Remove recipients from tree-node", func(s *mcclient.ClientSession, args *TreeNodeRecipientsDeleteOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.RECIPIENT_ID), "recipient_id")
		tmpObj.Add(jsonutils.NewString(args.RECIPIENT_TYPE), "recipient_type")
		tmpObj.Add(jsonutils.NewString(args.USER_TYPE), "type")
		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NODE_LABELS), "node_labels")
		params.Add(arr, "recipients")

		rst, err := modules.Recipients.DoDeleteRecipient(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 查询树节点的报警接收人信息
	 */
	type TreeNodeRecipientsListOptions struct {
		options.BaseListOptions
		LIST_TYPE string `help:"Type of list: junior|senior"`
		LABELS    string `help:"Labels for tree-node(split by comma)"`
	}
	R(&TreeNodeRecipientsListOptions{}, "treenode-recipient-list", "List recipient for the tree-node ", func(s *mcclient.ClientSession, args *TreeNodeRecipientsListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		params.Add(jsonutils.NewString(args.LIST_TYPE), "type")
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")

		result, err := modules.Recipients.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.Recipients.GetColumns(s))
		return nil
	})

}
