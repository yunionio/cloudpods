package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMs LLMManager
)

func init() {
	LLMs = LLMManager{
		modules.NewLLMManager("llm", "llms",
			[]string{},
			[]string{}),
	}
	modules.Register(&LLMs)
}

type LLMManager struct {
	modulebase.ResourceManager
}
