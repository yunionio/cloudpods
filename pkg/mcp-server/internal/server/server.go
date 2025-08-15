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
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/config"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/registry"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/tools"
)

type CloudpodsMCPServer struct {
	config    *config.Config
	logger    *logrus.Logger
	mcpServer *server.MCPServer
	registry  *registry.Registry
	tools     []tools.Tool
}

func NewServer(cfg *config.Config, logger *logrus.Logger) *CloudpodsMCPServer {

	mcpServer := server.NewMCPServer(
		cfg.MCP.Name,
		cfg.MCP.Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	reg := registry.NewRegistry(logger)

	var allTools []tools.Tool

	adapter := adapters.NewCloudpodsAdapter(cfg, logger)

	regionsTool := tools.NewCloudpodsRegionsTool(adapter, logger)
	vpcsTool := tools.NewCloudpodsVPCsTool(adapter, logger)
	networksTool := tools.NewCloudpodsNetworksTool(adapter, logger)
	imagesTool := tools.NewCloudpodsImagesTool(adapter, logger)
	skusTool := tools.NewCloudpodsServerSkusTool(adapter, logger)
	storagesTool := tools.NewCloudpodsStoragesTool(adapter, logger)
	serversTool := tools.NewCloudpodsServersTool(adapter, logger)

	serverStartTool := tools.NewCloudpodsServerStartTool(adapter, logger)
	serverStopTool := tools.NewCloudpodsServerStopTool(adapter, logger)
	serverRestartTool := tools.NewCloudpodsServerRestartTool(adapter, logger)
	serverResetPasswordTool := tools.NewCloudpodsServerResetPasswordTool(adapter, logger)
	serverDeleteTool := tools.NewCloudpodsServerDeleteTool(adapter, logger)
	serverCreateTool := tools.NewCloudpodsServerCreateTool(adapter, logger)

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
