package llm

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MCPAgentListOptions struct {
	options.BaseListOptions

	LLMDriver    string `json:"llm_driver" help:"filter by llm driver (ollama or openai)"`
	DefaultAgent *bool  `json:"default_agent,omitempty" help:"filter by default agent (true to list the default one)"`
}

func (o *MCPAgentListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type MCPAgentShowOptions struct {
	options.BaseShowOptions
}

func (o *MCPAgentShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type MCPAgentCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	LLM_URL             string `help:"AI 网关 OpenAI 兼容 base 请求地址" json:"llm_url"`
	LLM_DRIVER          string `help:"使用的大模型驱动，固定为 openai" json:"llm_driver" choices:"openai"`
	MODEL               string `help:"使用的模型名称" json:"model"`
	McpServer           string `help:"mcp 服务器的后端地址" json:"mcp_server"`
	AiProxyRoutingId    string `help:"关联的 AI 网关路由规则 ID" json:"aiproxy_routing_id"`
	AiproxyVirtualKeyId string `help:"关联的 AI 网关 API Key ID" json:"aiproxy_virtual_key_id"`
	DefaultAgent        bool   `help:"set as default MCP agent (only one can be true globally)" json:"default_agent"`
}

func (o *MCPAgentCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type MCPAgentUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID                  string
	LlmUrl              *string `help:"AI 网关 OpenAI 兼容 base 请求地址" json:"llm_url,omitempty"`
	LlmDriver           *string `help:"使用的大模型驱动，固定为 openai" json:"llm_driver,omitempty" choices:"openai"`
	Model               *string `help:"使用的模型名称" json:"model,omitempty"`
	McpServer           *string `help:"mcp 服务器的后端地址" json:"mcp_server,omitempty"`
	AiProxyRoutingId    *string `help:"关联的 AI 网关路由规则 ID" json:"aiproxy_routing_id,omitempty"`
	AiproxyVirtualKeyId *string `help:"关联的 AI 网关 API Key ID" json:"aiproxy_virtual_key_id,omitempty"`
	DefaultAgent        *bool   `help:"set as default MCP agent (only one can be true globally)" json:"default_agent,omitempty"`
}

func (o *MCPAgentUpdateOptions) GetId() string {
	return o.ID
}

func (o *MCPAgentUpdateOptions) Params() (jsonutils.JSONObject, error) {
	// 只包含非空字段
	params := jsonutils.NewDict()
	if o.LlmUrl != nil && len(*o.LlmUrl) > 0 {
		params.Set("llm_url", jsonutils.NewString(*o.LlmUrl))
	}
	if o.LlmDriver != nil && len(*o.LlmDriver) > 0 {
		params.Set("llm_driver", jsonutils.NewString(*o.LlmDriver))
	}
	if o.Model != nil && len(*o.Model) > 0 {
		params.Set("model", jsonutils.NewString(*o.Model))
	}
	if o.McpServer != nil && len(*o.McpServer) > 0 {
		params.Set("mcp_server", jsonutils.NewString(*o.McpServer))
	}
	if o.AiProxyRoutingId != nil && len(*o.AiProxyRoutingId) > 0 {
		params.Set("aiproxy_routing_id", jsonutils.NewString(*o.AiProxyRoutingId))
	}
	if o.AiproxyVirtualKeyId != nil && len(*o.AiproxyVirtualKeyId) > 0 {
		params.Set("aiproxy_virtual_key_id", jsonutils.NewString(*o.AiproxyVirtualKeyId))
	}
	if o.DefaultAgent != nil {
		params.Set("default_agent", jsonutils.NewBool(*o.DefaultAgent))
	}

	// 添加基础字段
	baseParams, err := options.StructToParams(&o.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	if baseParams != nil {
		params.Update(baseParams)
	}

	return params, nil
}

type MCPAgentDeleteOptions struct {
	options.BaseIdOptions
}

func (o *MCPAgentDeleteOptions) GetId() string {
	return o.ID
}

func (o *MCPAgentDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type MCPAgentIdOptions struct {
	ID string `help:"mcp agent id" json:"-"`
}

func (opts *MCPAgentIdOptions) GetId() string {
	return opts.ID
}

func (opts *MCPAgentIdOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type MCPAgentToolRequestOptions struct {
	MCPAgentIdOptions

	TOOL_NAME string   `help:"tool name" json:"tool_name"`
	Argument  []string `help:"tool arguments, e.g. key=value" json:"argument"`
}

func (opts *MCPAgentToolRequestOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMToolRequestInput{
		ToolName:  opts.TOOL_NAME,
		Arguments: make(map[string]interface{}),
	}
	for _, arg := range opts.Argument {
		idx := strings.Index(arg, "=")
		if idx > 0 {
			key := arg[:idx]
			val := arg[idx+1:]
			input.Arguments[key] = val
		}
	}
	return jsonutils.Marshal(input), nil
}

type MCPAgentMCPAgentRequestOptions struct {
	MCPAgentIdOptions

	MESSAGE string `help:"message to send to MCP agent" json:"message"`
	History string `help:"chat history as JSON string, e.g. '[{\"role\":\"user\",\"content\":\"hello\"}]'" json:"history,omitempty"`
}

func (opts *MCPAgentMCPAgentRequestOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMMCPAgentRequestInput{
		Message: opts.MESSAGE,
		History: []api.MCPAgentChatMessage{},
	}

	if len(opts.History) > 0 {
		historyJSON, err := jsonutils.ParseString(opts.History)
		if err != nil {
			return nil, fmt.Errorf("failed to parse history JSON: %v", err)
		}
		if historyJSON != nil {
			err = historyJSON.Unmarshal(&input.History)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal history: %v", err)
			}
		}
	}

	return jsonutils.Marshal(input), nil
}

// MCPAgentDefaultChatOptions 用于默认 Agent 聊天（不传 ID，使用 default_agent=true 的条目）
type MCPAgentDefaultChatOptions struct {
	MESSAGE string `help:"message to send to MCP agent" json:"message"`
	History string `help:"chat history as JSON string, e.g. '[{\"role\":\"user\",\"content\":\"hello\"}]'" json:"history,omitempty"`
}

func (opts *MCPAgentDefaultChatOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMMCPAgentRequestInput{
		Message: opts.MESSAGE,
		History: []api.MCPAgentChatMessage{},
	}
	if len(opts.History) > 0 {
		historyJSON, err := jsonutils.ParseString(opts.History)
		if err != nil {
			return nil, fmt.Errorf("failed to parse history JSON: %v", err)
		}
		if historyJSON != nil {
			err = historyJSON.Unmarshal(&input.History)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal history: %v", err)
			}
		}
	}
	return jsonutils.Marshal(input), nil
}
