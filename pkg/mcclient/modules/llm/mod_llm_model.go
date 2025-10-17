package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMModelManager struct {
	modulebase.ResourceManager
}

var (
	LLMModel LLMModelManager
)

func init() {
	LLMModel = LLMModelManager{
		ResourceManager: modules.NewLLMManager("llm_model", "llm_models",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&LLMModel)
}
