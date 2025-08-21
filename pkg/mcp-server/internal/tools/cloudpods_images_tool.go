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
	"strings"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/models"
)

type CloudpodsImagesTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsImagesTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsImagesTool {
	return &CloudpodsImagesTool{
		adapter: adapter,
		logger:  logger,
	}
}

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

func (c *CloudpodsImagesTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	var osTypes []string
	if osTypesStr := req.GetString("os_types", ""); osTypesStr != "" {
		osTypes = strings.Split(osTypesStr, ",")
		for i, osType := range osTypes {
			osTypes[i] = strings.TrimSpace(osType)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"limit":    limit,
		"offset":   offset,
		"search":   search,
		"os_types": osTypes,
	}).Info("开始查询Cloudpods镜像列表")

	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	imagesResponse, err := c.adapter.ListImages(limit, offset, search, osTypes, ak, sk)
	if err != nil {
		c.logger.WithError(err).Error("查询镜像列表失败")
		return nil, fmt.Errorf("查询镜像列表失败: %w", err)
	}

	formattedResult := c.formatImagesResult(imagesResponse, limit, offset, search, osTypes)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsImagesTool) GetName() string {
	return "cloudpods_list_images"
}

func (c *CloudpodsImagesTool) formatImagesResult(response *models.ImageListResponse, limit, offset int, search string, osTypes []string) map[string]interface{} {
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"search":   search,
			"os_types": osTypes,
			"total":    response.Total,
			"count":    len(response.Images),
		},
		"images": make([]map[string]interface{}, 0, len(response.Images)),
	}

	for _, image := range response.Images {
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
		formatted["images"] = append(formatted["images"].([]map[string]interface{}), imageInfo)
	}

	formatted["summary"] = map[string]interface{}{
		"total_images":   response.Total,
		"returned_count": len(response.Images),
		"has_more":       response.Total > int64(offset+len(response.Images)),
		"next_offset":    offset + len(response.Images),
	}

	return formatted
}
