package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMSkuManager struct {
	modulebase.ResourceManager
}

var (
	LLMSku LLMSkuManager
)

func init() {
	LLMSku = LLMSkuManager{
		ResourceManager: modules.NewLLMManager("llm_sku", "llm_skus",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&LLMSku)
}
