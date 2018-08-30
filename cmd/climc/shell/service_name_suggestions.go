package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 列出所有监控指标
	 */
	type ServiceNameSuggestionListOptions struct {
		options.BaseListOptions
	}
	R(&ServiceNameSuggestionListOptions{}, "servicenamesuggestion-list", "List all serviceNameSuggestion", func(s *mcclient.ClientSession, args *ServiceNameSuggestionListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.ServiceNameSuggestion.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ServiceNameSuggestion.GetColumns(s))
		return nil
	})

}
