package shell

import (
	"github.com/yunionio/onecloud/pkg/util/aliyun"
	"github.com/yunionio/onecloud/pkg/util/shellutils"
)

func init() {
	type TaskListOptions struct {
		TYPE   string   `help:"Task types, either ImportImage or ExportImage" choices:"ImportImage|ExportImage"`
		Task   []string `help:"Task ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&TaskListOptions{}, "task-list", "List tasks", func(cli *aliyun.SRegion, args *TaskListOptions) error {
		tasks, total, err := cli.GetTasks(aliyun.TaskActionType(args.TYPE), args.Task, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(tasks, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
