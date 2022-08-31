package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ModelartsPoolSkuManager struct {
	modulebase.ResourceManager
}

var (
	ModelartsPoolSku ModelartsPoolSkuManager
)

func init() {
	ModelartsPoolSku = ModelartsPoolSkuManager{modules.NewComputeManager("modelarts_pool_sku", "modelarts_pool_skus",
		[]string{},
		[]string{})}

	modules.RegisterCompute(&ModelartsPoolSku)
}
