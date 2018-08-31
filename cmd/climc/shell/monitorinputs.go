package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 列出所有监控数据源
	 */
	type MonitorInputsListOptions struct {
		options.BaseListOptions
	}
	R(&MonitorInputsListOptions{}, "monitorinputs-list", "List all monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.MonitorInputs.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.MonitorInputs.GetColumns(s))
		return nil
	})

	/**
	 * 查看监控数据源下的监控项
	 */
	type MonitorInputsShowOptions struct {
		options.BaseListOptions
		ID string `help:"The ID of the monitor-input"`
	}
	R(&MonitorInputsShowOptions{}, "monitorinputs-metrics-list", "List all metrics for the monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsShowOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.MonitorInputs.GetSpecific(s, args.ID, "metrics", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
