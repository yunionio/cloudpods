package shell

import (
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type TaskListOptions struct {
		BaseListOptions
	}
	R(&TaskListOptions{}, "task-list", "List taskman", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		params := FetchPagingParams(suboptions.BaseListOptions)
		result, err := modules.Tasks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Tasks.GetColumns(s))
		return nil
	})

	R(&TaskListOptions{}, "region-task-list", "List tasks on region server", func(s *mcclient.ClientSession, suboptions *TaskListOptions) error {
		params := FetchPagingParams(suboptions.BaseListOptions)
		result, err := modules.ComputeTasks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ComputeTasks.GetColumns(s))
		return nil
	})

	type TaskShowOptions struct {
		ID string `help:"ID or name of the task"`
	}
	R(&TaskShowOptions{}, "region-task-show", "Show details of a region task", func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		result, err := modules.ComputeTasks.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
