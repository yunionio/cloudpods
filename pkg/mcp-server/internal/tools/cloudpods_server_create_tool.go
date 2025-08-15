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

type CloudpodsServerCreateTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerCreateTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerCreateTool {
	return &CloudpodsServerCreateTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerCreateTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_create_server",
		mcp.WithDescription("创建Cloudpods虚拟机实例"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("虚拟机名称")),
		mcp.WithString("vcpu_count",
			mcp.Required(),
			mcp.Description("CPU核心数")),
		mcp.WithString("vmem_size",
			mcp.Required(),
			mcp.Description("内存大小(MB)")),
		mcp.WithString("image_id",
			mcp.Required(),
			mcp.Description("镜像ID")),
		mcp.WithString("disk_size",
			mcp.Description("系统盘大小(GB)，不指定则使用镜像默认大小")),
		mcp.WithString("network_id",
			mcp.Required(),
			mcp.Description("网络ID")),
		mcp.WithString("serversku_id",
			mcp.Description("套餐ID，指定后将忽略vcpu_count和vmem_size参数")),
		mcp.WithString("password",
			mcp.Description("虚拟机密码，长度8-30个字符")),
		mcp.WithString("count",
			mcp.Description("创建数量，默认为1")),
		mcp.WithString("auto_start",
			mcp.Description("是否自动启动，默认为true")),
		mcp.WithString("billing_type",
			mcp.Description("计费类型，例如：postpaid、prepaid")),
		mcp.WithString("duration",
			mcp.Description("包年包月时长，例如：1M、1Y")),
		mcp.WithString("description",
			mcp.Description("描述信息")),
		mcp.WithString("hostname",
			mcp.Description("主机名")),
		mcp.WithString("hypervisor",
			mcp.Description("虚拟化技术，如kvm, esxi等，默认为kvm")),
		mcp.WithString("metadata",
			mcp.Description("标签列表，格式为JSON字符串，例如：{\"key1\":\"value1\",\"key2\":\"value2\"}")),
		mcp.WithString("secgroup_id",
			mcp.Description("安全组ID")),
		mcp.WithString("secgroups",
			mcp.Description("安全组ID列表，多个ID用逗号分隔")),
		mcp.WithString("user_data",
			mcp.Description("用户自定义启动脚本")),
		mcp.WithString("keypair_id",
			mcp.Description("秘钥对ID")),
		mcp.WithString("project_id",
			mcp.Description("项目ID")),
		mcp.WithString("zone_id",
			mcp.Description("可用区ID")),
		mcp.WithString("region_id",
			mcp.Description("区域ID")),
		mcp.WithString("disable_delete",
			mcp.Description("是否开启删除保护，默认为true")),
		mcp.WithString("boot_order",
			mcp.Description("启动顺序，如cdn")),
		mcp.WithString("data_disks",
			mcp.Description("数据盘配置，格式为JSON字符串数组，例如：[{\"size\":100,\"disk_type\":\"data\"}]")),
	)
}

