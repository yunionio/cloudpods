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
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	models "yunion.io/x/onecloud/pkg/mcp-server/internal/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

type CloudpodsServersTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServersTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServersTool {
	return &CloudpodsServersTool{
		adapter: adapter,
		logger:  logger,
	}
}

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

func (c *CloudpodsServersTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 50
	if limitStr := req.GetString("limit", ""); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := req.GetString("offset", ""); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	search := req.GetString("search", "")
	status := req.GetString("status", "")

	c.logger.WithFields(logrus.Fields{
		"limit":  limit,
		"offset": offset,
		"search": search,
		"status": status,
	}).Info("开始查询Cloudpods虚拟机列表")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	serversResponse, err := c.adapter.ListServers(ctx, limit, offset, search, status, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询虚拟机列表失败")
		return nil, fmt.Errorf("查询虚拟机列表失败: %w", err)
	}

	formattedResult := c.formatServersResult(serversResponse, limit, offset, search, status)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServersTool) GetName() string {
	return "cloudpods_list_servers"
}

func (c *CloudpodsServersTool) formatServersResult(response *models.ServerListResponse, limit int, offset int, search string, status string) map[string]interface{} {
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"search": search,
			"status": status,
			"total":  response.Total,
			"count":  len(response.Servers),
		},
		"servers": make([]map[string]interface{}, 0, len(response.Servers)),
	}

	for _, server := range response.Servers {
		memoryGB := float64(server.VmemSize) / 1024

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

	formatted["summary"] = map[string]interface{}{
		"total_servers":  response.Total,
		"returned_count": len(response.Servers),
	}

	return formatted
}
