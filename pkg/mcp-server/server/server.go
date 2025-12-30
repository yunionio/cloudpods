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
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/server"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	api "yunion.io/x/onecloud/pkg/apis/identity"
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
	contextFunc := func(ctx context.Context, r *http.Request) context.Context {
		tokenStr := r.Header.Get(api.AUTH_TOKEN_HEADER)
		if len(tokenStr) > 0 {
			if auth.IsAuthed() {
				userCred, err := auth.Verify(ctx, tokenStr)
				if err != nil {
					log.Errorf("Verify token failed: %s", err)
					return ctx
				}
				// 将 userCred 放入 context
				ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
				log.Debugf("UserCred set in context from HTTP header token")
			} else {
				log.Warningf("Auth manager not initialized, skipping token verification")
			}
		}
		return ctx
	}

	sseServer := server.NewSSEServer(
		s.mcpServer,
		server.WithSSEContextFunc(contextFunc),
	)

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
