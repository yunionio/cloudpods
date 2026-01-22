package llm

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	shell.R(&options.MCPAgentMCPAgentRequestOptions{}, "mcp-agent-chat", "Chat with MCP Agent (Stream)", chatStream)
}

func chatStream(s *mcclient.ClientSession, args *options.MCPAgentMCPAgentRequestOptions) error {
	id, err := modules.MCPAgent.GetId(s, args.ID, nil)
	if err != nil {
		return err
	}

	bodyJSON, err := args.Params()
	if err != nil {
		return fmt.Errorf("failed to build request params: %v", err)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	body := strings.NewReader(bodyJSON.String())

	path := fmt.Sprintf("/mcp_agents/%s/chat-stream", id)
	resp, err := s.RawVersionRequest(
		modules.MCPAgent.ServiceType(),
		modules.MCPAgent.EndpointType(),
		"POST",
		path,
		headers,
		body,
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

	scanner := bufio.NewScanner(resp.Body)
	var eventData []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(eventData) > 0 {
				fmt.Print(strings.Join(eventData, "\n"))
				eventData = nil
			}
			continue
		}
		if after, found := strings.CutPrefix(line, "data: "); found {
			eventData = append(eventData, after)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	fmt.Println()
	return nil
}
