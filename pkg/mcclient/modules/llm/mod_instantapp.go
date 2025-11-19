package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMInstantApp LLMInstantAppManager
)

func init() {
	LLMInstantApp = LLMInstantAppManager{
		modules.NewLLMManager("llm_instant_app", "llm_instant_apps",
			[]string{},
			[]string{}),
	}
	modules.Register(&LLMInstantApp)
}

type LLMInstantAppManager struct {
	modulebase.ResourceManager
}
