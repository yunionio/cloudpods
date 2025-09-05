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
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

// CloudpodsStoragesTool 用于查询Cloudpods块存储列表的工具
type CloudpodsStoragesTool struct {
	// adapter 用于与Cloudpods API进行交互
	adapter *adapters.CloudpodsAdapter
	// logger 用于记录日志
	logger  *logrus.Logger
}

// NewCloudpodsStoragesTool 创建一个新的CloudpodsStoragesTool实例
// 
// 参数:
//   - adapter: 用于与Cloudpods API交互的适配器
//   - logger: 用于记录日志的logger实例
// 
// 返回值:
//   - *CloudpodsStoragesTool: CloudpodsStoragesTool实例指针
func NewCloudpodsStoragesTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsStoragesTool {
	return &CloudpodsStoragesTool{
		adapter: adapter,
		logger:  logger,
	}
}

// GetTool 定义并返回查询块存储列表工具的元数据
// 
// 工具用途:
//   查询Cloudpods块存储列表，获取存储资源信息
// 
// 参数说明:
//   - limit: 返回结果数量限制，默认为20
//   - offset: 返回结果偏移量，默认为0
//   - search: 搜索关键词，可以按存储名称搜索
//   - cloudregion_ids: 云区域ID，多个用逗号分隔
//   - zone_ids: 可用区ID，多个用逗号分隔
//   - providers: 云平台提供商，多个用逗号分隔，如：OneCloud,Aliyun,Huawei
//   - storage_types: 存储类型，多个用逗号分隔，如：local,rbd,nfs,cephfs
//   - host_id: 主机ID，过滤关联指定主机的存储
//   - ak: 用户登录cloudpods后获取的access key
//   - sk: 用户登录cloudpods后获取的secret key
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

// Handle 处理查询块存储列表的请求
// 
// 参数:
//   - ctx: 控制生命周期的上下文
//   - req: 包含查询参数的请求对象
// 
// 返回值:
//   - *mcp.CallToolResult: 包含块存储列表的响应对象
//   - error: 可能的错误信息
func (c *CloudpodsStoragesTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// 获取可选参数：云区域ID列表
	var cloudregionIds []string
	if cloudregionIdsStr := req.GetString("cloudregion_ids", ""); cloudregionIdsStr != "" {
		cloudregionIds = strings.Split(cloudregionIdsStr, ",")
		for i, id := range cloudregionIds {
			cloudregionIds[i] = strings.TrimSpace(id)
		}
	}

	// 获取可选参数：可用区ID列表
	var zoneIds []string
	if zoneIdsStr := req.GetString("zone_ids", ""); zoneIdsStr != "" {
		zoneIds = strings.Split(zoneIdsStr, ",")
		for i, id := range zoneIds {
			zoneIds[i] = strings.TrimSpace(id)
		}
	}

	// 获取可选参数：云平台提供商列表
	var providers []string
	if providersStr := req.GetString("providers", ""); providersStr != "" {
		providers = strings.Split(providersStr, ",")
		for i, provider := range providers {
			providers[i] = strings.TrimSpace(provider)
		}
	}

	// 获取可选参数：存储类型列表
	var storageTypes []string
	if storageTypesStr := req.GetString("storage_types", ""); storageTypesStr != "" {
		storageTypes = strings.Split(storageTypesStr, ",")
		for i, storageType := range storageTypes {
			storageTypes[i] = strings.TrimSpace(storageType)
		}
	}

	// 获取可选参数：主机ID
	hostId := req.GetString("host_id", "")

	// 记录查询块存储列表的日志
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

	// 获取可选参数：访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器查询块存储列表
	storagesResponse, err := c.adapter.ListStorages(limit, offset, search, cloudregionIds, zoneIds, providers, storageTypes, hostId, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询Cloudpods块存储列表失败")
		return nil, fmt.Errorf("查询块存储列表失败: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatStoragesResult(storagesResponse, limit, offset, search, cloudregionIds, zoneIds, providers, storageTypes, hostId)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回工具的名称标识符
// 
// 返回值:
//   - string: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsStoragesTool) GetName() string {
	return "cloudpods_list_storages"
}

// formatStoragesResult 格式化块存储列表的响应结果
//
// 参数:
//   - response: 原始响应数据
//   - limit: 查询限制
//   - offset: 查询偏移量
//   - search: 搜索关键词
//   - cloudregionIds: 云区域ID列表
//   - zoneIds: 可用区ID列表
//   - providers: 云平台提供商列表
//   - storageTypes: 存储类型列表
//   - hostId: 主机ID
//
// 返回值:
//   - map[string]interface{}: 包含块存储列表的格式化结果
func (c *CloudpodsStoragesTool) formatStoragesResult(
	response *models.StorageListResponse,
	limit, offset int,
	search string,
	cloudregionIds, zoneIds, providers, storageTypes []string,
	hostId string,
) map[string]interface{} {
	// 初始化格式化结果结构
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

	// 遍历块存储列表，构造每个块存储的详细信息
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

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_storages": response.Total, // 总存储数量
		"returned_count": len(response.Storages), // 当前返回的存储数量
		"has_more":       response.Total > int64(offset+len(response.Storages)), // 是否还有更多数据
		"next_offset":    offset + len(response.Storages), // 下一页的偏移量
	}

	return formatted
}
