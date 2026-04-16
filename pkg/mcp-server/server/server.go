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

package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
	"yunion.io/x/onecloud/pkg/mcp-server/registry"
	"yunion.io/x/onecloud/pkg/mcp-server/tools"
)

// CloudpodsMCPServer 是 MCP 服务器的核心结构体，包含配置、日志、MCP 实例、注册中心和工具列表
type CloudpodsMCPServer struct {
	mcpServer *server.MCPServer
	registry  *registry.Registry
	tools     []tools.Tool
}

// NewServer 创建一个新的 Cloudpods MCP 服务器实例，初始化 MCP 服务器和注册中心，并创建所有工具
func NewServer() *CloudpodsMCPServer {

	// 创建mcp server对象
	mcpServer := server.NewMCPServer(
		options.Options.MCPServerName,
		options.Options.MCPServerVersion,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// 创建注册中心对象
	reg := registry.NewRegistry()

	var allTools []tools.Tool

	// 创建mcclient sdk的适配器对象
	adapter := adapters.NewCloudpodsAdapter()

	// 创建具体的工具函数对象
	// 用于查询资源的工具函数
	regionsTool := tools.NewCloudpodsRegionsTool(adapter)
	vpcsTool := tools.NewCloudpodsVPCsTool(adapter)
	networksTool := tools.NewCloudpodsNetworksTool(adapter)
	imagesTool := tools.NewCloudpodsImagesTool(adapter)
	skusTool := tools.NewCloudpodsServerSkusTool(adapter)
	storagesTool := tools.NewCloudpodsStoragesTool(adapter)
	serversTool := tools.NewCloudpodsServersTool(adapter)

	// 用于操作资源的工具函数
	serverStartTool := tools.NewCloudpodsServerStartTool(adapter)
	serverStopTool := tools.NewCloudpodsServerStopTool(adapter)
	serverRestartTool := tools.NewCloudpodsServerRestartTool(adapter)
	serverResetPasswordTool := tools.NewCloudpodsServerResetPasswordTool(adapter)
	serverDeleteTool := tools.NewCloudpodsServerDeleteTool(adapter)
	serverCreateTool := tools.NewCloudpodsServerCreateTool(adapter)
	serverMonitorTool := tools.NewCloudpodsServerMonitorTool(adapter)
	serverStatsTool := tools.NewCloudpodsServerStatsTool(adapter)

	// 将所有的工具函数存储到一个切片中
	allTools = append(
		allTools,
		regionsTool,
		vpcsTool,
		networksTool,
		imagesTool,
		skusTool,
		storagesTool,
		serversTool,

		serverStartTool,
		serverStopTool,
		serverRestartTool,
		serverResetPasswordTool,
		serverDeleteTool,
		serverCreateTool,
		serverMonitorTool,
		serverStatsTool,
	)

	return &CloudpodsMCPServer{
		mcpServer: mcpServer,
		registry:  reg,
		tools:     allTools,
	}
}

// Initialize 初始化注册中心和所有工具
func (s *CloudpodsMCPServer) Initialize() error {

	// 初始化工具注册中心
	if err := s.registry.Initialize(s.mcpServer); err != nil {
		return fmt.Errorf("初始化工具注册中心失败: %w", err)
	}

	// 注册内置工具
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("注册内置工具失败: %w", err)
	}

	return nil
}

// registerAllTools 将所有工具注册到注册中心
func (s *CloudpodsMCPServer) registerAllTools() error {
	for _, tool := range s.tools {
		// 注册距离查询工具
		if err := s.registry.RegisterTool(
			tool.GetName(),
			tool.GetTool(),
			tool.Handle,
		); err != nil {
			return fmt.Errorf("注册工具失败: %w", err)
		}
	}

	log.Infof("All tools register completed")
	return nil
}

// Start 以sse模式启动 mcp 服务
func (s *CloudpodsMCPServer) Start() error {
	// 设置 contextFunc 来从 HTTP header 中提取认证信息并放入 context
	// 支持：X-Auth-Token（token）、AK/SK（Cursor 双 header）、X-API-Key（Claude 单 header：token 或 base64(ak:sk)）
	contextFunc := func(ctx context.Context, r *http.Request) context.Context {
		tokenStr := r.Header.Get(api.AUTH_TOKEN_HEADER)
		akStr := r.Header.Get("AK")
		skStr := r.Header.Get("SK")
		apiKey := r.Header.Get("X-API-Key")

		// 1) 优先使用 X-Auth-Token
		if tokenStr != "" {
			if auth.IsAuthed() {
				userCred, err := auth.Verify(ctx, tokenStr)
				if err != nil {
					log.Errorf("Verify token failed: %s", err)
				} else {
					ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
					log.Debugf("UserCred set in context from HTTP header token")
					return ctx
				}
			}
		}

		// 2) Cursor 方式：直接使用 AK、SK 两个 Header
		if akStr != "" && skStr != "" {
			ctx = context.WithValue(ctx, adapters.ContextKeyAK, akStr)
			ctx = context.WithValue(ctx, adapters.ContextKeySK, skStr)
			log.Debugf("AK/SK set in context from headers")
			return ctx
		}

		// 3) Claude 方式：X-API-Key 可为 token，或 base64(ak:sk)
		if apiKey != "" {
			if auth.IsAuthed() {
				if userCred, err := auth.Verify(ctx, apiKey); err == nil {
					ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
					log.Debugf("UserCred set in context from X-API-Key token")
					return ctx
				}
			}
			decoded, err := base64.StdEncoding.DecodeString(apiKey)
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
					ctx = context.WithValue(ctx, adapters.ContextKeyAK, parts[0])
					ctx = context.WithValue(ctx, adapters.ContextKeySK, parts[1])
					log.Debugf("AK/SK set in context from X-API-Key base64(ak:sk)")
					return ctx
				}
			}
		}

		return ctx
	}

	mux := http.NewServeMux()

	sseServer := server.NewSSEServer(
		s.mcpServer,
		server.WithSSEContextFunc(contextFunc),
		server.WithHTTPServer(&http.Server{Handler: mux}),
	)
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		appsrv.VersionHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.StatisticHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		appsrv.PingHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/worker_stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.WorkerStatsHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/process_stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.ProcessStatsHandler(context.Background(), w, r)
	})
	mux.Handle(sseServer.CompleteSsePath(), sseServer.SSEHandler())
	mux.Handle(sseServer.CompleteMessagePath(), sseServer.MessageHandler())

	if err := sseServer.Start(fmt.Sprintf("%s:%d", options.Options.Address, options.Options.Port)); err != nil {
		return err
	}
	log.Infof("Start mcp server successfully")

	return nil
}

// StartStdio  以stdio模式启动 mcp 服务
func (s *CloudpodsMCPServer) StartStdio() error {

	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		return err
	}
	log.Infof("Start mcp server successfully")
	return nil
}
