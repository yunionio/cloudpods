package models

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	llm "yunion.io/x/onecloud/pkg/apis/llm"
)

type ILLMChatMessage interface {
	GetRole() string
	GetContent() string
	GetToolCalls() []ILLMToolCall
}

// ILLMToolCall 表示工具调用接口
type ILLMToolCall interface {
	GetId() string
	GetFunction() ILLMFunctionCall
}

// ILLMFunctionCall 表示函数调用详情接口
type ILLMFunctionCall interface {
	GetName() string
	GetArguments() map[string]interface{}
}

// ILLMTool 表示工具定义接口
type ILLMTool interface {
	GetType() string
	GetFunction() ILLMToolFunction
}

// ILLMToolFunction 表示工具函数定义接口
type ILLMToolFunction interface {
	GetName() string
	GetDescription() string
	GetParameters() map[string]interface{}
}

// ILLMChatResponse 表示 LLM 聊天响应接口
// 参考 mcp_agent.go 中的 LLMChatResponse 接口设计
type ILLMChatResponse interface {
	// HasToolCalls 检查响应是否包含工具调用
	HasToolCalls() bool
	// GetToolCalls 获取工具调用列表
	GetToolCalls() []ILLMToolCall
	// GetContent 获取响应内容
	GetContent() string
}

type ILLMClient interface {
	GetType() llm.LLMClientType

	Chat(ctx context.Context, mcpAgent *SMCPAgent, messages interface{}, tools interface{}) (ILLMChatResponse, error)
	ChatStream(ctx context.Context, mcpAgent *SMCPAgent, messages interface{}, tools interface{}, onChunk func(ILLMChatResponse) error) error

	NewUserMessage(content string) ILLMChatMessage
	NewAssistantMessage(content string) ILLMChatMessage
	NewAssistantMessageWithToolCalls(toolCalls []ILLMToolCall) ILLMChatMessage
	NewToolMessage(toolId string, toolName string, content string) ILLMChatMessage
	NewSystemMessage(content string) ILLMChatMessage

	ConvertMCPTools(mcpTools []mcp.Tool) []ILLMTool
}

type SLLMToolCall struct {
	Id       string
	Function SLLMFunctionCall
}

func (tc *SLLMToolCall) GetId() string {
	return tc.Id
}

func (tc *SLLMToolCall) GetFunction() ILLMFunctionCall {
	return &tc.Function
}

type SLLMFunctionCall struct {
	Name      string
	Arguments map[string]interface{}
}

func (fc *SLLMFunctionCall) GetName() string {
	return fc.Name
}

func (fc *SLLMFunctionCall) GetArguments() map[string]interface{} {
	return fc.Arguments
}

var (
	llmClientDrivers = newDrivers()
)

func RegisterLLMClientDriver(drv ILLMClient) {
	registerDriver(llmClientDrivers, drv.GetType(), drv)
}

func GetLLMClientDriver(typ llm.LLMClientType) ILLMClient {
	return getDriver[llm.LLMClientType, ILLMClient](llmClientDrivers, typ)
}

func GetLLMClientDriverWithError(typ llm.LLMClientType) (ILLMClient, error) {
	return getDriverWithError[llm.LLMClientType, ILLMClient](llmClientDrivers, typ)
}
