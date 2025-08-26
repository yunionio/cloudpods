package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMs LLMManager
)

func init() {
	LLMs = LLMManager{
		modules.NewComputeManager("llm", "llms",
			[]string{"ID", "Name", "Model", "Guest_ID", "Container_ID", "Status"},
			[]string{}),
	}
	modules.RegisterCompute(&LLMs)
}

type LLMManager struct {
	modulebase.ResourceManager
}
