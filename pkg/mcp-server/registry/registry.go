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

package registry

import (
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"sync"
	"yunion.io/x/log"
)

type Registry struct {
	mu          sync.RWMutex
	tools       map[string]*ToolRegistration
	mcpServer   *server.MCPServer
	initialized bool
}

type ToolRegistration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*ToolRegistration),
	}
}

// Initialize 使用MCP服务器初始化注册中心
func (r *Registry) Initialize(mcpServer *server.MCPServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return fmt.Errorf("Fail to init register ")
	}

	r.mcpServer = mcpServer

	// 将所有已注册的工具添加到MCP服务器
	for _, registration := range r.tools {
		r.mcpServer.AddTool(registration.Tool, registration.Handler)
	}

	r.initialized = true

	return nil
}

// RegisterTool 注册单个工具
func (r *Registry) RegisterTool(toolName string, tool mcp.Tool, handler server.ToolHandlerFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolName]; exists {
		return fmt.Errorf("Tool already register: '%s' ", toolName)
	}

	registration := &ToolRegistration{
		Tool:    tool,
		Handler: handler,
	}

	r.tools[toolName] = registration
	log.Infof("Tool register successfully: %s", toolName)

	// 如果MCP服务器已设置，立即注册到服务器
	if r.mcpServer != nil {
		r.mcpServer.AddTool(tool, handler)
	}

	return nil
}
