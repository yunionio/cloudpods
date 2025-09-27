package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	ServerScreenDumps modulebase.ResourceManager
)

func init() {
	ServerScreenDumps = modules.NewComputeManager("guest_screen_dump", "guest_screen_dumps",
		[]string{"Guest_id", "Name", "Created_at", "S3_endpoint", "S3_bucket_name"},
		[]string{})

	modules.RegisterCompute(&ServerScreenDumps)
}
