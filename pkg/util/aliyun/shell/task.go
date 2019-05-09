package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TaskListOptions struct {
		TYPE   string   `help:"Task types, either ImportImage or ExportImage" choices:"ImportImage|ExportImage"`
		Task   []string `help:"Task ID"`
		Status string   `help:"Task status" choices:"Finished|Processing|Waiting|Deleted|Paused|Failed"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&TaskListOptions{}, "task-list", "List tasks", func(cli *aliyun.SRegion, args *TaskListOptions) error {
		tasks, total, err := cli.GetTasks(aliyun.TaskActionType(args.TYPE), args.Task, aliyun.TaskStatusType(args.Status), args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(tasks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type TaskIDOptions struct {
		ID string `help:"Task ID"`
	}

	shellutils.R(&TaskIDOptions{}, "task-show", "Show task", func(cli *aliyun.SRegion, args *TaskIDOptions) error {
		task, err := cli.GetTask(args.ID)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})

	shellutils.R(&TaskIDOptions{}, "cancel-task", "Cancel task", func(cli *aliyun.SRegion, args *TaskIDOptions) error {
		return cli.CancelTask(args.ID)
	})

}
