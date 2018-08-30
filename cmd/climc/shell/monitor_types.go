package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 列出全部的监控类型
	 */
	type MonitorTypesOptions struct {
		options.BaseListOptions
	}
	R(&MonitorTypesOptions{}, "monitortype-list", "List all monitor types", func(s *mcclient.ClientSession, args *MonitorTypesOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.MonitorTypes.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.MonitorTypes.GetColumns(s))
		return nil
	})

}
