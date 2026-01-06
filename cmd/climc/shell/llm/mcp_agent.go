package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.MCPAgent)
	cmd.List(new(options.MCPAgentListOptions))
	cmd.Show(new(options.MCPAgentShowOptions))
	cmd.Create(new(options.MCPAgentCreateOptions))
	cmd.Update(new(options.MCPAgentUpdateOptions))
	cmd.Delete(new(options.MCPAgentDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
	cmd.Get("mcp-tools", new(options.MCPAgentIdOptions))
	cmd.Get("tool-request", new(options.MCPAgentToolRequestOptions))
	cmd.Get("chat-test", new(options.MCPAgentChatTestOptions))
	cmd.Get("request", new(options.MCPAgentMCPAgentRequestOptions))
}
