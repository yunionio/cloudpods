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
	"strings"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

type CloudpodsStoragesTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsStoragesTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsStoragesTool {
	return &CloudpodsStoragesTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsStoragesTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_storages",
		mcp.WithDescription("查询Cloudpods块存储列表，获取存储资源信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为20")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按存储名称搜索")),
		mcp.WithString("cloudregion_ids", mcp.Description("云区域ID，多个用逗号分隔")),
		mcp.WithString("zone_ids", mcp.Description("可用区ID，多个用逗号分隔")),
		mcp.WithString("providers", mcp.Description("云平台提供商，多个用逗号分隔，如：OneCloud,Aliyun,Huawei")),
		mcp.WithString("storage_types", mcp.Description("存储类型，多个用逗号分隔，如：local,rbd,nfs,cephfs")),
		mcp.WithString("host_id", mcp.Description("主机ID，过滤关联指定主机的存储")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

func (c *CloudpodsStoragesTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	var cloudregionIds []string
	if cloudregionIdsStr := req.GetString("cloudregion_ids", ""); cloudregionIdsStr != "" {
		cloudregionIds = strings.Split(cloudregionIdsStr, ",")
		for i, id := range cloudregionIds {
			cloudregionIds[i] = strings.TrimSpace(id)
		}
	}

	var zoneIds []string
	if zoneIdsStr := req.GetString("zone_ids", ""); zoneIdsStr != "" {
		zoneIds = strings.Split(zoneIdsStr, ",")
		for i, id := range zoneIds {
			zoneIds[i] = strings.TrimSpace(id)
		}
	}

	var providers []string
	if providersStr := req.GetString("providers", ""); providersStr != "" {
		providers = strings.Split(providersStr, ",")
		for i, provider := range providers {
			providers[i] = strings.TrimSpace(provider)
		}
	}

	var storageTypes []string
	if storageTypesStr := req.GetString("storage_types", ""); storageTypesStr != "" {
		storageTypes = strings.Split(storageTypesStr, ",")
		for i, storageType := range storageTypes {
			storageTypes[i] = strings.TrimSpace(storageType)
		}
	}

	hostId := req.GetString("host_id", "")

	c.logger.WithFields(logrus.Fields{
		"limit":           limit,
		"offset":          offset,
		"search":          search,
		"cloudregion_ids": cloudregionIds,
		"zone_ids":        zoneIds,
		"providers":       providers,
		"storage_types":   storageTypes,
		"host_id":         hostId,
	}).Info("开始查询Cloudpods块存储列表")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	storagesResponse, err := c.adapter.ListStorages(limit, offset, search, cloudregionIds, zoneIds, providers, storageTypes, hostId, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询块存储列表失败")
		return nil, fmt.Errorf("查询块存储列表失败: %w", err)
	}

	formattedResult := c.formatStoragesResult(storagesResponse, limit, offset, search, cloudregionIds, zoneIds, providers, storageTypes, hostId)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsStoragesTool) GetName() string {
	return "cloudpods_list_storages"
}

func (c *CloudpodsStoragesTool) formatStoragesResult(
	response *models.StorageListResponse,
	limit, offset int,
	search string,
	cloudregionIds, zoneIds, providers, storageTypes []string,
	hostId string,
) map[string]interface{} {
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":           limit,
			"offset":          offset,
			"search":          search,
			"cloudregion_ids": cloudregionIds,
			"zone_ids":        zoneIds,
			"providers":       providers,
			"storage_types":   storageTypes,
			"host_id":         hostId,
			"total":           response.Total,
			"count":           len(response.Storages),
		},
		"storages": make([]map[string]interface{}, 0, len(response.Storages)),
	}

	for _, storage := range response.Storages {
		capacityGB := float64(storage.Capacity) / 1024
		usedCapacityGB := float64(storage.UsedCapacity) / 1024
		freeCapacityGB := float64(storage.FreeCapacity) / 1024
		actualUsedGB := float64(storage.ActualCapacityUsed) / 1024

		storageInfo := map[string]interface{}{
			"id":                    storage.Id,
			"name":                  storage.Name,
			"description":           storage.Description,
			"status":                storage.Status,
			"enabled":               storage.Enabled,
			"storage_type":          storage.StorageType,
			"medium_type":           storage.MediumType,
			"provider":              storage.Provider,
			"brand":                 storage.Brand,
			"cloud_env":             storage.CloudEnv,
			"cloudregion":           storage.Cloudregion,
			"cloudregion_id":        storage.CloudregionId,
			"zone":                  storage.Zone,
			"zone_id":               storage.ZoneId,
			"zone_ext_id":           storage.ZoneExtId,
			"capacity_mb":           storage.Capacity,
			"capacity_gb":           fmt.Sprintf("%.2f GB", capacityGB),
			"used_capacity_mb":      storage.UsedCapacity,
			"used_capacity_gb":      fmt.Sprintf("%.2f GB", usedCapacityGB),
			"free_capacity_mb":      storage.FreeCapacity,
			"free_capacity_gb":      fmt.Sprintf("%.2f GB", freeCapacityGB),
			"actual_capacity_used":  storage.ActualCapacityUsed,
			"actual_used_gb":        fmt.Sprintf("%.2f GB", actualUsedGB),
			"virtual_capacity":      storage.VirtualCapacity,
			"waste_capacity":        storage.WasteCapacity,
			"reserved":              storage.Reserved,
			"commit_bound":          storage.CommitBound,
			"commit_rate":           storage.CommitRate,
			"cmtbound":              storage.Cmtbound,
			"is_sys_disk_store":     storage.IsSysDiskStore,
			"is_public":             storage.IsPublic,
			"is_emulated":           storage.IsEmulated,
			"disk_count":            storage.DiskCount,
			"host_count":            storage.HostCount,
			"snapshot_count":        storage.SnapshotCount,
			"master_host":           storage.MasterHost,
			"master_host_name":      storage.MasterHostName,
			"storagecache_id":       storage.StoragecacheId,
			"account":               storage.Account,
			"account_id":            storage.AccountId,
			"account_status":        storage.AccountStatus,
			"account_health_status": storage.AccountHealthStatus,
			"account_read_only":     storage.AccountReadOnly,
			"manager":               storage.Manager,
			"manager_id":            storage.ManagerId,
			"manager_domain":        storage.ManagerDomain,
			"manager_domain_id":     storage.ManagerDomainId,
			"manager_project":       storage.ManagerProject,
			"manager_project_id":    storage.ManagerProjectId,
			"external_id":           storage.ExternalId,
			"source":                storage.Source,
			"region":                storage.Region,
			"region_id":             storage.RegionId,
			"region_ext_id":         storage.RegionExtId,
			"region_external_id":    storage.RegionExternalId,
			"environment":           storage.Environment,
			"domain_id":             storage.DomainId,
			"domain_src":            storage.DomainSrc,
			"project_domain":        storage.ProjectDomain,
			"public_scope":          storage.PublicScope,
			"public_src":            storage.PublicSrc,
			"shared_domains":        storage.SharedDomains,
			"shared_projects":       storage.SharedProjects,
			"schedtags":             storage.Schedtags,
			"hosts":                 storage.Hosts,
			"storage_conf":          storage.StorageConf,
			"metadata":              storage.Metadata,
			"progress":              storage.Progress,
			"can_delete":            storage.CanDelete,
			"can_update":            storage.CanUpdate,
			"update_version":        storage.UpdateVersion,
			"created_at":            storage.CreatedAt,
			"updated_at":            storage.UpdatedAt,
			"imported_at":           storage.ImportedAt,
		}
		formatted["storages"] = append(formatted["storages"].([]map[string]interface{}), storageInfo)
	}

	formatted["summary"] = map[string]interface{}{
		"total_storages": response.Total,
		"returned_count": len(response.Storages),
		"has_more":       response.Total > int64(offset+len(response.Storages)),
		"next_offset":    offset + len(response.Storages),
	}

	return formatted
}
