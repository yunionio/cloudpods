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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// CloudpodsServersTool 是用于查询 Cloudpods 虚拟机实例列表的工具
type CloudpodsServersTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServersTool 创建一个新的 Cloudpods 虚拟机查询工具
// adapter: 用于与Cloudpods API交互的适配器
// 返回值: CloudpodsServersTool实例指针
func NewCloudpodsServersTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServersTool {
	return &CloudpodsServersTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回查询虚拟机实例列表工具的元数据
// 该工具用于查询Cloudpods虚拟机实例列表，获取虚拟机信息
// limit: 返回结果数量限制，默认为50
// offset: 结果偏移量，默认为0
// search: 按名称或ID模糊搜索
// status: 虚拟机状态，例如：running、stopped、creating等
// ak: 用户登录cloudpods后获取的access key
// sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsServersTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_servers",
		mcp.WithDescription("查询Cloudpods虚拟机实例列表，获取虚拟机信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为50")),
		mcp.WithString("offset", mcp.Description("结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("按名称或ID模糊搜索")),
		mcp.WithString("status", mcp.Description("虚拟机状态，例如：running、stopped、creating等")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理查询 Cloudpods 虚拟机实例列表的请求
// ctx: 控制生命周期的上下文
// req: 包含查询参数的请求对象
// 返回值: 包含虚拟机列表的响应对象和可能的错误
func (c *CloudpodsServersTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取可选参数：返回结果数量限制，如果指定则转换为整数
	limit := 50
	if limitStr := req.GetString("limit", ""); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 获取可选参数：结果偏移量，如果指定则转换为整数
	offset := 0
	if offsetStr := req.GetString("offset", ""); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// 获取可选参数：搜索关键词和虚拟机状态
	search := req.GetString("search", "")
	status := req.GetString("status", "")

	// 获取可选参数：访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器查询虚拟机列表
	serversResponse, err := c.adapter.ListServers(ctx, limit, offset, search, status, ak, sk)
	if err != nil {
		log.Errorf("Fail to query server: %s", err)
		return nil, fmt.Errorf("fail to query server: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatServersResult(serversResponse, limit, offset, search, status)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		log.Errorf("Fail to serialize result: %s", err)
		return nil, fmt.Errorf("fail to serialize result: %w", err)
	}

	// 返回格式化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回工具的名称标识符
// 返回值: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsServersTool) GetName() string {
	return "cloudpods_list_servers"
}

// formatServersResult 格式化虚拟机实例列表查询结果
// response: 原始虚拟机列表响应数据
// limit: 查询限制数量
// offset: 查询偏移量
// search: 搜索关键词
// status: 虚拟机状态
// 返回值: 包含虚拟机列表的格式化结果
func (c *CloudpodsServersTool) formatServersResult(response *models.ServerListResponse, limit int, offset int, search string, status string) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		// 添加查询信息
		"query_info": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"search": search,
			"status": status,
			"total":  response.Total,
			"count":  len(response.Servers),
		},
		// 初始化虚拟机列表
		"servers": make([]map[string]interface{}, 0, len(response.Servers)),
	}

	// 遍历虚拟机列表，构造每个虚拟机的详细信息
	for _, server := range response.Servers {
		// 将内存大小从MB转换为GB
		memoryGB := float64(server.VmemSize) / 1024

		// 构造虚拟机信息
		serverInfo := map[string]interface{}{
			"id":         server.Id,
			"name":       server.Name,
			"status":     server.Status,
			"vcpu_count": server.VcpuCount,
			"vmem_size":  server.VmemSize,
			"memory_gb":  fmt.Sprintf("%.1f GB", memoryGB),
			"os_name":    server.OsName,
			"ips":        server.Ips,
			"host":       server.Host,
			"zone":       server.Zone,
			"region":     server.Cloudregion,
			"created_at": server.CreatedAt,
		}
		formatted["servers"] = append(formatted["servers"].([]map[string]interface{}), serverInfo)
	}

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_servers":  response.Total,
		"returned_count": len(response.Servers),
	}

	return formatted
}
