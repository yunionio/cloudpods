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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建一条报警规则
	 */
	type NodealertCreateOptions struct {
		Type       string  `help:"Alert rule type" required:"true" choices:"guest|host"`
		Metric     string  `help:"Metric name, include measurement and field, e.g. vm_cpu.usage_active" required:"true"`
		NodeName   string  `help:"Name of the guest or host" required:"true"`
		NodeID     string  `help:"ID of the guest or host" required:"true"`
		Period     string  `help:"Specify the query time period for the data" required:"true"`
		Window     string  `help:"Specify the query interval for the data" required:"true"`
		Threshold  float64 `help:"Threshold value of the metric" required:"true"`
		Comparator string  `help:"Comparison operator for join expressions" choices:">|<|>=|<=|=|!=" required:"true"`
		Recipients string  `help:"Comma separated recipient ID" required:"true"`
		Level      string  `help:"Alert level" required:"true" choices:"normal|important|fatal"`
		Channel    string  `help:"Ways to send an alarm" required:"true" choices:"email|mobile|dingtalk"`
	}
	R(&NodealertCreateOptions{}, "nodealert-create", "Create a node alert rule", func(s *mcclient.ClientSession, args *NodealertCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}

		rst, err := modules.NodeAlert.Create(s, params)
		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 修改指定的报警规则
	 */
	type NodealertUpdateOptions struct {
		ID         string   `help:"ID of the alert rule" required:"true" positional:"true"`
		Type       string   `help:"Alert rule type" choices:"guest|host"`
		Metric     string   `help:"Metric name, include measurement and field, such as vm_cpu.usage_active"`
		NodeName   string   `help:"Name of the guest or host"`
		NodeID     string   `help:"ID of the guest or host"`
		Period     string   `help:"Specify the query time period for the data"`
		Window     string   `help:"Specify the query interval for the data"`
		Threshold  *float64 `help:"Threshold value of the metric"`
		Comparator string   `help:"Comparison operator for join expressions" choices:">|<|>=|<=|=|!="`
		Recipients string   `help:"Comma separated recipient ID"`
		Level      string   `help:"Alert level" choices:"normal|important|fatal"`
		Channel    string   `help:"Ways to send an alarm" choices:"email|mobile"`
	}
	R(&NodealertUpdateOptions{}, "nodealert-update", "Update the node alert rule", func(s *mcclient.ClientSession, args *NodealertUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}

		rst, err := modules.NodeAlert.Patch(s, args.ID, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 删除指定ID的报警规则
	 */
	type NodealertDeleteOptions struct {
		ID []string `help:"ID of node alert" required:"true" positional:"true"`
	}
	R(&NodealertDeleteOptions{}, "nodealert-delete", "Delete a node alert", func(s *mcclient.ClientSession, args *NodealertDeleteOptions) error {
		ret := modules.NodeAlert.BatchDelete(s, args.ID, nil)
		printBatchResults(ret, modules.NodeAlert.GetColumns(s))
		return nil
	})

	/**
	 * 启用指定ID的报警规则状态
	 */
	type NodealertUpdateStatusOptions struct {
		ID string `help:"ID of the node alert" required:"true" positional:"true"`
	}
	R(&NodealertUpdateStatusOptions{}, "nodealert-enable", "Enable alert rule for the specified ID", func(s *mcclient.ClientSession, args *NodealertUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("Enabled"), "status")

		alarm, err := modules.NodeAlert.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 禁用指定ID的报警规则状态
	 */
	R(&NodealertUpdateStatusOptions{}, "nodealert-disable", "Disaable alert rule for the specified ID", func(s *mcclient.ClientSession, args *NodealertUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("Disabled"), "status")

		alarm, err := modules.NodeAlert.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 列出报警规则
	 */
	type NodealertListOptions struct {
		Type     string `help:"Alarm rule type" choices:"guest|host"`
		Metric   string `help:"Metric name, include measurement and field, e.g. vm_cpu.usage_active"`
		NodeName string `help:"Name of the guest or host"`
		NodeID   string `help:"ID of the guest or host"`
		options.BaseListOptions
	}
	R(&NodealertListOptions{}, "nodealert-list", "List node alert", func(s *mcclient.ClientSession, args *NodealertListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		result, err := modules.NodeAlert.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NodeAlert.GetColumns(s))
		return nil
	})
}
