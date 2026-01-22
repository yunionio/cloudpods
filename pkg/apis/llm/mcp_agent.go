package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

// LLMClientType 定义 LLM 驱动类型
type LLMClientType string

const (
	LLM_CLIENT_OLLAMA LLMClientType = "ollama"
	LLM_CLIENT_OPENAI LLMClientType = "openai"

	MCP_AGENT_SYSTEM_PROMPT = `你是一个 Cloudpods 云平台管理助手。你可以使用提供的工具来帮助用户管理云资源。

## 你的能力
- 查询云平台资源（虚拟机、镜像、网络、存储、区域等）
- 管理虚拟机（创建、启动、停止、重启、删除、重置密码）
- 获取虚拟机监控信息和实时统计数据

## 重要规则（必须严格遵守）
**如果用户的问题涉及查询、创建、修改或删除云资源，你必须先调用相应的工具，而不是直接回答。**
- 对于需要查询资源的问题（如"列出虚拟机"、"查询状态"等），必须调用工具获取数据后再回答
- 对于需要操作资源的问题（如"创建"、"启动"、"停止"等），必须调用工具执行操作后再回答
- 只有在以下情况才可以直接回复：
  1. 用户只是询问一般性问题（如"你能做什么"、"如何使用"等）
  2. 没有合适的工具可以解决用户的问题
  3. 工具调用失败后需要向用户说明错误原因

## 工作流程
1. 理解用户的需求
2. **优先检查是否有合适的工具可以完成任务，如果有则必须调用工具**
3. 分析工具返回的结果
4. 如果需要更多信息，继续调用其他工具
5. 最后用自然语言总结结果给用户

## 注意事项
- 认证信息已由系统自动处理，调用工具时无需提供认证参数
- 如果工具调用失败，尝试分析错误原因并告知用户
- 回复时使用中文，语言简洁明了
- **不要在没有调用工具的情况下直接回答需要查询或操作资源的问题**
`
)

var (
	LLM_CLIENT_TYPES = sets.NewString(
		string(LLM_CLIENT_OLLAMA),
		string(LLM_CLIENT_OPENAI),
	)
)

// IsLLMClientType 检查给定的字符串是否是有效的 LLM 驱动类型
func IsLLMClientType(t string) bool {
	return LLM_CLIENT_TYPES.Has(t)
}

// MCP Agent 配置相关的 API 定义
type MCPAgentListInput struct {
	apis.SharableVirtualResourceListInput

	LLMDriver string `json:"llm_driver"`
}

type MCPAgentCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	LLMId     string `json:"llm_id" help:"LLM 实例 ID，如果提供则自动获取 llm_url"`
	LLMUrl    string `json:"llm_url" help:"后端大模型的 base 请求地址"`
	LLMDriver string `json:"llm_driver" help:"使用的大模型驱动，可以是 ollama 或 openai"`
	Model     string `json:"model" help:"使用的模型名称"`
	ApiKey    string `json:"api_key" help:"在 llm_driver 中需要用到的认证"`
	McpServer string `json:"mcp_server" help:"mcp 服务器的后端地址"`
}

type MCPAgentUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	LLMId     *string `json:"llm_id,omitempty" help:"LLM 实例 ID，如果提供则自动获取 llm_url"`
	LLMUrl    *string `json:"llm_url,omitempty" help:"后端大模型的 base 请求地址"`
	LLMDriver *string `json:"llm_driver,omitempty" help:"使用的大模型驱动，可以是 ollama 或 openai"`
	Model     *string `json:"model,omitempty" help:"使用的模型名称"`
	ApiKey    *string `json:"api_key,omitempty" help:"在 llm_driver 中需要用到的认证"`
	McpServer *string `json:"mcp_server,omitempty" help:"mcp 服务器的后端地址"`
}

type MCPAgentDetails struct {
	apis.SharableVirtualResourceDetails

	LLMId   string `json:"llm_id"`
	LLMName string `json:"llm_name"`
}

type LLMToolRequestInput struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type LLMMCPAgentRequestInput struct {
	Message string                `json:"message" help:"message to send to MCP agent"`
	History []MCPAgentChatMessage `json:"history" help:"chat history"`
}

type MCPAgentChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ToolCalls []MCPAgentToolCallRecord `json:"tool_calls,omitempty"`
}

// MCPAgentResponse 表示 Agent 响应
type MCPAgentResponse struct {
	// Success 是否成功
	Success bool `json:"success"`
	// Answer 自然语言回答
	Answer string `json:"answer"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
	// ToolCalls 工具调用记录
	ToolCalls []MCPAgentToolCallRecord `json:"tool_calls,omitempty"`
}

// MCPAgentToolCallRecord 记录工具调用
type MCPAgentToolCallRecord struct {
	Id        string                 `json:"id,omitempty"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result"`
}

const (
	// MCPAgentMaxIterations 最大迭代次数，防止无限循环
	MCPAgentMaxIterations = 10
)
