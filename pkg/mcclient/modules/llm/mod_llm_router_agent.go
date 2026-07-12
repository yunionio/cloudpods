package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMRouterAgentManager struct {
	modulebase.ResourceManager
}

var (
	LLMRouterAgent LLMRouterAgentManager
)

func init() {
	LLMRouterAgent = LLMRouterAgentManager{
		ResourceManager: modules.NewLLMManager("llm_router_agent", "llm_router_agents",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&LLMRouterAgent)
}
