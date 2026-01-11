package llm

import (
	"fmt"
	"io"
	"net/url"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	// cmd.Get("chat-test", new(options.MCPAgentChatTestOptions))
	cmd.Get("request", new(options.MCPAgentMCPAgentRequestOptions))
	shell.R(&options.MCPAgentChatTestOptions{}, "mcp-agent-chat", "Chat with MCP Agent (Stream)", func(s *mcclient.ClientSession, args *options.MCPAgentChatTestOptions) error {
		id, err := modules.MCPAgent.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("/mcp_agents/%s/chat-stream?message=%s", id, url.QueryEscape(args.Message))

		resp, err := s.RawVersionRequest(
			modules.MCPAgent.ServiceType(),
			modules.MCPAgent.EndpointType(),
			"GET",
			path,
			nil,
			nil,
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// Read error body
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("Error: %s %s", resp.Status, string(body))
		}

		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				fmt.Print(string(buffer[:n]))
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
		}
		fmt.Println()
		return nil
	})
}
