package llm_client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func init() {
	models.RegisterLLMClientDriver(newOpenAI())
}

type openai struct{}

func newOpenAI() models.ILLMClient {
	return new(openai)
}

func (o *openai) GetType() api.LLMClientType {
	return api.LLM_CLIENT_OPENAI
}

func (o *openai) Chat(ctx context.Context, mcpAgent *models.SMCPAgent, messages interface{}, tools interface{}) (models.ILLMChatResponse, error) {
	// 转换 messages
	var openaiMessages []OpenAIChatMessage
	if msgs, ok := messages.([]OpenAIChatMessage); ok {
		openaiMessages = msgs
	} else if msgs, ok := messages.([]models.ILLMChatMessage); ok {
		openaiMessages = make([]OpenAIChatMessage, len(msgs))
		for i, msg := range msgs {
			// Check if it's an OpenAIChatMessage to preserve ToolCallID
			if om, ok := msg.(*OpenAIChatMessage); ok {
				openaiMessages[i] = *om
			} else {
				// General conversion
				openaiMessages[i] = OpenAIChatMessage{
					Role:    msg.GetRole(),
					Content: msg.GetContent(),
				}
				// 转换工具调用
				if toolCalls := msg.GetToolCalls(); len(toolCalls) > 0 {
					openaiMessages[i].ToolCalls = make([]OpenAIToolCall, len(toolCalls))
					for j, tc := range toolCalls {
						fc := tc.GetFunction()
						argsBytes, _ := json.Marshal(fc.GetArguments())
						openaiMessages[i].ToolCalls[j] = OpenAIToolCall{
							ID:   tc.GetId(),
							Type: "function",
							Function: OpenAIFunctionCall{
								Name:      fc.GetName(),
								Arguments: string(argsBytes),
							},
						}
					}
				}
			}
		}
	} else {
		return nil, errors.Error("invalid messages type")
	}

	// 转换 tools
	var openaiTools []OpenAITool
	if ts, ok := tools.([]OpenAITool); ok {
		openaiTools = ts
	} else if ts, ok := tools.([]models.ILLMTool); ok {
		openaiTools = make([]OpenAITool, len(ts))
		for i, t := range ts {
			tf := t.GetFunction()
			openaiTools[i] = OpenAITool{
				Type: t.GetType(),
				Function: OpenAIToolFunction{
					Name:        tf.GetName(),
					Description: tf.GetDescription(),
					Parameters:  tf.GetParameters(),
				},
			}
		}
	} else if tools == nil {
		openaiTools = nil
	}

	return o.doChatRequest(ctx, mcpAgent, openaiMessages, openaiTools)
}

func (o *openai) doChatRequest(ctx context.Context, mcpAgent *models.SMCPAgent, messages []OpenAIChatMessage, tools []OpenAITool) (*OpenAIChatResponse, error) {
	req := OpenAIChatRequest{
		Model:    mcpAgent.Model,
		Messages: messages,
		Tools:    tools,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request")
	}

	endpoint := strings.TrimSuffix(mcpAgent.LLMUrl, "/")
	// Default to /v1/chat/completions if not specified and not a custom path
	if !strings.Contains(endpoint, "/chat/completions") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint = endpoint + "/chat/completions"
		} else {
			endpoint = endpoint + "/v1/chat/completions"
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if mcpAgent.ApiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+mcpAgent.ApiKey)
	}

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "do request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var chatResp OpenAIChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, errors.Wrapf(err, "decode response: %s", string(body))
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.Error("no choices in response")
	}

	return &chatResp, nil
}

func (o *openai) NewUserMessage(content string) models.ILLMChatMessage {
	return &OpenAIChatMessage{
		Role:    "user",
		Content: content,
	}
}

func (o *openai) NewAssistantMessageWithToolCalls(toolCalls []models.ILLMToolCall) models.ILLMChatMessage {
	openaiToolCalls := make([]OpenAIToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		if otc, ok := tc.(*OpenAIToolCall); ok {
			openaiToolCalls[i] = *otc
		} else {
			fc := tc.GetFunction()
			argsBytes, _ := json.Marshal(fc.GetArguments())
			openaiToolCalls[i] = OpenAIToolCall{
				ID:   tc.GetId(),
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      fc.GetName(),
					Arguments: string(argsBytes),
				},
			}
		}
	}

	return &OpenAIChatMessage{
		Role:      "assistant",
		ToolCalls: openaiToolCalls,
	}
}

func (o *openai) NewToolMessage(toolId string, toolName string, content string) models.ILLMChatMessage {
	return &OpenAIChatMessage{
		Role:       "tool",
		ToolCallID: toolId,
		Content:    content,
	}
}

func (o *openai) NewSystemMessage(content string) models.ILLMChatMessage {
	return &OpenAIChatMessage{
		Role:    "system",
		Content: content,
	}
}

func (o *openai) ConvertMCPTools(mcpTools []mcp.Tool) []models.ILLMTool {
	tools := make([]models.ILLMTool, len(mcpTools))
	for i, t := range mcpTools {
		var params map[string]interface{}
		if t.RawInputSchema != nil {
			_ = json.Unmarshal(t.RawInputSchema, &params)
		} else {
			schemaBytes, _ := json.Marshal(t.InputSchema)
			_ = json.Unmarshal(schemaBytes, &params)
		}
		tools[i] = &OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return tools
}

// Structures

type OpenAIChatMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

func (m *OpenAIChatMessage) GetRole() string    { return m.Role }
func (m *OpenAIChatMessage) GetContent() string { return m.Content }
func (m *OpenAIChatMessage) GetToolCalls() []models.ILLMToolCall {
	if len(m.ToolCalls) == 0 {
		return nil
	}
	toolCalls := make([]models.ILLMToolCall, len(m.ToolCalls))
	for i := range m.ToolCalls {
		tc := m.ToolCalls[i]
		toolCalls[i] = &tc
	}
	return toolCalls
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

func (tc *OpenAIToolCall) GetFunction() models.ILLMFunctionCall { return &tc.Function }
func (tc *OpenAIToolCall) GetId() string                        { return tc.ID }

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (fc *OpenAIFunctionCall) GetName() string { return fc.Name }
func (fc *OpenAIFunctionCall) GetArguments() map[string]interface{} {
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(fc.Arguments), &args)
	return args
}

type OpenAITool struct {
	Type     string             `json:"type"`
	Function OpenAIToolFunction `json:"function"`
}

func (t *OpenAITool) GetType() string                      { return t.Type }
func (t *OpenAITool) GetFunction() models.ILLMToolFunction { return &t.Function }

type OpenAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func (tf *OpenAIToolFunction) GetName() string                       { return tf.Name }
func (tf *OpenAIToolFunction) GetDescription() string                { return tf.Description }
func (tf *OpenAIToolFunction) GetParameters() map[string]interface{} { return tf.Parameters }

type OpenAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenAIChatMessage `json:"messages"`
	Tools    []OpenAITool        `json:"tools,omitempty"`
}

type OpenAIChatResponse struct {
	ID      string         `json:"id"`
	Choices []OpenAIChoice `json:"choices"`
}

type OpenAIChoice struct {
	Message      OpenAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

func (r *OpenAIChatResponse) GetContent() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}

func (r *OpenAIChatResponse) HasToolCalls() bool {
	return len(r.Choices) > 0 && len(r.Choices[0].Message.ToolCalls) > 0
}

func (r *OpenAIChatResponse) GetToolCalls() []models.ILLMToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	return r.Choices[0].Message.GetToolCalls()
}
