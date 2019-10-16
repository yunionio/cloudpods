package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	InstanceSnapshots modulebase.ResourceManager
)

func init() {
	InstanceSnapshots = NewComputeManager("instance_snapshot", "instance_snapshots",
		[]string{"ID", "Name",
			"Status", "Guest_Id",
		},
		[]string{"Guest_Name", "Snapshots"},
	)

	registerCompute(&InstanceSnapshots)
}
