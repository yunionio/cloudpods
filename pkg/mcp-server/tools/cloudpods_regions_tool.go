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
	"github.com/sirupsen/logrus"
	"strconv"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

type CloudpodsRegionsTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsRegionsTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsRegionsTool {
	return &CloudpodsRegionsTool{
		adapter: adapter,
		logger:  logger,
	}
}

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

func (c *CloudpodsRegionsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	provider := req.GetString("provider", "")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	regionsResponse, err := c.adapter.ListCloudRegions(ctx, limit, offset, search, provider, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询区域列表失败")
		return nil, fmt.Errorf("查询区域列表失败: %w", err)
	}

	formattedResult := c.formatRegionsResult(regionsResponse, limit, offset, search, provider)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsRegionsTool) formatRegionsResult(response *models.CloudregionListResponse, limit, offset int, search, provider string) map[string]interface{} {
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

	for _, region := range response.Cloudregions {
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
		formatted["cloudregions"] = append(formatted["cloudregions"].([]map[string]interface{}), regionInfo)
	}

	formatted["summary"] = map[string]interface{}{
		"total_cloudregions": response.Total,
		"returned_count":     len(response.Cloudregions),
		"has_more":           response.Total > int64(offset+len(response.Cloudregions)),
		"next_offset":        offset + len(response.Cloudregions),
	}

	return formatted
}

func (c *CloudpodsRegionsTool) GetName() string {
	return "cloudpods_list_regions"
}