func (c *CloudpodsServerCreateTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return nil, err
	}

	imageID, err := req.RequireString("image_id")
	if err != nil {
		return nil, err
	}

	networkID, err := req.RequireString("network_id")
	if err != nil {
		return nil, err
	}

	vcpuCountStr, err := req.RequireString("vcpu_count")
	if err != nil {
		return nil, err
	}
	vcpuCount, err := strconv.ParseInt(vcpuCountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的CPU核心数: %s", vcpuCountStr)
	}

	vmemSizeStr, err := req.RequireString("vmem_size")
	if err != nil {
		return nil, err
	}
	vmemSize, err := strconv.ParseInt(vmemSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的内存大小: %s", vmemSizeStr)
	}

	serverSkuID := req.GetString("serversku_id", "")

	diskSize := int64(0)
	if diskSizeStr := req.GetString("disk_size", ""); diskSizeStr != "" {
		if parsedSize, err := strconv.ParseInt(diskSizeStr, 10, 64); err == nil && parsedSize > 0 {
			diskSize = parsedSize
		}
	}

	password := req.GetString("password", "")
	if password != "" && (len(password) < 8 || len(password) > 30) {
		return nil, fmt.Errorf("密码长度必须在8-30个字符之间")
	}

	count := 1
	if countStr := req.GetString("count", "1"); countStr != "1" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	autoStart := true
	if autoStartStr := req.GetString("auto_start", "true"); autoStartStr == "false" {
		autoStart = false
	}

	disableDelete := true
	if disableDeleteStr := req.GetString("disable_delete", "true"); disableDeleteStr == "false" {
		disableDelete = false
	}

	billingType := req.GetString("billing_type", "")
	duration := req.GetString("duration", "")
	description := req.GetString("description", "")
	hostname := req.GetString("hostname", "")
	hypervisor := req.GetString("hypervisor", "")
	secgroupID := req.GetString("secgroup_id", "")
	userData := req.GetString("user_data", "")
	keypairID := req.GetString("keypair_id", "")
	projectID := req.GetString("project_id", "")
	zoneID := req.GetString("zone_id", "")
	regionID := req.GetString("region_id", "")
	bootOrder := req.GetString("boot_order", "")

	var secgroups []string
	if secgroupsStr := req.GetString("secgroups", ""); secgroupsStr != "" {
		secgroups = strings.Split(secgroupsStr, ",")
	}

	metadata := make(map[string]string)
	if metadataStr := req.GetString("metadata", ""); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return nil, fmt.Errorf("无效的元数据JSON格式: %w", err)
		}
	}

	var dataDisks []models.DiskConfig
	if dataDisksStr := req.GetString("data_disks", ""); dataDisksStr != "" {
		if err := json.Unmarshal([]byte(dataDisksStr), &dataDisks); err != nil {
			return nil, fmt.Errorf("无效的数据盘配置JSON格式: %w", err)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"name":           name,
		"image_id":       imageID,
		"vcpu_count":     vcpuCount,
		"vmem_size":      vmemSize,
		"disk_size":      diskSize,
		"network_id":     networkID,
		"serversku_id":   serverSkuID,
		"count":          count,
		"auto_start":     autoStart,
		"billing_type":   billingType,
		"duration":       duration,
		"description":    description,
		"hostname":       hostname,
		"hypervisor":     hypervisor,
		"secgroup_id":    secgroupID,
		"secgroups":      secgroups,
		"keypair_id":     keypairID,
		"project_id":     projectID,
		"zone_id":        zoneID,
		"region_id":      regionID,
		"disable_delete": disableDelete,
		"boot_order":     bootOrder,
		"data_disks":     len(dataDisks),
	}).Info("开始创建虚拟机")

	createRequest := models.CreateServerRequest{
		Name:          name,
		VcpuCount:     vcpuCount,
		VmemSize:      vmemSize,
		ImageId:       imageID,
		DiskSize:      diskSize,
		NetworkId:     networkID,
		ServerskuId:   serverSkuID,
		Count:         count,
		Password:      password,
		AutoStart:     autoStart,
		BillingType:   billingType,
		Duration:      duration,
		Description:   description,
		Hostname:      hostname,
		Hypervisor:    hypervisor,
		Metadata:      metadata,
		SecgroupId:    secgroupID,
		Secgroups:     secgroups,
		UserData:      userData,
		KeypairId:     keypairID,
		ProjectId:     projectID,
		ZoneId:        zoneID,
		RegionId:      regionID,
		DisableDelete: disableDelete,
		BootOrder:     bootOrder,
		DataDisks:     dataDisks,
	}

	response, err := c.adapter.CreateServer(ctx, createRequest)
	if err != nil {
		c.logger.WithError(err).Error("创建虚拟机失败")
		return nil, fmt.Errorf("创建虚拟机失败: %w", err)
	}

	formattedResult := c.formatCreateResult(response, &createRequest)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerCreateTool) GetName() string {
	return "cloudpods_create_server"
}

func (c *CloudpodsServerCreateTool) formatCreateResult(response *models.CreateServerResponse, request *models.CreateServerRequest) map[string]interface{} {
	formatted := map[string]interface{}{
		"create_info": map[string]interface{}{
			"name":           request.Name,
			"vcpu_count":     request.VcpuCount,
			"vmem_size":      request.VmemSize,
			"image_id":       request.ImageId,
			"disk_size":      request.DiskSize,
			"network_id":     request.NetworkId,
			"serversku_id":   request.ServerskuId,
			"count":          request.Count,
			"auto_start":     request.AutoStart,
			"billing_type":   request.BillingType,
			"duration":       request.Duration,
			"description":    request.Description,
			"hostname":       request.Hostname,
			"hypervisor":     request.Hypervisor,
			"secgroup_id":    request.SecgroupId,
			"keypair_id":     request.KeypairId,
			"project_id":     request.ProjectId,
			"zone_id":        request.ZoneId,
			"region_id":      request.RegionId,
			"disable_delete": request.DisableDelete,
			"boot_order":     request.BootOrder,
		},
		"result": map[string]interface{}{
			"status":  response.Status,
			"message": response.Message,
			"servers": make([]map[string]interface{}, 0, len(response.Data.Servers)),
		},
	}

	for _, server := range response.Data.Servers {
		serverInfo := map[string]interface{}{
			"id":      server.ID,
			"name":    server.Name,
			"status":  server.Status,
			"task_id": server.TaskID,
		}
		formatted["result"].(map[string]interface{})["servers"] = append(
			formatted["result"].(map[string]interface{})["servers"].([]map[string]interface{}),
			serverInfo,
		)
	}

	formatted["summary"] = map[string]interface{}{
		"requested_count": request.Count,
		"created_count":   len(response.Data.Servers),
		"success":         response.Status == 200,
	}

	return formatted
}
