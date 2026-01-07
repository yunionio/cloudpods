package llm_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func init() {
	models.RegisterLLMClientDriver(newOllama())
}

type ollama struct{}

func newOllama() models.ILLMClient {
	return new(ollama)
}

func (o *ollama) GetType() api.LLMClientType {
	return api.LLM_CLIENT_OLLAMA
}

func (o *ollama) Chat(ctx context.Context, mcpAgent *models.SMCPAgent, messages interface{}, tools interface{}) (models.ILLMChatResponse, error) {
	// 转换 messages
	var ollamaMessages []OllamaChatMessage
	if msgs, ok := messages.([]OllamaChatMessage); ok {
		ollamaMessages = msgs
	} else if msgs, ok := messages.([]models.ILLMChatMessage); ok {
		ollamaMessages = make([]OllamaChatMessage, len(msgs))
		for i, msg := range msgs {
			ollamaMessages[i] = OllamaChatMessage{
				Role:    msg.GetRole(),
				Content: msg.GetContent(),
			}
			// 转换工具调用
			if toolCalls := msg.GetToolCalls(); len(toolCalls) > 0 {
				ollamaMessages[i].ToolCalls = make([]OllamaToolCall, len(toolCalls))
				for j, tc := range toolCalls {
					fc := tc.GetFunction()
					ollamaMessages[i].ToolCalls[j] = OllamaToolCall{
						Function: OllamaFunctionCall{
							Name:      fc.GetName(),
							Arguments: fc.GetArguments(),
						},
					}
				}
			}
		}
	} else if msgs, ok := messages.([]interface{}); ok {
		ollamaMessages = make([]OllamaChatMessage, 0, len(msgs))
		for _, msg := range msgs {
			if m, ok := msg.(OllamaChatMessage); ok {
				ollamaMessages = append(ollamaMessages, m)
			} else if m, ok := msg.(models.ILLMChatMessage); ok {
				ollamaMessages = append(ollamaMessages, OllamaChatMessage{
					Role:    m.GetRole(),
					Content: m.GetContent(),
				})
			}
		}
	} else {
		return nil, errors.Error("invalid messages type, expected []OllamaChatMessage or []ILLMChatMessage")
	}

	// 转换 tools
	var ollamaTools []OllamaTool
	if ts, ok := tools.([]OllamaTool); ok {
		ollamaTools = ts
	} else if ts, ok := tools.([]models.ILLMTool); ok {
		ollamaTools = make([]OllamaTool, len(ts))
		for i, t := range ts {
			tf := t.GetFunction()
			ollamaTools[i] = OllamaTool{
				Type: t.GetType(),
				Function: OllamaToolFunction{
					Name:        tf.GetName(),
					Description: tf.GetDescription(),
					Parameters:  tf.GetParameters(),
				},
			}
		}
	} else if ts, ok := tools.([]interface{}); ok && ts != nil {
		ollamaTools = make([]OllamaTool, 0, len(ts))
		for _, tool := range ts {
			if t, ok := tool.(OllamaTool); ok {
				ollamaTools = append(ollamaTools, t)
			} else if t, ok := tool.(models.ILLMTool); ok {
				tf := t.GetFunction()
				ollamaTools = append(ollamaTools, OllamaTool{
					Type: t.GetType(),
					Function: OllamaToolFunction{
						Name:        tf.GetName(),
						Description: tf.GetDescription(),
						Parameters:  tf.GetParameters(),
					},
				})
			}
		}
	} else if tools == nil {
		ollamaTools = nil
	} else {
		return nil, errors.Error("invalid tools type, expected []OllamaTool or []ILLMTool or nil")
	}

	// 调用底层方法
	return o.doChatRequest(ctx, mcpAgent.LLMUrl, mcpAgent.Model, ollamaMessages, ollamaTools)
}

