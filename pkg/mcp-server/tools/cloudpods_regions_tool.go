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
	"github.com/mark3labs/mcp-go/mcp"
	"strconv"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// CloudpodsRegionsTool 是用于查询 Cloudpods 区域列表的工具
type CloudpodsRegionsTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsRegionsTool 创建一个新的 Cloudpods 区域查询工具
// adapter: 用于与 Cloudpods API 进行交互的适配器
// 返回值: 指向新创建的 CloudpodsRegionsTool 实例的指针
func NewCloudpodsRegionsTool(adapter *adapters.CloudpodsAdapter) *CloudpodsRegionsTool {
	return &CloudpodsRegionsTool{
		adapter: adapter,
	}
}

// GetTool 返回 MCP 工具定义，用于查询 Cloudpods 区域列表
// 该工具用于查询Cloudpods中的区域列表，获取所有可用的云区域信息
// 支持的参数包括：
// - limit: 返回结果数量限制，默认为50
// - offset: 返回结果偏移量，默认为0
// - search: 搜索关键词，可以按区域名称搜索
// - provider: 云平台提供商，例如：aws、azure、aliyun等
// - ak: 用户登录cloudpods后获取的access key
// - sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsRegionsTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_regions",
		mcp.WithDescription("查询Cloudpods区域列表，获取所有可用的云区域信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为50")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按区域名称搜索")),
		mcp.WithString("provider", mcp.Description("云平台提供商，例如：aws、azure、aliyun等")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理查询 Cloudpods 区域列表的请求
// ctx: 控制请求生命周期的上下文
// req: 包含查询参数的请求对象
// 返回值: 包含查询结果的工具结果对象或错误信息
func (c *CloudpodsRegionsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 设置默认查询限制为50
	limit := 50
	if limitStr := req.GetString("limit", ""); limitStr != "" {
		// 解析limit参数，如果解析成功且大于0，则使用解析后的值
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 设置默认偏移量为0
	offset := 0
	if offsetStr := req.GetString("offset", ""); offsetStr != "" {
		// 解析offset参数，如果解析成功且大于等于0，则使用解析后的值
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// 获取搜索关键词和提供商参数
	search := req.GetString("search", "")
	provider := req.GetString("provider", "")

	// 获取访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器获取区域列表
	regionsResponse, err := c.adapter.ListCloudRegions(ctx, limit, offset, search, provider, ak, sk)
	if err != nil {
		log.Errorf("Fail to query region: %s", err)
		return nil, fmt.Errorf("fail to query region: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatRegionsResult(regionsResponse, limit, offset, search, provider)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		log.Errorf("Fail to serialize result: %s", err)
		return nil, fmt.Errorf("fail to serialize result: %w", err)
	}
	// 返回格式化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// formatRegionsResult 格式化区域列表查询结果
// response: 从适配器获取的原始区域数据
// limit: 查询限制数量
// offset: 查询偏移量
// search: 搜索关键词
// provider: 云平台提供商
// 返回值: 格式化后的区域列表数据，包含查询信息、区域列表和摘要信息
func (c *CloudpodsRegionsTool) formatRegionsResult(response *models.CloudregionListResponse, limit, offset int, search, provider string) map[string]interface{} {
	// 初始化结果结构，包含查询信息和区域列表
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"search":   search,
			"provider": provider,
			"total":    response.Total,
			"count":    len(response.Cloudregions),
		},
		"cloudregions": make([]map[string]interface{}, 0, len(response.Cloudregions)),
	}

	// 遍历原始区域数据，构造每个区域的详细信息
	for _, region := range response.Cloudregions {
		// 构造单个区域信息
		regionInfo := map[string]interface{}{
			"id":                    region.Id,
			"name":                  region.Name,
			"description":           region.Description,
			"provider":              region.Provider,
			"cloud_env":             region.CloudEnv,
			"environment":           region.Environment,
			"city":                  region.City,
			"country_code":          region.CountryCode,
			"latitude":              region.Latitude,
			"longitude":             region.Longitude,
			"status":                region.Status,
			"enabled":               region.Enabled,
			"external_id":           region.ExternalId,
			"guest_count":           region.GuestCount,
			"guest_increment_count": region.GuestIncrementCount,
			"network_count":         region.NetworkCount,
			"vpc_count":             region.VpcCount,
			"zone_count":            region.ZoneCount,
			"progress":              region.Progress,
			"source":                region.Source,
			"can_delete":            region.CanDelete,
			"can_update":            region.CanUpdate,
			"is_emulated":           region.IsEmulated,
			"metadata":              region.Metadata,
			"created_at":            region.CreatedAt,
			"updated_at":            region.UpdatedAt,
			"imported_at":           region.ImportedAt,
		}
		// 将区域信息添加到结果数组中
		formatted["cloudregions"] = append(formatted["cloudregions"].([]map[string]interface{}), regionInfo)
	}

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_cloudregions": response.Total,
		"returned_count":     len(response.Cloudregions),
		"has_more":           response.Total > int64(offset+len(response.Cloudregions)),
		"next_offset":        offset + len(response.Cloudregions),
	}

	// 返回格式化后的结果
	return formatted
}

// GetName 返回工具名称
// 返回值: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsRegionsTool) GetName() string {
	return "cloudpods_list_regions"
}
