package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMInstantModel LLMInstantModelManager
)

func init() {
	LLMInstantModel = LLMInstantModelManager{
		modules.NewLLMManager("llm_instant_model", "llm_instant_models",
			[]string{},
			[]string{}),
	}
	modules.Register(&LLMInstantModel)
}

type LLMInstantModelManager struct {
	modulebase.ResourceManager
}