// doChatRequest 执行聊天请求
func (o *ollama) doChatRequest(ctx context.Context, endpoint, model string, messages []OllamaChatMessage, tools []OllamaTool) (*OllamaChatResponse, error) {
	req := OllamaChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request")
	}

	// 规范化 endpoint，确保以 / 结尾
	endpoint = strings.TrimSuffix(endpoint, "/")

	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid endpoint URL %s", endpoint)
	}

	// 构建完整的 URL
	apiURL := baseURL.JoinPath("/api/chat")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "do request")
	}
	defer resp.Body.Close()

	// 读取响应体以便错误处理
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var chatResp OllamaChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, errors.Wrapf(err, "decode response: %s", string(body))
	}

	return &chatResp, nil
}

func (o *ollama) NewUserMessage(content string) models.ILLMChatMessage {
	return &OllamaChatMessage{
		Role:    "user",
		Content: content,
	}
}

func (o *ollama) NewAssistantMessageWithToolCalls(toolCalls []models.ILLMToolCall) models.ILLMChatMessage {
	// to ollama tool calls
	ollamaToolCalls := make([]OllamaToolCall, len(toolCalls))

	for i, tc := range toolCalls {
		if otc, ok := tc.(*OllamaToolCall); ok {
			ollamaToolCalls[i] = *otc
		} else {
			fc := tc.GetFunction()
			ollamaToolCalls[i] = OllamaToolCall{
				Function: OllamaFunctionCall{
					Name:      fc.GetName(),
					Arguments: fc.GetArguments(),
				},
			}
		}
	}

	return &OllamaChatMessage{
		Role:      "assistant",
		Content:   "",
		ToolCalls: ollamaToolCalls,
	}
}

func (o *ollama) NewToolMessage(toolId string, toolName string, content string) models.ILLMChatMessage {
	return &OllamaChatMessage{
		Role:    "tool",
		Content: fmt.Sprintf("[%s] %s", toolName, content),
	}
}

func (o *ollama) NewSystemMessage(content string) models.ILLMChatMessage {
	return OllamaChatMessage{
		Role:    "system",
		Content: content,
	}
}

