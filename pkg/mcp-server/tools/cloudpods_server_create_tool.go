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

// CloudpodsServerCreateTool 用于创建Cloudpods虚拟机实例
type CloudpodsServerCreateTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerCreateTool 创建一个新的CloudpodsServerCreateTool实例
// adapter: 用于与Cloudpods API交互的适配器
// 返回值: CloudpodsServerCreateTool实例指针
func NewCloudpodsServerCreateTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerCreateTool {
	return &CloudpodsServerCreateTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回创建虚拟机工具的元数据
// 该工具用于创建Cloudpods虚拟机实例，支持指定各种配置参数
// name: 虚拟机名称 (必填)
// vcpu_count: CPU核心数 (必填)
// vmem_size: 内存大小(MB) (必填)
// image_id: 镜像ID (必填)
// disk_size: 系统盘大小(GB)，不指定则使用镜像默认大小
// network_id: 网络ID (必填)
// serversku_id: 套餐ID，指定后将忽略vcpu_count和vmem_size参数
// password: 虚拟机密码，长度8-30个字符
// count: 创建数量，默认为1
// auto_start: 是否自动启动，默认为true
// billing_type: 计费类型，例如：postpaid、prepaid
// duration: 包年包月时长，例如：1M、1Y
// description: 描述信息
// hostname: 主机名
// hypervisor: 虚拟化技术，如kvm, esxi等，默认为kvm
// metadata: 标签列表，格式为JSON字符串，例如：{"key1":"value1","key2":"value2"}
// secgroup_id: 安全组ID
// secgroups: 安全组ID列表，多个ID用逗号分隔
// user_data: 用户自定义启动脚本
// keypair_id: 秘钥对ID
// project_id: 项目ID
// zone_id: 可用区ID
// region_id: 区域ID
// disable_delete: 是否开启删除保护，默认为true
// boot_order: 启动顺序，如cdn
// data_disks: 数据盘配置，格式为JSON字符串数组，例如：[{"size":100,"disk_type":"data"}]
// ak: 用户登录cloudpods后获取的access key
// sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsServerCreateTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_create_server",
		mcp.WithDescription("创建Cloudpods虚拟机实例"),
		mcp.WithString("name", mcp.Required(), mcp.Description("虚拟机名称")),
		mcp.WithString("vcpu_count", mcp.Required(), mcp.Description("CPU核心数")),
		mcp.WithString("vmem_size", mcp.Required(), mcp.Description("内存大小(MB)")),
		mcp.WithString("image_id", mcp.Required(), mcp.Description("镜像ID")),
		mcp.WithString("disk_size", mcp.Description("系统盘大小(GB)，不指定则使用镜像默认大小")),
		mcp.WithString("network_id", mcp.Required(), mcp.Description("网络ID")),
		mcp.WithString("serversku_id", mcp.Description("套餐ID，指定后将忽略vcpu_count和vmem_size参数")),
		mcp.WithString("password", mcp.Description("虚拟机密码，长度8-30个字符")),
		mcp.WithString("count", mcp.Description("创建数量，默认为1")),
		mcp.WithString("auto_start", mcp.Description("是否自动启动，默认为true")),
		mcp.WithString("billing_type", mcp.Description("计费类型，例如：postpaid、prepaid")),
		mcp.WithString("duration", mcp.Description("包年包月时长，例如：1M、1Y")),
		mcp.WithString("description", mcp.Description("描述信息")),
		mcp.WithString("hostname", mcp.Description("主机名")),
		mcp.WithString("hypervisor", mcp.Description("虚拟化技术，如kvm, esxi等，默认为kvm")),
		mcp.WithString("metadata", mcp.Description("标签列表，格式为JSON字符串，例如：{\"key1\":\"value1\",\"key2\":\"value2\"}")),
		mcp.WithString("secgroup_id", mcp.Description("安全组ID")),
		mcp.WithString("secgroups", mcp.Description("安全组ID列表，多个ID用逗号分隔")),
		mcp.WithString("user_data", mcp.Description("用户自定义启动脚本")),
		mcp.WithString("keypair_id", mcp.Description("秘钥对ID")),
		mcp.WithString("project_id", mcp.Description("项目ID")),
		mcp.WithString("zone_id", mcp.Description("可用区ID")),
		mcp.WithString("region_id", mcp.Description("区域ID")),
		mcp.WithString("disable_delete", mcp.Description("是否开启删除保护，默认为true")),
		mcp.WithString("boot_order", mcp.Description("启动顺序，如cdn")),
		mcp.WithString("data_disks", mcp.Description("数据盘配置，格式为JSON字符串数组，例如：[{\"size\":100,\"disk_type\":\"data\"}]")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理创建虚拟机的请求
// ctx: 上下文，用于控制请求的生命周期
// req: 包含创建虚拟机所需参数的请求对象
// 返回值: 包含创建结果的工具结果对象或错误信息
func (c *CloudpodsServerCreateTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取必填参数：虚拟机名称
	name, err := req.RequireString("name")
	if err != nil {
		return nil, err
	}

	// 获取必填参数：镜像ID
	imageID, err := req.RequireString("image_id")
	if err != nil {
		return nil, err
	}

	// 获取必填参数：网络ID
	networkID, err := req.RequireString("network_id")
	if err != nil {
		return nil, err
	}

	// 获取必填参数：CPU核心数并转换为整数
	vcpuCountStr, err := req.RequireString("vcpu_count")
	if err != nil {
		return nil, err
	}
	vcpuCount, err := strconv.ParseInt(vcpuCountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的CPU核心数: %s", vcpuCountStr)
	}

	// 获取必填参数：内存大小并转换为整数
	vmemSizeStr, err := req.RequireString("vmem_size")
	if err != nil {
		return nil, err
	}
	vmemSize, err := strconv.ParseInt(vmemSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的内存大小: %s", vmemSizeStr)
	}

	// 获取可选参数：套餐ID
	serverSkuID := req.GetString("serversku_id", "")

	// 获取可选参数：磁盘大小，如果指定则转换为整数
	diskSize := int64(0)
	if diskSizeStr := req.GetString("disk_size", ""); diskSizeStr != "" {
		if parsedSize, err := strconv.ParseInt(diskSizeStr, 10, 64); err == nil && parsedSize > 0 {
			diskSize = parsedSize
		}
	}

	// 获取可选参数：虚拟机密码，并验证长度
	password := req.GetString("password", "")
	if password != "" && (len(password) < 8 || len(password) > 30) {
		return nil, fmt.Errorf("密码长度必须在8-30个字符之间")
	}

	// 获取可选参数：创建数量，默认为1
	count := 1
	if countStr := req.GetString("count", "1"); countStr != "1" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	// 获取可选参数：是否自动启动，默认为true
	autoStart := true
	if autoStartStr := req.GetString("auto_start", "true"); autoStartStr == "false" {
		autoStart = false
	}

	// 获取可选参数：是否开启删除保护，默认为true
	disableDelete := true
	if disableDeleteStr := req.GetString("disable_delete", "true"); disableDeleteStr == "false" {
		disableDelete = false
	}

	// 获取其他可选参数
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

	// 获取安全组ID列表，并按逗号分割
	var secgroups []string
	if secgroupsStr := req.GetString("secgroups", ""); secgroupsStr != "" {
		secgroups = strings.Split(secgroupsStr, ",")
	}

	// 解析元数据JSON字符串
	metadata := make(map[string]string)
	if metadataStr := req.GetString("metadata", ""); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return nil, fmt.Errorf("无效的元数据JSON格式: %w", err)
		}
	}

	// 解析数据盘配置JSON数组
	var dataDisks []models.DiskConfig
	if dataDisksStr := req.GetString("data_disks", ""); dataDisksStr != "" {
		if err := json.Unmarshal([]byte(dataDisksStr), &dataDisks); err != nil {
			return nil, fmt.Errorf("无效的数据盘配置JSON格式: %w", err)
		}
	}

	// 构造创建虚拟机的请求对象
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

	// 获取访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器创建虚拟机
	response, err := c.adapter.CreateServer(ctx, createRequest, ak, sk)
	if err != nil {
		log.Errorf("Fail to create server: %s", err)
		return nil, fmt.Errorf("fail to create server: %w", err)
	}

	// 格式化创建结果
	formattedResult := c.formatCreateResult(response, &createRequest)

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
// 返回值: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsServerCreateTool) GetName() string {
	return "cloudpods_create_server"
}

// formatCreateResult 格式化创建虚拟机的响应结果
// response: 原始的创建虚拟机响应数据
// request: 原始的创建虚拟机请求数据
// 返回值: 格式化后的结果，包含创建信息、结果详情和摘要
func (c *CloudpodsServerCreateTool) formatCreateResult(response *models.CreateServerResponse, request *models.CreateServerRequest) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		// 创建请求的基本信息
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
		// 创建响应的结果信息
		"result": map[string]interface{}{
			"status":  response.Status,
			"message": response.Message,
			"servers": make([]map[string]interface{}, 0, len(response.Data.Servers)),
		},
	}

	// 遍历创建的虚拟机列表，构造每个虚拟机的详细信息
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

	// 构造摘要信息
	formatted["summary"] = map[string]interface{}{
		"requested_count": request.Count,              // 请求创建的虚拟机数量
		"created_count":   len(response.Data.Servers), // 实际创建的虚拟机数量
		"success":         response.Status == 200,     // 创建是否成功
	}

	return formatted
}
