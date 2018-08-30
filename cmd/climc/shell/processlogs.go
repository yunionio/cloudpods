package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type TaskListOptions struct {
		options.BaseListOptions
	}
	R(&TaskListOptions{}, "prosesslog-list", "List processlogs", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Processlogs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Processlogs.GetColumns(s))
		return nil
	})
}
