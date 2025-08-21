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
	"yunion.io/x/onecloud/pkg/mcp-server/config"
	"yunion.io/x/onecloud/pkg/mcp-server/registry"
	tools2 "yunion.io/x/onecloud/pkg/mcp-server/tools"
)

// CloudpodsMCPServer 是 MCP 服务器的核心结构体，包含配置、日志、MCP 实例、注册中心和工具列表
type CloudpodsMCPServer struct {
	config    *config.Config
	logger    *logrus.Logger
	mcpServer *server.MCPServer
	registry  *registry.Registry
	tools     []tools2.Tool
}

// NewServer 创建一个新的 Cloudpods MCP 服务器实例，初始化 MCP 服务器和注册中心，并创建所有工具
func NewServer(cfg *config.Config, logger *logrus.Logger) *CloudpodsMCPServer {

	mcpServer := server.NewMCPServer(
		cfg.MCP.Name,
		cfg.MCP.Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	reg := registry.NewRegistry(logger)

	var allTools []tools2.Tool

	adapter := adapters.NewCloudpodsAdapter(cfg, logger)

	regionsTool := tools2.NewCloudpodsRegionsTool(adapter, logger)
	vpcsTool := tools2.NewCloudpodsVPCsTool(adapter, logger)
	networksTool := tools2.NewCloudpodsNetworksTool(adapter, logger)
	imagesTool := tools2.NewCloudpodsImagesTool(adapter, logger)
	skusTool := tools2.NewCloudpodsServerSkusTool(adapter, logger)
	storagesTool := tools2.NewCloudpodsStoragesTool(adapter, logger)
	serversTool := tools2.NewCloudpodsServersTool(adapter, logger)

	serverStartTool := tools2.NewCloudpodsServerStartTool(adapter, logger)
	serverStopTool := tools2.NewCloudpodsServerStopTool(adapter, logger)
	serverRestartTool := tools2.NewCloudpodsServerRestartTool(adapter, logger)
	serverResetPasswordTool := tools2.NewCloudpodsServerResetPasswordTool(adapter, logger)
	serverDeleteTool := tools2.NewCloudpodsServerDeleteTool(adapter, logger)
	serverCreateTool := tools2.NewCloudpodsServerCreateTool(adapter, logger)

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
	)

	return &CloudpodsMCPServer{
		config:    cfg,
		logger:    logger,
		mcpServer: mcpServer,
		registry:  reg,
		tools:     allTools,
	}
}

// Initialize 初始化注册中心和所有工具
// Initialize 初始化服务器
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
// registerBuiltinTools 注册内置工具
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

// Start 启动 SSE 服务器
// Start 启动服务器
func (s *CloudpodsMCPServer) Start() error {

	if err := server.NewSSEServer(s.mcpServer).Start("localhost:12005"); err != nil {
		return err
	}
	s.logger.WithField("address", "mcp server stdio").Info("启动mcp server")

	return nil
}
