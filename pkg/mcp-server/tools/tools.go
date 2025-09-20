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

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool 是所有工具的接口，定义了工具的基本方法
// GetTool 返回 MCP 工具定义
// Handle 处理工具调用请求
// GetName 返回工具名称
type Tool interface {
	GetTool() mcp.Tool
	Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	GetName() string
}
