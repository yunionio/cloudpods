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
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/llm/options"
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
	closed      atomic.Bool
	userCred    mcclient.TokenCredential

	// requestTimeout 单次 JSON-RPC（含 tools/call）等待 SSE 回包的上限。
	// 须覆盖 climc_server_create 的 forecast+等待（ServerCreateWaitSeconds），且小于整段 chat 超时。
	requestTimeout time.Duration

	pendingReqs map[int64]chan *rawMCPResponse
	reqMu       sync.Mutex
}

// NewMCPClient 创建一个新的 MCP 客户端。
// timeout 为单次 RPC 等待 SSE 响应的超时；SSE 长连接本身不设整体 Timeout，避免读 body 被提前掐断。
func NewMCPClient(serverURL string, timeout time.Duration, userCred mcclient.TokenCredential) *MCPClient {
	if timeout <= 0 {
		timeout = time.Duration(options.Options.MCPAgentTimeout) * time.Second
	}
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	return &MCPClient{
		serverURL: strings.TrimSuffix(serverURL, "/"),
		client: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
				ResponseHeaderTimeout: 60 * time.Second,
			},
		},
		requestTimeout: timeout,
		userCred:       userCred,
		pendingReqs:    make(map[int64]chan *rawMCPResponse),
	}
}

// setAuthHeaders 将当前用户 token 写入请求头，供 mcp-server SSE/message 鉴权。
func (c *MCPClient) setAuthHeaders(req *http.Request) {
	if c.userCred == nil {
		return
	}
	if tok := strings.TrimSpace(c.userCred.GetTokenString()); tok != "" {
		req.Header.Set(identity.AUTH_TOKEN_HEADER, tok)
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
	c.setAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "connect to SSE")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return errors.Errorf("SSE connection failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.sseBody = resp.Body

	// Channel to signal session URL found
	done := make(chan struct{})
	var initErr error

	// 读取 endpoint 事件获取 session URL
	go func() {
		reader := bufio.NewReaderSize(c.sseBody, 1024*1024)
		foundSession := false
		var dataLines []string
		defer func() {
			if !foundSession {
				select {
				case <-done:
				default:
					close(done)
				}
			}
		}()

		flushData := func() {
			if len(dataLines) == 0 {
				return
			}
			data := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]
			if !foundSession {
				if strings.Contains(data, "/message") {
					c.sessionURL = c.serverURL + data
					log.Infof("MCP Client initialized with session URL: %s", c.sessionURL)
					foundSession = true
					close(done)
				}
				return
			}
			// 忽略服务端发起的请求/通知（如 keepalive ping），避免 ID 与 pending tools/call 冲突
			var probe struct {
				JSONRPC string          `json:"jsonrpc"`
				Method  string          `json:"method"`
				Result  json.RawMessage `json:"result"`
				Error   *mcpError       `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &probe); err != nil || probe.JSONRPC != mcp.JSONRPC_VERSION {
				return
			}
			if probe.Method != "" && len(probe.Result) == 0 && probe.Error == nil {
				return
			}
			if len(probe.Result) == 0 && probe.Error == nil {
				return
			}

			var resp rawMCPResponse
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				return
			}
			var reqID int64
			if idVal, ok := resp.ID.Value().(int64); ok {
				reqID = idVal
			} else if idVal, ok := resp.ID.Value().(float64); ok {
				reqID = int64(idVal)
			} else {
				return
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

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if !foundSession {
					initErr = err
				} else if !isExpectedSSEClose(err) && !c.closed.Load() {
					log.Warningf("SSE connection closed: %v", err)
				}
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				flushData()
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
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
			Name:    fmt.Sprintf("%s-mcp-agent", options.ResolvedPlatformName()),
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

	// 如果响应为空，等待 SSE 推送（/message 常返回空 body，结果走 SSE）
	wait := c.requestTimeout
	if wait <= 0 {
		wait = 3 * time.Minute
	}
	if deadline, ok := ctx.Deadline(); ok {
		remain := time.Until(deadline)
		if remain <= 0 {
			return nil, ctx.Err()
		}
		if remain < wait {
			wait = remain
		}
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case mcpResp := <-respChan:
		log.Debugf("MCP response (SSE): ID=%v", mcpResp.ID)
		if mcpResp.Error != nil {
			return nil, errors.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
		}
		return mcpResp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.Errorf("timeout waiting for SSE response after %s", wait)
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
	if len(resp.Result) == 0 {
		if resp.Error != nil {
			return nil, errors.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return nil, errors.Errorf("empty result for tools/call (id=%v); possible SSE keepalive/ping ID collision", resp.ID)
	}

	var result mcp.CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, errors.Wrapf(err, "decode tool call result (len=%d)", len(resp.Result))
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

// isExpectedSSEClose 判断是否为正常关闭（主动 Close / EOF / 连接已关）
func isExpectedSSEClose(err error) bool {
	if err == nil {
		return true
	}
	if errors.Cause(err) == io.EOF || errors.Cause(err) == net.ErrClosed {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "closed network connection") ||
		strings.Contains(msg, "http: read on closed response body")
}

// Close 关闭客户端连接
func (c *MCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed.Store(true)
	c.initialized = false
	c.sessionURL = ""
	if c.sseBody != nil {
		_ = c.sseBody.Close()
		c.sseBody = nil
	}
	return nil
}
