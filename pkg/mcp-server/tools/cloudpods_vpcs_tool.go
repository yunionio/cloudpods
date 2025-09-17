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

// CloudpodsVPCsTool 用于查询Cloudpods VPC列表的工具
//
// 字段:
//   - adapter: 用于与Cloudpods API进行交互的适配器
type CloudpodsVPCsTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsVPCsTool 创建CloudpodsVPCsTool实例
//
// 参数:
//   - adapter: 用于与Cloudpods API交互的适配器
//
// 返回值:
//   - *CloudpodsVPCsTool: CloudpodsVPCsTool实例指针
func NewCloudpodsVPCsTool(adapter *adapters.CloudpodsAdapter) *CloudpodsVPCsTool {
	return &CloudpodsVPCsTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回查询VPC列表工具的元数据
//
// 工具用途:
//
//	查询Cloudpods VPC列表，获取虚拟私有网络信息
//
// 参数说明:
//   - limit: 返回结果数量限制，默认为20
//   - offset: 返回结果偏移量，默认为0
//   - search: 搜索关键词，可以按VPC名称搜索
//   - cloudregion_id: 过滤指定云区域的VPC资源
//   - ak: 用户登录cloudpods后获取的access key
//   - sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsVPCsTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_vpcs",
		mcp.WithDescription("查询Cloudpods VPC列表，获取虚拟私有网络信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为20")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按VPC名称搜索")),
		mcp.WithString("cloudregion_id", mcp.Description("过滤指定云区域的VPC资源")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理查询VPC列表的请求
//
// 参数:
//   - ctx: 控制生命周期的上下文
//   - req: 包含查询参数的请求对象
//
// 返回值:
//   - *mcp.CallToolResult: 包含VPC列表的响应对象
//   - error: 可能的错误信息
func (c *CloudpodsVPCsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取可选参数：返回结果数量限制，如果指定则转换为整数
	limit := 20
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

	// 获取可选参数：搜索关键词
	search := req.GetString("search", "")
	// 获取可选参数：云区域ID
	cloudRegionID := req.GetString("cloudregion_id", "")

	// 获取可选参数：访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器查询VPC列表
	vpcsResponse, err := c.adapter.ListVPCs(limit, offset, search, cloudRegionID, ak, sk)
	if err != nil {
		log.Errorf("Fail to query vpc: %s", err)
		return nil, fmt.Errorf("fail to query vpc: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatVPCsResult(vpcsResponse, limit, offset, search, cloudRegionID)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		log.Errorf("Fail to serialize result: %s", err)
		return nil, fmt.Errorf("fail to serialize result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回工具的名称标识符
//
// 返回值:
//   - string: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsVPCsTool) GetName() string {
	return "cloudpods_list_vpcs"
}

// formatVPCsResult 格式化VPC列表的响应结果
//
// 参数:
//   - response: 原始响应数据
//   - limit: 查询限制
//   - offset: 查询偏移量
//   - search: 搜索关键词
//   - cloudRegionID: 云区域ID
//
// 返回值:
//   - map[string]interface{}: 包含VPC列表的格式化结果
func (c *CloudpodsVPCsTool) formatVPCsResult(response *models.VpcListResponse, limit, offset int, search, cloudRegionID string) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":          limit,
			"offset":         offset,
			"search":         search,
			"cloudregion_id": cloudRegionID,
			"total":          response.Total,
			"count":          len(response.Vpcs),
		},
		"vpcs": make([]map[string]interface{}, 0, len(response.Vpcs)),
	}

	// 遍历VPC列表，构造每个VPC的详细信息
	for _, vpc := range response.Vpcs {
		vpcInfo := map[string]interface{}{
			"id":                     vpc.Id,
			"name":                   vpc.Name,
			"description":            vpc.Description,
			"cidr_block":             vpc.CidrBlock,
			"cidr_block6":            vpc.CidrBlock6,
			"status":                 vpc.Status,
			"enabled":                vpc.Enabled,
			"is_default":             vpc.IsDefault,
			"is_public":              vpc.IsPublic,
			"provider":               vpc.Provider,
			"brand":                  vpc.Brand,
			"cloud_env":              vpc.CloudEnv,
			"environment":            vpc.Environment,
			"cloudregion":            vpc.Cloudregion,
			"cloudregion_id":         vpc.CloudregionId,
			"region":                 vpc.Region,
			"region_id":              vpc.RegionId,
			"external_id":            vpc.ExternalId,
			"external_access_mode":   vpc.ExternalAccessMode,
			"globalvpc":              vpc.Globalvpc,
			"globalvpc_id":           vpc.GlobalvpcId,
			"account":                vpc.Account,
			"account_id":             vpc.AccountId,
			"account_status":         vpc.AccountStatus,
			"account_health_status":  vpc.AccountHealthStatus,
			"manager":                vpc.Manager,
			"manager_id":             vpc.ManagerId,
			"manager_domain":         vpc.ManagerDomain,
			"manager_domain_id":      vpc.ManagerDomainId,
			"manager_project":        vpc.ManagerProject,
			"manager_project_id":     vpc.ManagerProjectId,
			"network_count":          vpc.NetworkCount,
			"wire_count":             vpc.WireCount,
			"dns_zone_count":         vpc.DnsZoneCount,
			"natgateway_count":       vpc.NatgatewayCount,
			"routetable_count":       vpc.RoutetableCount,
			"accept_vpc_peer_count":  vpc.AcceptVpcPeerCount,
			"request_vpc_peer_count": vpc.RequestVpcPeerCount,
			"direct":                 vpc.Direct,
			"domain_id":              vpc.DomainId,
			"domain_src":             vpc.DomainSrc,
			"project_domain":         vpc.ProjectDomain,
			"public_scope":           vpc.PublicScope,
			"public_src":             vpc.PublicSrc,
			"region_ext_id":          vpc.RegionExtId,
			"region_external_id":     vpc.RegionExternalId,
			"source":                 vpc.Source,
			"progress":               vpc.Progress,
			"shared_domains":         vpc.SharedDomains,
			"shared_projects":        vpc.SharedProjects,
			"can_delete":             vpc.CanDelete,
			"can_update":             vpc.CanUpdate,
			"is_emulated":            vpc.IsEmulated,
			"metadata":               vpc.Metadata,
			"created_at":             vpc.CreatedAt,
			"updated_at":             vpc.UpdatedAt,
			"imported_at":            vpc.ImportedAt,
		}
		formatted["vpcs"] = append(formatted["vpcs"].([]map[string]interface{}), vpcInfo)
	}

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_vpcs":     response.Total,
		"returned_count": len(response.Vpcs),
		"has_more":       response.Total > int64(offset+len(response.Vpcs)),
		"next_offset":    offset + len(response.Vpcs),
	}

	return formatted
}
