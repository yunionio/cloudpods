package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出全部的监控类型
	 */
	type MonitorTypesOptions struct {
		BaseListOptions
	}
	R(&MonitorTypesOptions{}, "monitortype-list", "List all monitor types", func(s *mcclient.ClientSession, args *MonitorTypesOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.MonitorTypes.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.MonitorTypes.GetColumns(s))
		return nil
	})

}
