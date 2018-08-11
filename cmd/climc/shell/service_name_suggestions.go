package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出所有监控指标
	 */
	type ServiceNameSuggestionListOptions struct {
		BaseListOptions
	}
	R(&ServiceNameSuggestionListOptions{}, "servicenamesuggestion-list", "List all serviceNameSuggestion", func(s *mcclient.ClientSession, args *ServiceNameSuggestionListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.ServiceNameSuggestion.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ServiceNameSuggestion.GetColumns(s))
		return nil
	})

}
