package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type TaskListOptions struct {
		BaseListOptions
	}
	R(&TaskListOptions{}, "prosesslog-list", "List processlogs", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		params := FetchPagingParams(suboptions.BaseListOptions)
		result, err := modules.Processlogs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Processlogs.GetColumns(s))
		return nil
	})
}
