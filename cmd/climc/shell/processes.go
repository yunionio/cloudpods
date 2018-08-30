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
	R(&TaskListOptions{}, "process-list", "List processes", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Processes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Processes.GetColumns(s))
		return nil
	})

	type ProcessShowOptions struct {
		ID string `help:"ID or Name of the process to show"`
	}
	R(&ProcessShowOptions{}, "process-show", "Show process details", func(s *mcclient.ClientSession, args *ProcessShowOptions) error {
		result, err := modules.Processes.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
