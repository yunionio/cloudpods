package llm

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MCPAgentListOptions struct {
	options.BaseListOptions

	LLMDriver string `json:"llm_driver" help:"filter by llm driver (ollama or openai)"`
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

	LlmId      string `help:"LLM 实例 ID，如果提供则自动获取 llm_url" json:"llm_id"`
	LLM_URL    string `help:"后端大模型的 base 请求地址" json:"llm_url"`
	LLM_DRIVER string `help:"使用的大模型驱动，可以是 ollama 或 openai" json:"llm_driver" choices:"ollama|openai"`
	MODEL      string `help:"使用的模型名称" json:"model"`
	API_KEY    string `help:"访问大模型的密钥" json:"api_key"`
	McpServer  string `help:"mcp 服务器的后端地址" json:"mcp_server"`
}

func (o *MCPAgentCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type MCPAgentUpdateOptions struct {
	apis.SharableVirtualResourceCreateInput

	ID         string
	LlmId      *string `help:"LLM 实例 ID，如果提供则自动获取 llm_url" json:"llm_id,omitempty"`
	LLM_URL    *string `help:"后端大模型的 base 请求地址" json:"llm_url,omitempty"`
	LLM_DRIVER *string `help:"使用的大模型驱动，可以是 ollama 或 openai" json:"llm_driver,omitempty" choices:"ollama|openai"`
	MODEL      *string `help:"使用的模型名称" json:"model,omitempty"`
	API_KEY    *string `help:"访问大模型的密钥" json:"api_key,omitempty"`
	McpServer  *string `help:"mcp 服务器的后端地址" json:"mcp_server,omitempty"`
}

func (o *MCPAgentUpdateOptions) GetId() string {
	return o.ID
}

func (o *MCPAgentUpdateOptions) Params() (jsonutils.JSONObject, error) {
	// 只包含非空字段
	params := jsonutils.NewDict()
	if o.LlmId != nil && len(*o.LlmId) > 0 {
		params.Set("llm_id", jsonutils.NewString(*o.LlmId))
	}
	if o.LLM_URL != nil && len(*o.LLM_URL) > 0 {
		params.Set("llm_url", jsonutils.NewString(*o.LLM_URL))
	}
	if o.LLM_DRIVER != nil && len(*o.LLM_DRIVER) > 0 {
		params.Set("llm_driver", jsonutils.NewString(*o.LLM_DRIVER))
	}
	if o.MODEL != nil && len(*o.MODEL) > 0 {
		params.Set("model", jsonutils.NewString(*o.MODEL))
	}
	if o.API_KEY != nil && len(*o.API_KEY) > 0 {
		params.Set("api_key", jsonutils.NewString(*o.API_KEY))
	}
	if o.McpServer != nil && len(*o.McpServer) > 0 {
		params.Set("mcp_server", jsonutils.NewString(*o.McpServer))
	}

	// 添加基础字段
	baseParams, err := options.StructToParams(&o.SharableVirtualResourceCreateInput)
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

type MCPAgentChatTestOptions struct {
	MCPAgentIdOptions

	Message string `help:"test message to send to LLM" json:"message"`
}

func (opts *MCPAgentChatTestOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMChatTestInput{
		Message: opts.Message,
	}
	return jsonutils.Marshal(input), nil
}

type MCPAgentMCPAgentRequestOptions struct {
	MCPAgentIdOptions

	Query string `help:"query to send to MCP agent" json:"query"`
}

func (opts *MCPAgentMCPAgentRequestOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMMCPAgentRequestInput{
		Query: opts.Query,
	}
	return jsonutils.Marshal(input), nil
}
