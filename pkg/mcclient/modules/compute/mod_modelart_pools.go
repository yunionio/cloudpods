package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ModelartsPoolManager struct {
	modulebase.ResourceManager
}

var (
	ModelartsPools ModelartsPoolManager
)

func init() {
	ModelartsPools = ModelartsPoolManager{modules.NewComputeManager("modelarts_pool", "modelarts_pools",
		[]string{},
		[]string{})}

	modules.RegisterCompute(&ModelartsPools)
}
