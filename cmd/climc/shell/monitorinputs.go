package shell

import (
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {

	/**
	 * 列出所有监控数据源
	 */
	type MonitorInputsListOptions struct {
		BaseListOptions
	}
	R(&MonitorInputsListOptions{}, "monitorinputs-list", "List all monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

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
		BaseListOptions
		ID string `help:"The ID of the monitor-input"`
	}
	R(&MonitorInputsShowOptions{}, "monitorinputs-metrics-list", "List all metrics for the monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsShowOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.MonitorInputs.GetSpecific(s, args.ID, "metrics", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
