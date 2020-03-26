package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	BillTasks modulebase.ResourceManager
)

func init() {
	BillTasks = NewMeterManager("bill_task", "bill_tasks",
		[]string{"status"},
		[]string{},
	)
	register(&BillTasks)
}
