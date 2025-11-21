package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type DifySkuManager struct {
	modulebase.ResourceManager
}

var (
	DifySku DifySkuManager
)

func init() {
	DifySku = DifySkuManager{
		ResourceManager: modules.NewLLMManager("dify_sku", "dify_skus",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&DifySku)
}
