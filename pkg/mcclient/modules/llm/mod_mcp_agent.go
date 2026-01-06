package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type MCPAgentManager struct {
	modulebase.ResourceManager
}

var (
	MCPAgent MCPAgentManager
)

func init() {
	MCPAgent = MCPAgentManager{
		ResourceManager: modules.NewLLMManager("mcp_agent", "mcp_agents",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&MCPAgent)
}
