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

// CloudpodsImagesTool 是一个用于查询 Cloudpods 镜像列表的工具
// 它封装了 Cloudpods 适配器和日志记录器
type CloudpodsImagesTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsImagesTool 创建一个新的 CloudpodsImagesTool 实例
// 参数:
//   - adapter: Cloudpods 适配器实例，用于与 Cloudpods API 交互
//
// 返回值:
//   - *CloudpodsImagesTool: 新创建的 CloudpodsImagesTool 实例
func NewCloudpodsImagesTool(adapter *adapters.CloudpodsAdapter) *CloudpodsImagesTool {
	return &CloudpodsImagesTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回 Cloudpods 镜像列表查询工具的元数据
// 该工具允许用户查询 Cloudpods 中的磁盘镜像列表，并支持多种查询参数
// 返回值:
//   - mcp.Tool: 定义了工具名称、描述和参数的工具对象
func (c *CloudpodsImagesTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_list_images",
		mcp.WithDescription("查询Cloudpods磁盘镜像列表，获取系统镜像信息"),
		mcp.WithString("limit", mcp.Description("返回结果数量限制，默认为20")),
		mcp.WithString("offset", mcp.Description("返回结果偏移量，默认为0")),
		mcp.WithString("search", mcp.Description("搜索关键词，可以按镜像名称搜索")),
		mcp.WithString("os_types", mcp.Description("操作系统类型，多个用逗号分隔，如：Linux,Windows,FreeBSD")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理 Cloudpods 镜像列表查询请求
// 该方法解析请求参数，调用适配器查询镜像列表，并格式化返回结果
// 参数:
//   - ctx: 上下文对象，用于控制请求生命周期
//   - req: 工具调用请求对象，包含查询参数
//
// 返回值:
//   - *mcp.CallToolResult: 格式化后的镜像列表查询结果
//   - error: 如果查询过程中发生错误，则返回相应的错误信息
func (c *CloudpodsImagesTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 设置默认的查询结果数量限制为20
	limit := 20
	// 如果请求中包含limit参数且为有效正整数，则使用该值
	if limitStr := req.GetString("limit", ""); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 设置默认的查询偏移量为0
	offset := 0
	// 如果请求中包含offset参数且为有效非负整数，则使用该值
	if offsetStr := req.GetString("offset", ""); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// 获取搜索关键词参数
	search := req.GetString("search", "")

	// 解析操作系统类型参数，支持多个类型用逗号分隔
	var osTypes []string
	if osTypesStr := req.GetString("os_types", ""); osTypesStr != "" {
		osTypes = strings.Split(osTypesStr, ",")
		for i, osType := range osTypes {
			osTypes[i] = strings.TrimSpace(osType)
		}
	}

	// 获取访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器查询镜像列表
	imagesResponse, err := c.adapter.ListImages(ctx, limit, offset, search, osTypes, ak, sk)
	if err != nil {
		log.Errorf("Fail to query image: %s", err)
		return nil, fmt.Errorf("fail to query image: %w", err)
	}

	// 格式化查询结果
	formattedResult := c.formatImagesResult(imagesResponse, limit, offset, search, osTypes)

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
// 返回值:
//   - string: 工具名称，用于唯一标识该工具
func (c *CloudpodsImagesTool) GetName() string {
	return "cloudpods_list_images"
}

// formatImagesResult 格式化镜像列表查询结果
// 该方法将从适配器获取的原始镜像数据转换为结构化的响应格式，包含查询信息、镜像详情和摘要信息
// 参数:
//   - response: 从适配器获取的原始镜像列表响应数据
//   - limit: 查询结果数量限制
//   - offset: 查询偏移量
//   - search: 搜索关键词
//   - osTypes: 操作系统类型过滤条件
//
// 返回值:
//   - map[string]interface{}: 格式化后的镜像列表数据，包含查询信息、镜像详情和摘要
func (c *CloudpodsImagesTool) formatImagesResult(response *models.ImageListResponse, limit, offset int, search string, osTypes []string) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		// 查询信息部分，包含查询参数和结果统计
		"query_info": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"search":   search,
			"os_types": osTypes,
			"total":    response.Total,
			"count":    len(response.Images),
		},
		// 镜像列表部分，初始化为空数组
		"images": make([]map[string]interface{}, 0, len(response.Images)),
	}

	// 遍历原始镜像数据，提取每个镜像的详细信息
	for _, image := range response.Images {
		// 构造单个镜像的详细信息
		imageInfo := map[string]interface{}{
			"id":                         image.Id,
			"name":                       image.Name,
			"description":                image.Description,
			"status":                     image.Status,
			"disk_format":                image.DiskFormat,
			"size":                       image.Size,
			"checksum":                   image.Checksum,
			"oss_checksum":               image.OssChecksum,
			"fast_hash":                  image.FastHash,
			"location":                   image.Location,
			"os_arch":                    image.OsArch,
			"min_disk":                   image.MinDisk,
			"min_ram":                    image.MinRam,
			"is_data":                    image.IsData,
			"is_guest_image":             image.IsGuestImage,
			"is_public":                  image.IsPublic,
			"is_standard":                image.IsStandard,
			"is_system":                  image.IsSystem,
			"is_emulated":                image.IsEmulated,
			"protected":                  image.Protected,
			"disable_delete":             image.DisableDelete,
			"freezed":                    image.Freezed,
			"pending_deleted":            image.PendingDeleted,
			"pending_deleted_at":         image.PendingDeletedAt,
			"auto_delete_at":             image.AutoDeleteAt,
			"encrypt_alg":                image.EncryptAlg,
			"encrypt_key":                image.EncryptKey,
			"encrypt_key_id":             image.EncryptKeyId,
			"encrypt_key_user":           image.EncryptKeyUser,
			"encrypt_key_user_domain":    image.EncryptKeyUserDomain,
			"encrypt_key_user_domain_id": image.EncryptKeyUserDomainId,
			"encrypt_key_user_id":        image.EncryptKeyUserId,
			"encrypt_status":             image.EncryptStatus,
			"owner":                      image.Owner,
			"project":                    image.Project,
			"project_id":                 image.ProjectId,
			"project_domain":             image.ProjectDomain,
			"project_metadata":           image.ProjectMetadata,
			"project_src":                image.ProjectSrc,
			"tenant":                     image.Tenant,
			"tenant_id":                  image.TenantId,
			"domain_id":                  image.DomainId,
			"public_scope":               image.PublicScope,
			"public_src":                 image.PublicSrc,
			"shared_domains":             image.SharedDomains,
			"shared_projects":            image.SharedProjects,
			"properties":                 image.Properties,
			"metadata":                   image.Metadata,
			"progress":                   image.Progress,
			"can_delete":                 image.CanDelete,
			"can_update":                 image.CanUpdate,
			"update_version":             image.UpdateVersion,
			"created_at":                 image.CreatedAt,
			"updated_at":                 image.UpdatedAt,
		}
		// 将镜像信息添加到结果数组中
		formatted["images"] = append(formatted["images"].([]map[string]interface{}), imageInfo)
	}

	// 构造结果摘要信息
	formatted["summary"] = map[string]interface{}{
		"total_images":   response.Total,
		"returned_count": len(response.Images),
		"has_more":       response.Total > int64(offset+len(response.Images)),
		"next_offset":    offset + len(response.Images),
	}

	// 返回格式化后的完整结果
	return formatted
}