func (o *ollama) ConvertMCPTools(mcpTools []mcp.Tool) []models.ILLMTool {
	tools := make([]models.ILLMTool, len(mcpTools))
	for i, t := range mcpTools {
		var params map[string]interface{}
		if t.RawInputSchema != nil {
			_ = json.Unmarshal(t.RawInputSchema, &params)
		} else {
			schemaBytes, _ := json.Marshal(t.InputSchema)
			_ = json.Unmarshal(schemaBytes, &params)
		}
		tools[i] = &OllamaTool{
			Type: "function",
			Function: OllamaToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return tools
}

// OllamaChatMessage 表示聊天消息
// 实现 ILLMChatMessage 接口
type OllamaChatMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

// GetRole 实现 ILLMChatMessage 接口
func (m OllamaChatMessage) GetRole() string {
	return m.Role
}

// GetContent 实现 ILLMChatMessage 接口
func (m OllamaChatMessage) GetContent() string {
	return m.Content
}

// GetToolCalls 实现 ILLMChatMessage 接口
func (m OllamaChatMessage) GetToolCalls() []models.ILLMToolCall {
	if len(m.ToolCalls) == 0 {
		return nil
	}
	toolCalls := make([]models.ILLMToolCall, len(m.ToolCalls))
	for i := range m.ToolCalls {
		// 创建副本以避免引用问题
		tc := m.ToolCalls[i]
		toolCalls[i] = &tc
	}
	return toolCalls
}

// OllamaToolCall 表示工具调用
// 实现 ILLMToolCall 接口
type OllamaToolCall struct {
	Function OllamaFunctionCall `json:"function"`
}

// GetFunction 实现 ILLMToolCall 接口
func (tc *OllamaToolCall) GetFunction() models.ILLMFunctionCall {
	return &tc.Function
}

// GetId 实现 ILLMToolCall 接口
func (tc *OllamaToolCall) GetId() string {
	return ""
}

// OllamaFunctionCall 表示函数调用详情
// 实现 ILLMFunctionCall 接口
type OllamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// GetName 实现 ILLMFunctionCall 接口
func (fc *OllamaFunctionCall) GetName() string {
	return fc.Name
}

// GetArguments 实现 ILLMFunctionCall 接口
func (fc *OllamaFunctionCall) GetArguments() map[string]interface{} {
	return fc.Arguments
}

// OllamaTool 表示工具定义
// 实现 ILLMTool 接口
type OllamaTool struct {
	Type     string             `json:"type"`
	Function OllamaToolFunction `json:"function"`
}

// GetType 实现 ILLMTool 接口
func (t OllamaTool) GetType() string {
	return t.Type
}

// GetFunction 实现 ILLMTool 接口
func (t OllamaTool) GetFunction() models.ILLMToolFunction {
	return &t.Function
}

// OllamaToolFunction 表示工具函数定义
// 实现 ILLMToolFunction 接口
type OllamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GetName 实现 ILLMToolFunction 接口
func (tf *OllamaToolFunction) GetName() string {
	return tf.Name
}

// GetDescription 实现 ILLMToolFunction 接口
func (tf *OllamaToolFunction) GetDescription() string {
	return tf.Description
}

// GetParameters 实现 ILLMToolFunction 接口
func (tf *OllamaToolFunction) GetParameters() map[string]interface{} {
	return tf.Parameters
}

// OllamaChatRequest 表示聊天请求
type OllamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []OllamaChatMessage `json:"messages"`
	Tools    []OllamaTool        `json:"tools,omitempty"`
	Stream   bool                `json:"stream"`
}

// OllamaChatResponse 表示聊天响应
type OllamaChatResponse struct {
	Model      string            `json:"model"`
	CreatedAt  string            `json:"created_at"`
	Message    OllamaChatMessage `json:"message"`
	Done       bool              `json:"done"`
	DoneReason string            `json:"done_reason,omitempty"`
}

// GetContent 获取响应内容
func (r *OllamaChatResponse) GetContent() string {
	return r.Message.Content
}

// HasToolCalls 检查响应是否包含工具调用
func (r *OllamaChatResponse) HasToolCalls() bool {
	return len(r.Message.ToolCalls) > 0
}

// GetToolCalls 获取工具调用列表
func (r *OllamaChatResponse) GetToolCalls() []models.ILLMToolCall {
	if len(r.Message.ToolCalls) == 0 {
		return nil
	}
	toolCalls := make([]models.ILLMToolCall, len(r.Message.ToolCalls))
	for i := range r.Message.ToolCalls {
		toolCalls[i] = &r.Message.ToolCalls[i]
	}
	return toolCalls
}

// // ChatStream 发送流式聊天请求（未来扩展）
// func (c *OllamaClient) ChatStream(ctx context.Context, messages []OllamaChatMessage, tools []OllamaTool, onChunk func(*OllamaChatResponse) error) error {
// 	req := OllamaChatRequest{
// 		Model:    c.model,
// 		Messages: messages,
// 		Tools:    tools,
// 		Stream:   true,
// 	}

// 	reqBody, err := json.Marshal(req)
// 	if err != nil {
// 		return errors.Wrap(err, "marshal request")
// 	}

// 	apiURL := c.baseURL.JoinPath("/api/chat")
// 	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL.String(), bytes.NewReader(reqBody))
// 	if err != nil {
// 		return errors.Wrap(err, "create request")
// 	}
// 	httpReq.Header.Set("Content-Type", "application/json")
// 	httpReq.Header.Set("Accept", "text/event-stream")

// 	resp, err := c.client.Do(httpReq)
// 	if err != nil {
// 		return errors.Wrap(err, "do request")
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		body, _ := io.ReadAll(resp.Body)
// 		return errors.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
// 	}

// 	decoder := json.NewDecoder(resp.Body)
// 	for {
// 		var chunk OllamaChatResponse
// 		if err := decoder.Decode(&chunk); err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			return errors.Wrap(err, "decode stream chunk")
// 		}

// 		if onChunk != nil {
// 			if err := onChunk(&chunk); err != nil {
// 				return errors.Wrap(err, "process chunk")
// 			}
// 		}

// 		if chunk.Done {
// 			break
// 		}
// 	}

// 	return nil
// }
