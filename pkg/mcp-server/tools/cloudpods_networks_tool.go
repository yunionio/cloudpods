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

type CloudpodsNetworksTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsNetworksTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsNetworksTool {
	return &CloudpodsNetworksTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsNetworksTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_networks",
		mcp.WithDescription("查询Cloudpods IP子网列表，获取网络配置信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为20")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按网络名称搜索")),
		mcp.WithString("vpc_id", mcp.Description("过滤指定VPC的网络资源")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

func (c *CloudpodsNetworksTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	vpcId := req.GetString("vpc_id", "")

	c.logger.WithFields(logrus.Fields{
		"limit":  limit,
		"offset": offset,
		"search": search,
		"vpc_id": vpcId,
	}).Info("开始查询Cloudpods网络列表")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	networksResponse, err := c.adapter.ListNetworks(limit, offset, search, vpcId, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询网络列表失败")
		return nil, fmt.Errorf("查询网络列表失败: %w", err)
	}

	formattedResult := c.formatNetworksResult(networksResponse, limit, offset, search, vpcId)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsNetworksTool) GetName() string {
	return "cloudpods_list_networks"
}

func (c *CloudpodsNetworksTool) formatNetworksResult(response *models.NetworkListResponse, limit, offset int, search, vpcId string) map[string]interface{} {
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"search": search,
			"vpc_id": vpcId,
			"total":  response.Total,
			"count":  len(response.Networks),
		},
		"networks": make([]map[string]interface{}, 0, len(response.Networks)),
	}

	for _, network := range response.Networks {
		networkInfo := map[string]interface{}{
			"id":                     network.Id,
			"name":                   network.Name,
			"description":            network.Description,
			"status":                 network.Status,
			"guest_ip_start":         network.GuestIpStart,
			"guest_ip_end":           network.GuestIpEnd,
			"guest_ip_mask":          network.GuestIpMask,
			"guest_gateway":          network.GuestGateway,
			"guest_dns":              network.GuestDns,
			"guest_dhcp":             network.GuestDhcp,
			"guest_ntp":              network.GuestNtp,
			"guest_domain":           network.GuestDomain,
			"guest_ip6_start":        network.GuestIp6Start,
			"guest_ip6_end":          network.GuestIp6End,
			"guest_ip6_mask":         network.GuestIp6Mask,
			"guest_gateway6":         network.GuestGateway6,
			"guest_dns6":             network.GuestDns6,
			"guest_domain6":          network.GuestDomain6,
			"vpc":                    network.Vpc,
			"vpc_id":                 network.VpcId,
			"vpc_ext_id":             network.VpcExtId,
			"wire":                   network.Wire,
			"wire_id":                network.WireId,
			"zone":                   network.Zone,
			"zone_id":                network.ZoneId,
			"cloudregion":            network.Cloudregion,
			"cloudregion_id":         network.CloudregionId,
			"region":                 network.Region,
			"region_id":              network.RegionId,
			"provider":               network.Provider,
			"brand":                  network.Brand,
			"cloud_env":              network.CloudEnv,
			"environment":            network.Environment,
			"external_id":            network.ExternalId,
			"account":                network.Account,
			"account_id":             network.AccountId,
			"account_status":         network.AccountStatus,
			"account_health_status":  network.AccountHealthStatus,
			"manager":                network.Manager,
			"manager_id":             network.ManagerId,
			"manager_domain":         network.ManagerDomain,
			"manager_domain_id":      network.ManagerDomainId,
			"manager_project":        network.ManagerProject,
			"manager_project_id":     network.ManagerProjectId,
			"server_type":            network.ServerType,
			"alloc_policy":           network.AllocPolicy,
			"vlan_id":                network.VlanId,
			"bgp_type":               network.BgpType,
			"is_auto_alloc":          network.IsAutoAlloc,
			"is_classic":             network.IsClassic,
			"is_default_vpc":         network.IsDefaultVpc,
			"is_public":              network.IsPublic,
			"is_system":              network.IsSystem,
			"is_emulated":            network.IsEmulated,
			"exit":                   network.Exit,
			"freezed":                network.Freezed,
			"pending_deleted":        network.PendingDeleted,
			"pending_deleted_at":     network.PendingDeletedAt,
			"ports":                  network.Ports,
			"ports_used":             network.PortsUsed,
			"ports6_used":            network.Ports6Used,
			"total":                  network.Total,
			"total6":                 network.Total6,
			"vnics":                  network.Vnics,
			"vnics4":                 network.Vnics4,
			"vnics6":                 network.Vnics6,
			"bm_vnics":               network.BmVnics,
			"bm_reused_vnics":        network.BmReusedVnics,
			"eip_vnics":              network.EipVnics,
			"group_vnics":            network.GroupVnics,
			"lb_vnics":               network.LbVnics,
			"nat_vnics":              network.NatVnics,
			"networkinterface_vnics": network.NetworkinterfaceVnics,
			"rds_vnics":              network.RdsVnics,
			"reserve_vnics4":         network.ReserveVnics4,
			"reserve_vnics6":         network.ReserveVnics6,
			"routes":                 network.Routes,
			"schedtags":              network.Schedtags,
			"additional_wires":       network.AdditionalWires,
			"shared_domains":         network.SharedDomains,
			"shared_projects":        network.SharedProjects,
			"project":                network.Project,
			"project_id":             network.ProjectId,
			"project_domain":         network.ProjectDomain,
			"project_metadata":       network.ProjectMetadata,
			"project_src":            network.ProjectSrc,
			"tenant":                 network.Tenant,
			"tenant_id":              network.TenantId,
			"domain_id":              network.DomainId,
			"public_scope":           network.PublicScope,
			"public_src":             network.PublicSrc,
			"source":                 network.Source,
			"progress":               network.Progress,
			"can_delete":             network.CanDelete,
			"can_update":             network.CanUpdate,
			"metadata":               network.Metadata,
			"created_at":             network.CreatedAt,
			"updated_at":             network.UpdatedAt,
			"imported_at":            network.ImportedAt,
		}
		formatted["networks"] = append(formatted["networks"].([]map[string]interface{}), networkInfo)
	}

	formatted["summary"] = map[string]interface{}{
		"total_networks": response.Total,
		"returned_count": len(response.Networks),
		"has_more":       response.Total > int64(offset+len(response.Networks)),
		"next_offset":    offset + len(response.Networks),
	}

	return formatted
}
