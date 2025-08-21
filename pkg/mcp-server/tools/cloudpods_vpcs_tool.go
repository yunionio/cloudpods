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

type CloudpodsVPCsTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsVPCsTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsVPCsTool {
	return &CloudpodsVPCsTool{
		adapter: adapter,
		logger:  logger,
	}
}

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

func (c *CloudpodsVPCsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 20
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
	cloudRegionID := req.GetString("cloudregion_id", "")

	c.logger.WithFields(logrus.Fields{
		"limit":          limit,
		"offset":         offset,
		"search":         search,
		"cloudregion_id": cloudRegionID,
	}).Info("开始查询Cloudpods VPC列表")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	vpcsResponse, err := c.adapter.ListVPCs(limit, offset, search, cloudRegionID, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询VPC列表失败")
		return nil, fmt.Errorf("查询VPC列表失败: %w", err)
	}

	formattedResult := c.formatVPCsResult(vpcsResponse, limit, offset, search, cloudRegionID)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsVPCsTool) GetName() string {
	return "cloudpods_list_vpcs"
}

func (c *CloudpodsVPCsTool) formatVPCsResult(response *models.VpcListResponse, limit, offset int, search, cloudRegionID string) map[string]interface{} {
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

	formatted["summary"] = map[string]interface{}{
		"total_vpcs":     response.Total,
		"returned_count": len(response.Vpcs),
		"has_more":       response.Total > int64(offset+len(response.Vpcs)),
		"next_offset":    offset + len(response.Vpcs),
	}

	return formatted
}
