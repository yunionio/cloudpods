package llm

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMRouterAgent)
	cmd.List(new(options.LLMRouterAgentListOptions))
	cmd.Show(new(options.LLMRouterAgentShowOptions))
	cmd.Create(new(options.LLMRouterAgentCreateOptions))
	cmd.Update(new(options.LLMRouterAgentUpdateOptions))
	cmd.Delete(new(options.LLMRouterAgentDeleteOptions))
	shell.R(&options.LLMRouterAgentRouteOptions{}, "llm-router-agent-route", "Route llm_router_agent", routeLLMRouterAgent)
}

func routeLLMRouterAgent(s *mcclient.ClientSession, args *options.LLMRouterAgentRouteOptions) error {
	id := args.ID
	if id != "default" {
		var err error
		id, err = modules.LLMRouterAgent.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
	}
	bodyJSON, err := args.Params()
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := s.RawVersionRequest(
		modules.LLMRouterAgent.ServiceType(),
		modules.LLMRouterAgent.EndpointType(),
		"POST",
		fmt.Sprintf("/llm_router_agents/%s/route", id),
		headers,
		strings.NewReader(bodyJSON.String()),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Error: %s %s", resp.Status, string(respBody))
	}
	obj, err := jsonutils.Parse(respBody)
	if err != nil {
		return err
	}
	shell.PrintObject(obj)
	return nil
}
