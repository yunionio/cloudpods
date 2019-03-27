package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建一条报警规则
	 */
	type NodealertCreateOptions struct {
		TYPE       string  `help:"Alert rule type" choices:"guest|host"`
		METRIC     string  `help:"Metric name, include measurement and field, such as vm_cpu.usage_active"`
		NODE_NAME  string  `help:"Name of the guest or host"`
		NODE_ID    string  `help:"ID of the guest or host"`
		PERIOD     string  `help:"Specify the query time period for the data"`
		WINDOW     string  `help:"Specify the query interval for the data"`
		THRESHOLD  float64 `help:"Threshold value of the metric"`
		COMPARATOR string  `help:"Comparison operator for join expressions" choices:">|<|>=|<=|=|!="`
		RECIPIENTS string  `help:"Comma separated recipient ID"`
		LEVEL      string  `help:"Alert level" choices:"normal|important|fatal"`
		CHANNEL    string  `help:"Ways to send an alarm" choices:"email|mobile"`
	}
	R(&NodealertCreateOptions{}, "nodealert-create", "Create a node alert rule", func(s *mcclient.ClientSession, args *NodealertCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.METRIC), "metric")
		params.Add(jsonutils.NewString(args.NODE_NAME), "node_name")
		params.Add(jsonutils.NewString(args.NODE_ID), "node_id")
		params.Add(jsonutils.NewString(args.PERIOD), "period")
		params.Add(jsonutils.NewString(args.WINDOW), "window")
		params.Add(jsonutils.NewFloat(args.THRESHOLD), "threshold")
		params.Add(jsonutils.NewString(args.COMPARATOR), "comparator")
		params.Add(jsonutils.NewString(args.RECIPIENTS), "recipients")
		params.Add(jsonutils.NewString(args.LEVEL), "level")
		params.Add(jsonutils.NewString(args.CHANNEL), "channel")

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
		ID         string  `help:"ID of the alert rule"`
		Type       string  `help:"Alert rule type" choices:"guest|host"`
		Metric     string  `help:"Metric name, include measurement and field, such as vm_cpu.usage_active"`
		NodeName   string  `help:"Name of the guest or host"`
		NodeID     string  `help:"ID of the guest or host"`
		Period     string  `help:"Specify the query time period for the data"`
		Window     string  `help:"Specify the query interval for the data"`
		Threshold  float64 `help:"Threshold value of the metric"`
		Comparator string  `help:"Comparison operator for join expressions" choices:">|<|>=|<=|=|!="`
		Recipients string  `help:"Comma separated recipient ID"`
		Level      string  `help:"Alert level" choices:"normal|important|fatal"`
		Channel    string  `help:"Ways to send an alarm" choices:"email|mobile"`
	}
	R(&NodealertUpdateOptions{}, "nodealert-update", "Update the node alert rule", func(s *mcclient.ClientSession, args *NodealertUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.Metric) > 0 {
			params.Add(jsonutils.NewString(args.Metric), "metric")
		}
		if len(args.NodeName) > 0 {
			params.Add(jsonutils.NewString(args.NodeName), "node_name")
		}
		if len(args.NodeID) > 0 {
			params.Add(jsonutils.NewString(args.NodeID), "node_id")
		}
		if len(args.Period) > 0 {
			params.Add(jsonutils.NewString(args.Period), "period")
		}
		if len(args.Window) > 0 {
			params.Add(jsonutils.NewString(args.Window), "window")
		}
		if args.Threshold > 0.0 {
			params.Add(jsonutils.NewFloat(args.Threshold), "threshold")
		}
		if len(args.Comparator) > 0 {
			params.Add(jsonutils.NewString(args.Comparator), "comparator")
		}
		if len(args.Recipients) > 0 {
			params.Add(jsonutils.NewString(args.Recipients), "recipients")
		}
		if len(args.Level) > 0 {
			params.Add(jsonutils.NewString(args.Level), "level")
		}
		if len(args.Channel) > 0 {
			params.Add(jsonutils.NewString(args.Channel), "channel")
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
		ID string `help:"ID of alarm"`
	}
	R(&NodealertDeleteOptions{}, "nodealert-delete", "Delete a node alert", func(s *mcclient.ClientSession, args *NodealertDeleteOptions) error {
		alarm, e := modules.NodeAlert.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 修改指定ID的报警规则状态
	 */
	type NodealertUpdateStatusOptions struct {
		ID     string `help:"ID of the node alert"`
		STATUS string `help:"Name of the new alarm" choices:"Enabled|Disabled"`
	}
	R(&NodealertUpdateStatusOptions{}, "nodealert-change-status", "Change status of a node alert", func(s *mcclient.ClientSession, args *NodealertUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")

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
		Metric   string `help:"Metric name, include measurement and field, such as vm_cpu.usage_active"`
		NodeName string `help:"Name of the guest or host"`
		NodeID   string `help:"ID of the guest or host"`
		options.BaseListOptions
	}
	R(&NodealertListOptions{}, "nodealert-list", "List node alert", func(s *mcclient.ClientSession, args *NodealertListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
			if len(args.Type) > 0 {
				params.Add(jsonutils.NewString(args.Type), "type")
			}
			if len(args.Metric) > 0 {
				params.Add(jsonutils.NewString(args.Metric), "metric")
			}
			if len(args.NodeName) > 0 {
				params.Add(jsonutils.NewString(args.NodeName), "node_name")
			}
			if len(args.NodeID) > 0 {
				params.Add(jsonutils.NewString(args.NodeID), "node_id")
			}
		}
		result, err := modules.NodeAlert.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NodeAlert.GetColumns(s))
		return nil
	})
}
