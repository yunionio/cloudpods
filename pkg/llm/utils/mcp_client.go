// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// mcpError represents the error object in a JSON-RPC response
type mcpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// rawMCPResponse 用于处理 MCP 响应，支持延迟解析 Result
type rawMCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      mcp.RequestId   `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

// MCPClient 是 MCP Server 的客户端，通过 SSE 协议与 MCP Server 通信
type MCPClient struct {
	serverURL   string
	client      *http.Client
	sessionURL  string
	sseBody     io.ReadCloser
	messageID   int64
	mu          sync.Mutex
	initialized bool
	userCred    mcclient.TokenCredential

	pendingReqs map[int64]chan *rawMCPResponse
	reqMu       sync.Mutex
}

// NewMCPClient 创建一个新的 MCP 客户端
func NewMCPClient(serverURL string, timeout time.Duration, userCred mcclient.TokenCredential) *MCPClient {
	return &MCPClient{
		serverURL: strings.TrimSuffix(serverURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
		userCred:    userCred,
		pendingReqs: make(map[int64]chan *rawMCPResponse),
	}
}

// connectSSE 连接 SSE 端点并开始事件循环
func (c *MCPClient) connectSSE(ctx context.Context) error {
	// 连接 SSE 端点获取 session URL
	sseURL := c.serverURL + "/sse"
	req, err := http.NewRequestWithContext(ctx, "GET", sseURL, nil)
	if err != nil {
		return errors.Wrap(err, "create SSE request")
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "connect to SSE")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return errors.Errorf("SSE connection failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.sseBody = resp.Body

	// Channel to signal session URL found
	done := make(chan struct{})
	var initErr error

	// 读取 endpoint 事件获取 session URL
	go func() {
		reader := bufio.NewReader(c.sseBody)
		foundSession := false
		defer func() {
			if !foundSession {
				select {
				case <-done:
				default:
					close(done)
				}
			}
		}()

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if !foundSession {
					initErr = err
				} else {
					log.Warningf("SSE connection closed: %v", err)
				}
				return
			}

			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if !foundSession {
					if strings.Contains(data, "/message") {
						// 解析 session URL
						c.sessionURL = c.serverURL + data
						log.Infof("MCP Client initialized with session URL: %s", c.sessionURL)
						foundSession = true
						close(done)
					}
				} else {
					// 尝试解析为 JSON-RPC 响应
					var resp rawMCPResponse
					if err := json.Unmarshal([]byte(data), &resp); err == nil && resp.JSONRPC == mcp.JSONRPC_VERSION {
						// 提取 ID
						var reqID int64
						if idVal, ok := resp.ID.Value().(int64); ok {
							reqID = idVal
						} else if idVal, ok := resp.ID.Value().(float64); ok {
							reqID = int64(idVal)
						} else {
							// 可能是通知或 ID 类型不匹配，忽略
							continue
						}

						c.reqMu.Lock()
						ch, ok := c.pendingReqs[reqID]
						if ok {
							delete(c.pendingReqs, reqID)
						}
						c.reqMu.Unlock()

						if ok {
							select {
							case ch <- &resp:
							default:
								log.Warningf("response channel blocked for request %d", reqID)
							}
						}
					}
				}
			}
		}
	}()

	// Wait for session URL
	select {
	case <-done:
		if initErr != nil {
			c.sseBody.Close()
			return errors.Wrap(initErr, "read SSE event")
		}
	case <-time.After(10 * time.Second):
		c.sseBody.Close()
		return errors.Error("timeout waiting for session URL")
	case <-ctx.Done():
		c.sseBody.Close()
		return ctx.Err()
	}
	return nil
}

// Initialize 初始化 MCP 客户端连接
func (c *MCPClient) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	if err := c.connectSSE(ctx); err != nil {
		return err
	}

	// 发送初始化请求
	initParams := mcp.InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    mcp.ClientCapabilities{},
		ClientInfo: mcp.Implementation{
			Name:    "cloudpods-mcp-agent",
			Version: "1.0.0",
		},
	}

	initReq := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(c.nextMessageID()),
		Params:  initParams,
	}
	initReq.Method = string(mcp.MethodInitialize)

	_, err := c.sendRequest(ctx, initReq)
	if err != nil {
		c.sseBody.Close()
		return errors.Wrap(err, "send initialize request")
	}

	// 发送 initialized 通知
	notifyReq := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
	}
	notifyReq.Method = "notifications/initialized"

	_, err = c.sendRequest(ctx, notifyReq)
	if err != nil {
		log.Warningf("send initialized notification failed: %v", err)
	}

	c.initialized = true
	return nil
}

// nextMessageID 生成下一个消息 ID
func (c *MCPClient) nextMessageID() int64 {
	return atomic.AddInt64(&c.messageID, 1)
}

// sendRequest 发送 JSON-RPC 请求
func (c *MCPClient) sendRequest(ctx context.Context, req mcp.JSONRPCRequest) (*rawMCPResponse, error) {
	var respChan chan *rawMCPResponse
	var reqID int64
	var hasID bool

	if !req.ID.IsNil() {
		if idVal, ok := req.ID.Value().(int64); ok {
			reqID = idVal
			hasID = true
		}
	}

	if hasID {
		respChan = make(chan *rawMCPResponse, 1)
		c.reqMu.Lock()
		c.pendingReqs[reqID] = respChan
		c.reqMu.Unlock()

		// 确保在出错返回时清理 pendingReqs
		defer func() {
			c.reqMu.Lock()
			delete(c.pendingReqs, reqID)
			c.reqMu.Unlock()
		}()
	}

	reqBody := jsonutils.Marshal(req)
	log.Infof("MCP request: %s", reqBody.String())

	cli := auth.Client()
	if cli == nil {
		cli = mcclient.NewClient("", 0, false, true, "", "")
	}

	cred := c.userCred
	if cred == nil {
		log.Warningf("userCred is nil in sendRequest, creating empty token")
		cred = &mcclient.SSimpleToken{}
	}

	s := cli.NewSession(ctx, "", "", "", cred)
	s.SetServiceUrl("mcp", c.sessionURL)

	_, respBody, err := s.JSONRequest("mcp", "", "POST", "", nil, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "send request")
	}

	// 对于通知请求，可能没有响应体
	if !hasID {
		return nil, nil
	}

	// 如果有响应体，直接解析
	if respBody != nil {
		log.Debugf("MCP response (HTTP): %s", respBody.String())
		var mcpResp rawMCPResponse
		if err := respBody.Unmarshal(&mcpResp); err != nil {
			return nil, errors.Wrap(err, "decode response")
		}
		if mcpResp.Error != nil {
			return nil, errors.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
		}
		// 成功收到 HTTP 响应，从 pending 中移除（defer 会做，但我们可以提前返回）
		return &mcpResp, nil
	}

	// 如果响应为空，等待 SSE 推送
	select {
	case mcpResp := <-respChan:
		log.Debugf("MCP response (SSE): ID=%v", mcpResp.ID)
		if mcpResp.Error != nil {
			return nil, errors.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
		}
		return mcpResp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, errors.Error("timeout waiting for SSE response")
	}
}

// ListTools 获取可用工具列表
func (c *MCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, errors.Wrap(err, "initialize client")
		}
	}

	req := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(c.nextMessageID()),
	}
	req.Method = string(mcp.MethodToolsList)

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "send tools/list request")
	}

	if resp == nil {
		return nil, errors.Error("empty response for tools/list")
	}

	var result mcp.ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, errors.Wrap(err, "decode tools list result")
	}

	return result.Tools, nil
}

// CallTool 调用工具
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, errors.Wrap(err, "initialize client")
		}
	}

	params := mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}

	req := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(c.nextMessageID()),
		Params:  params,
	}
	req.Method = string(mcp.MethodToolsCall)

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "send tools/call request")
	}

	if resp == nil {
		return nil, errors.Error("empty response for tools/call")
	}

	var result mcp.CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, errors.Wrap(err, "decode tool call result")
	}

	return &result, nil
}

// GetToolResultText 从工具调用结果中提取文本
func GetToolResultText(r *mcp.CallToolResult) string {
	var texts []string
	for _, content := range r.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			texts = append(texts, textContent.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// FormatToolResult 格式化工具调用结果
func FormatToolResult(toolName string, result *mcp.CallToolResult, err error) string {
	if err != nil {
		return fmt.Sprintf("工具 %s 调用失败: %v", toolName, err)
	}
	if result.IsError {
		return fmt.Sprintf("工具 %s 返回错误: %s", toolName, GetToolResultText(result))
	}
	return GetToolResultText(result)
}

// Close 关闭客户端连接
func (c *MCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initialized = false
	c.sessionURL = ""
	if c.sseBody != nil {
		c.sseBody.Close()
		c.sseBody = nil
	}
	return nil
}
