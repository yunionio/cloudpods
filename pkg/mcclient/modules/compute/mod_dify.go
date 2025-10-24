package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Difies DifyManager
)

func init() {
	Difies = DifyManager{
		modules.NewComputeManager("dify", "difies",
			[]string{"ID", "Name", "Guest_ID", "Containers", "Status"},
			[]string{}),
	}
	modules.RegisterCompute(&Difies)
}

type DifyManager struct {
	modulebase.ResourceManager
}
