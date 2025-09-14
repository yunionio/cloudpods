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
	"fmt"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
	"yunion.io/x/onecloud/pkg/mcp-server/registry"
	"yunion.io/x/onecloud/pkg/mcp-server/tools"
)

// CloudpodsMCPServer 是 MCP 服务器的核心结构体，包含配置、日志、MCP 实例、注册中心和工具列表
type CloudpodsMCPServer struct {
	logger    *logrus.Logger
	mcpServer *server.MCPServer
	registry  *registry.Registry
	tools     []tools.Tool
}

// NewServer 创建一个新的 Cloudpods MCP 服务器实例，初始化 MCP 服务器和注册中心，并创建所有工具
func NewServer(logger *logrus.Logger) *CloudpodsMCPServer {

	// 创建mcp server对象
	mcpServer := server.NewMCPServer(
		options.Options.MCPServerName,
		options.Options.MCPServerVersion,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// 创建注册中心对象
	reg := registry.NewRegistry(logger)

	var allTools []tools.Tool

	// 创建mcclient sdk的适配器对象
	adapter := adapters.NewCloudpodsAdapter(logger)

	// 创建具体的工具函数对象
	// 用于查询资源的工具函数
	regionsTool := tools.NewCloudpodsRegionsTool(adapter, logger)
	vpcsTool := tools.NewCloudpodsVPCsTool(adapter, logger)
	networksTool := tools.NewCloudpodsNetworksTool(adapter, logger)
	imagesTool := tools.NewCloudpodsImagesTool(adapter, logger)
	skusTool := tools.NewCloudpodsServerSkusTool(adapter, logger)
	storagesTool := tools.NewCloudpodsStoragesTool(adapter, logger)
	serversTool := tools.NewCloudpodsServersTool(adapter, logger)

	// 用于操作资源的工具函数
	serverStartTool := tools.NewCloudpodsServerStartTool(adapter, logger)
	serverStopTool := tools.NewCloudpodsServerStopTool(adapter, logger)
	serverRestartTool := tools.NewCloudpodsServerRestartTool(adapter, logger)
	serverResetPasswordTool := tools.NewCloudpodsServerResetPasswordTool(adapter, logger)
	serverDeleteTool := tools.NewCloudpodsServerDeleteTool(adapter, logger)
	serverCreateTool := tools.NewCloudpodsServerCreateTool(adapter, logger)
	serverMonitorTool := tools.NewCloudpodsServerMonitorTool(adapter, logger)
	serverStatsTool := tools.NewCloudpodsServerStatsTool(adapter, logger)

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
		logger:    logger,
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

	s.logger.WithField("total_tools", len(s.tools)).Info("所有工具注册完成")
	return nil
}

// Start 以sse模式启动 mcp 服务
func (s *CloudpodsMCPServer) Start() error {

	if err := server.NewSSEServer(s.mcpServer).Start(fmt.Sprintf("%s:%d", options.Options.Server.Host, options.Options.Server.Port)); err != nil {
		return err
	}
	s.logger.WithField("address", "mcp server stdio").Info("启动mcp server")

	return nil
}

// StartStdio  以stdio模式启动 mcp 服务
func (s *CloudpodsMCPServer) StartStdio() error {

	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		return err
	}
	s.logger.WithField("address", "mcp server stdio").Info("启动mcp server")
	return nil
}
