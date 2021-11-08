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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 查看所有的监控节点 | 列出匹配标签的节点列表
	 */
	type NodeListOptions struct {
		options.BaseListOptions
		Labels []string `help:"Node labels"`
	}
	R(&NodeListOptions{}, "node-list", "List all nodes", func(s *mcclient.ClientSession, args *NodeListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Labels) > 0 {
			for _, f := range args.Labels {
				parts := strings.Split(f, "=")
				params.Add(jsonutils.NewString(parts[1]), parts[0])
			}
		}

		result, err := modules.Nodes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Nodes.GetColumns(s))
		return nil
	})

	/**
	 * 查看节点的标签
	 */
	type NodeBaseOptions struct {
		ID string `help:"ID of the node"`
	}
	R(&NodeBaseOptions{}, "list-labels-for-node", "List labels for the node", func(s *mcclient.ClientSession, args *NodeBaseOptions) error {
		result, err := modules.Labels.ListInContext(s, nil, &modules.Nodes, args.ID)
		if err != nil {
			return err
		}
		printList(result, modules.Labels.GetColumns(s))
		return nil
	})

	/**
	 * 新增标签
	 */
	type NodeLabelsAddOptions struct {
		ID     string   `help:"ID of node"`
		LABELS []string `help:"Node labels"`
	}
	R(&NodeLabelsAddOptions{}, "add-labels-to-node", "Add labels to node", func(s *mcclient.ClientSession, args *NodeLabelsAddOptions) error {
		labels := jsonutils.NewDict()
		for _, f := range args.LABELS {
			parts := strings.Split(f, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(labels, "labels")

		_, err := modules.Nodes.PerformAction(s, args.ID, "add-labels", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 批量新增标签
	 */
	type NodeLabelsBatchOptions struct {
		NAMES  []string `help:"Node names"`
		Labels []string `help:"Node labels" nargs:"+"`
	}
	R(&NodeLabelsBatchOptions{}, "batch-add-labels-to-node", "Batch add labels to nodes", func(s *mcclient.ClientSession, args *NodeLabelsBatchOptions) error {
		nodes := jsonutils.NewArray()
		labels := jsonutils.NewDict()

		for _, n := range args.NAMES {
			nodes.Add(jsonutils.NewString(n))
		}

		for _, l := range args.Labels {
			parts := strings.Split(l, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(nodes, "nodes")
		params.Add(labels, "labels")

		ret := modules.Nodes.BatchPerformActionInContexts(s, args.NAMES, "add-labels", params, nil)
		printBatchResults(ret, modules.Nodes.GetColumns(s))
		return nil
	})

	/**
	 * 移除标签
	 */
	type NodeLabelRemoveOptions struct {
		ID     string   `help:"ID of node"`
		LABELS []string `help:"Node labels"`
	}
	R(&NodeLabelRemoveOptions{}, "remove-labels-from-node", "Remove labels from node", func(s *mcclient.ClientSession, args *NodeLabelRemoveOptions) error {
		labels := jsonutils.NewDict()
		for _, f := range args.LABELS {
			parts := strings.Split(f, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(labels, "labels")

		_, err := modules.Nodes.PerformAction(s, args.ID, "remove-labels", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 批量移除标签
	 */
	R(&NodeLabelsBatchOptions{}, "batch-remove-labels-from-node", "Batch remove labels from nodes", func(s *mcclient.ClientSession, args *NodeLabelsBatchOptions) error {
		nodes := jsonutils.NewArray()
		labels := jsonutils.NewDict()

		for _, n := range args.NAMES {
			nodes.Add(jsonutils.NewString(n))
		}

		for _, l := range args.Labels {
			parts := strings.Split(l, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(nodes, "nodes")
		params.Add(labels, "labels")

		ret := modules.Nodes.BatchPerformActionInContexts(s, args.NAMES, "remove-labels", params, nil)
		printBatchResults(ret, modules.Nodes.GetColumns(s))
		return nil
	})

	type NodeCreateOptions struct {
		NAME    string `help:"Node name"`
		IP      string `help:"Node IP"`
		ResType string `help:"Resource type" choices:"host|guest"`
	}
	R(&NodeCreateOptions{}, "node-create", "Create a monitor node record", func(s *mcclient.ClientSession, args *NodeCreateOptions) error {
		data, err := modules.Nodes.NewNode(s, args.NAME, args.IP, args.ResType)
		if err != nil {
			return err
		}
		printObject(data)
		return nil
	})
}
