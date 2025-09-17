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

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// CloudpodsServerSkusTool 用于查询Cloudpods主机套餐规格列表的工具
type CloudpodsServerSkusTool struct {
	// adapter 用于与Cloudpods API进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerSkusTool 创建一个新的CloudpodsServerSkusTool实例
//
// 参数:
//   - adapter: 用于与Cloudpods API交互的适配器
//
// 返回值:
//   - *CloudpodsServerSkusTool: CloudpodsServerSkusTool实例指针
func NewCloudpodsServerSkusTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerSkusTool {
	return &CloudpodsServerSkusTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回查询主机套餐规格列表工具的元数据
//
// 工具用途:
//
//	查询Cloudpods主机套餐规格列表，获取虚拟机规格信息
//
// 参数说明:
//   - limit: 返回结果数量限制，默认为20
//   - offset: 返回结果偏移量，默认为0
//   - search: 搜索关键词，可以按规格名称搜索
//   - cloudregion_ids: 云区域ID，多个用逗号分隔
//   - zone_ids: 可用区ID，多个用逗号分隔
//   - cpu_core_count: CPU核心数，多个用逗号分隔，如：1,2,4,8
//   - memory_size_mb: 内存大小MB，多个用逗号分隔，如：1024,2048,4096
//   - providers: 云平台提供商，多个用逗号分隔，如：OneCloud,Aliyun,Huawei
//   - cpu_arch: CPU架构，多个用逗号分隔，如：x86,arm
//   - ak: 用户登录cloudpods后获取的access key
//   - sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsServerSkusTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_serverskus",
		mcp.WithDescription("查询Cloudpods主机套餐规格列表，获取虚拟机规格信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为20")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按规格名称搜索")),
		mcp.WithString("cloudregion_ids", mcp.Description("云区域ID，多个用逗号分隔")),
		mcp.WithString("zone_ids", mcp.Description("可用区ID，多个用逗号分隔")),
		mcp.WithString("cpu_core_count", mcp.Description("CPU核心数，多个用逗号分隔，如：1,2,4,8")),
		mcp.WithString("memory_size_mb", mcp.Description("内存大小MB，多个用逗号分隔，如：1024,2048,4096")),
		mcp.WithString("providers", mcp.Description("云平台提供商，多个用逗号分隔，如：OneCloud,Aliyun,Huawei")),
		mcp.WithString("cpu_arch", mcp.Description("CPU架构，多个用逗号分隔，如：x86,arm")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理查询主机套餐规格列表的请求
//
// 参数:
//   - ctx: 控制生命周期的上下文
//   - req: 包含查询参数的请求对象
//
// 返回值:
//   - *mcp.CallToolResult: 包含主机套餐规格列表的响应对象
//   - error: 可能的错误信息
func (c *CloudpodsServerSkusTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// 获取可选参数：CPU核心数列表
	var cpuCoreCount []string
	if cpuCoreCountStr := req.GetString("cpu_core_count", ""); cpuCoreCountStr != "" {
		cpuCoreCount = strings.Split(cpuCoreCountStr, ",")
		for i, count := range cpuCoreCount {
			cpuCoreCount[i] = strings.TrimSpace(count)
		}
	}

	// 获取可选参数：内存大小列表（MB）
	var memorySizeMB []string
	if memorySizeMBStr := req.GetString("memory_size_mb", ""); memorySizeMBStr != "" {
		memorySizeMB = strings.Split(memorySizeMBStr, ",")
		for i, size := range memorySizeMB {
			memorySizeMB[i] = strings.TrimSpace(size)
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

	// 获取可选参数：CPU架构列表
	var cpuArch []string
	if cpuArchStr := req.GetString("cpu_arch", ""); cpuArchStr != "" {
		cpuArch = strings.Split(cpuArchStr, ",")
		for i, arch := range cpuArch {
			cpuArch[i] = strings.TrimSpace(arch)
		}
	}

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器查询主机套餐规格列表
	skusResponse, err := c.adapter.ListServerSkus(limit, offset, search, cloudregionIds, zoneIds, cpuCoreCount, memorySizeMB, providers, cpuArch, ak, sk)
	if err != nil {
		log.Errorf("Fail to query server skus: %s", err)
		return nil, fmt.Errorf("fail to query server skus: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatServerSkusResult(skusResponse, limit, offset, search, cloudregionIds, zoneIds, cpuCoreCount, memorySizeMB, providers, cpuArch)

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
func (c *CloudpodsServerSkusTool) GetName() string {
	return "cloudpods_list_serverskus"
}

// formatServerSkusResult 格式化主机套餐规格列表的响应结果
//
// 参数:
//   - response: 原始主机套餐规格列表响应数据
//   - limit: 查询限制数量
//   - offset: 查询偏移量
//   - search: 搜索关键词
//   - cloudregionIds: 云区域ID列表
//   - zoneIds: 可用区ID列表
//   - cpuCoreCount: CPU核心数列表
//   - memorySizeMB: 内存大小列表（MB）
//   - providers: 云平台提供商列表
//   - cpuArch: CPU架构列表
//
// 返回值:
//   - map[string]interface{}: 包含主机套餐规格列表的格式化结果
func (c *CloudpodsServerSkusTool) formatServerSkusResult(
	response *models.ServerSkuListResponse,
	limit, offset int,
	search string,
	cloudregionIds, zoneIds, cpuCoreCount, memorySizeMB, providers, cpuArch []string,
) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":           limit,
			"offset":          offset,
			"search":          search,
			"cloudregion_ids": cloudregionIds,
			"zone_ids":        zoneIds,
			"cpu_core_count":  cpuCoreCount,
			"memory_size_mb":  memorySizeMB,
			"providers":       providers,
			"cpu_arch":        cpuArch,
			"total":           response.Total,
			"count":           len(response.Serverskus),
		},
		"serverskus": make([]map[string]interface{}, 0, len(response.Serverskus)),
	}

	// 遍历主机套餐列表，构造每个主机套餐的详细信息
	for _, sku := range response.Serverskus {
		skuInfo := map[string]interface{}{
			"id":                     sku.Id,
			"name":                   sku.Name,
			"description":            sku.Description,
			"status":                 sku.Status,
			"enabled":                sku.Enabled,
			"provider":               sku.Provider,
			"cloud_env":              sku.CloudEnv,
			"cloudregion":            sku.Cloudregion,
			"cloudregion_id":         sku.CloudregionId,
			"zone":                   sku.Zone,
			"zone_id":                sku.ZoneId,
			"zone_ext_id":            sku.ZoneExtId,
			"cpu_core_count":         sku.CpuCoreCount,
			"memory_size_mb":         sku.MemorySizeMB,
			"cpu_arch":               sku.CpuArch,
			"instance_type_family":   sku.InstanceTypeFamily,
			"instance_type_category": sku.InstanceTypeCategory,
			"local_category":         sku.LocalCategory,
			"sys_disk_type":          sku.SysDiskType,
			"sys_disk_min_size_gb":   sku.SysDiskMinSizeGB,
			"sys_disk_max_size_gb":   sku.SysDiskMaxSizeGB,
			"sys_disk_resizable":     sku.SysDiskResizable,
			"data_disk_types":        sku.DataDiskTypes,
			"data_disk_max_count":    sku.DataDiskMaxCount,
			"attached_disk_count":    sku.AttachedDiskCount,
			"attached_disk_size_gb":  sku.AttachedDiskSizeGB,
			"attached_disk_type":     sku.AttachedDiskType,
			"nic_type":               sku.NicType,
			"nic_max_count":          sku.NicMaxCount,
			"gpu_attachable":         sku.GpuAttachable,
			"gpu_count":              sku.GpuCount,
			"gpu_max_count":          sku.GpuMaxCount,
			"gpu_spec":               sku.GpuSpec,
			"os_name":                sku.OsName,
			"postpaid_status":        sku.PostpaidStatus,
			"prepaid_status":         sku.PrepaidStatus,
			"total_guest_count":      sku.TotalGuestCount,
			"external_id":            sku.ExternalId,
			"source":                 sku.Source,
			"is_emulated":            sku.IsEmulated,
			"region":                 sku.Region,
			"region_id":              sku.RegionId,
			"region_ext_id":          sku.RegionExtId,
			"region_external_id":     sku.RegionExternalId,
			"md5":                    sku.Md5,
			"metadata":               sku.Metadata,
			"progress":               sku.Progress,
			"can_delete":             sku.CanDelete,
			"can_update":             sku.CanUpdate,
			"update_version":         sku.UpdateVersion,
			"created_at":             sku.CreatedAt,
			"updated_at":             sku.UpdatedAt,
			"imported_at":            sku.ImportedAt,
		}
		formatted["serverskus"] = append(formatted["serverskus"].([]map[string]interface{}), skuInfo)
	}

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_serverskus": response.Total,
		"returned_count":   len(response.Serverskus),
		"has_more":         response.Total > int64(offset+len(response.Serverskus)),
		"next_offset":      offset + len(response.Serverskus),
	}

	return formatted
}
