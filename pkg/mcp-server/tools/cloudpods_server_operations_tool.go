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

// CloudpodsServerStartTool 用于启动指定的Cloudpods虚拟机实例
type CloudpodsServerStartTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerStartTool 创建一个新的CloudpodsServerStartTool实例
func NewCloudpodsServerStartTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerStartTool {
	return &CloudpodsServerStartTool{
		adapter: adapter,
	}
}

// GetTool 返回启动虚拟机工具的定义，包括参数和描述
func (c *CloudpodsServerStartTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_start_server",
		mcp.WithDescription("启动指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("auto_prepaid", mcp.Description("按量机器自动转换为包年包月，默认为false")),
		mcp.WithString("qemu_version", mcp.Description("指定启动虚拟机的Qemu版本，可选值：2.12.1, 4.2.0，仅适用于KVM虚拟机")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理启动虚拟机的请求，调用适配器执行启动操作并返回结果
func (c *CloudpodsServerStartTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中获取必需的 server_id 参数
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 解析 auto_prepaid 参数，决定是否自动转换为包年包月
	autoPrepaid := false
	if autoPrepaidStr := req.GetString("auto_prepaid", "false"); autoPrepaidStr == "true" {
		autoPrepaid = true
	}

	// 获取 qemu_version 参数，用于指定启动虚拟机的 Qemu 版本
	qemuVersion := req.GetString("qemu_version", "")

	// 构造启动虚拟机的请求参数
	startReq := models.ServerStartRequest{
		AutoPrepaid: autoPrepaid,
		QemuVersion: qemuVersion,
	}

	// 获取认证所需的 access key 和 secret key
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器的 StartServer 方法执行启动操作
	response, err := c.adapter.StartServer(ctx, serverID, startReq, ak, sk)
	if err != nil {
		log.Errorf("Fail to start server: %s", err)
		return nil, fmt.Errorf("fail to start server: %w", err)
	}

	// 构造返回结果，包含任务ID、成功状态和状态信息
	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "start",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	// 如果有错误信息，则添加到结果中
	if response.Error != "" {
		result["error"] = response.Error
	}

	// 将结果序列化为 JSON 格式
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 返回序列化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回启动虚拟机工具的名称
func (c *CloudpodsServerStartTool) GetName() string {
	return "cloudpods_start_server"
}

// CloudpodsServerStopTool 用于停止指定的Cloudpods虚拟机实例
type CloudpodsServerStopTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerStopTool 创建一个新的CloudpodsServerStopTool实例
func NewCloudpodsServerStopTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerStopTool {
	return &CloudpodsServerStopTool{
		adapter: adapter,
	}
}

// GetTool 返回停止虚拟机工具的定义，包括参数和描述
func (c *CloudpodsServerStopTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_stop_server",
		mcp.WithDescription("停止指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("is_force", mcp.Description("是否强制停止，默认为false")),
		mcp.WithString("stop_charging", mcp.Description("是否关机停止计费，默认为false")),
		mcp.WithString("timeout_secs", mcp.Description("关机等待时间，如果是强制关机，则等待时间为0，如果不设置，默认为30秒")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理停止虚拟机的请求，调用适配器执行停止操作并返回结果
func (c *CloudpodsServerStopTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中获取必需的 server_id 参数
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 解析 is_force 参数，决定是否强制停止虚拟机
	isForce := false
	if isForceStr := req.GetString("is_force", "false"); isForceStr == "true" {
		isForce = true
	}

	// 解析 stop_charging 参数，决定是否停止计费
	stopCharging := false
	if stopChargingStr := req.GetString("stop_charging", "false"); stopChargingStr == "true" {
		stopCharging = true
	}

	// 解析 timeout_secs 参数，设置停止操作的超时时间
	timeoutSecs := int64(0)
	if timeoutSecsStr := req.GetString("timeout_secs", ""); timeoutSecsStr != "" {
		if parsed, err := strconv.ParseInt(timeoutSecsStr, 10, 64); err == nil && parsed > 0 {
			timeoutSecs = parsed
		}
	}

	// 构造停止虚拟机的请求参数
	stopReq := models.ServerStopRequest{
		IsForce:      isForce,
		StopCharging: stopCharging,
		TimeoutSecs:  timeoutSecs,
	}

	// 获取认证所需的 access key 和 secret key
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器的 StopServer 方法执行停止操作
	response, err := c.adapter.StopServer(ctx, serverID, stopReq, ak, sk)
	if err != nil {
		log.Errorf("Fail to stop server: %s", err)
		return nil, fmt.Errorf("fail to stop server: %w", err)
	}

	// 构造返回结果，包含任务ID、成功状态和状态信息
	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "stop",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	// 如果有错误信息，则添加到结果中
	if response.Error != "" {
		result["error"] = response.Error
	}

	// 将结果序列化为 JSON 格式
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 返回序列化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回停止虚拟机工具的名称
func (c *CloudpodsServerStopTool) GetName() string {
	return "cloudpods_stop_server"
}

// CloudpodsServerRestartTool 用于重启指定的Cloudpods虚拟机实例
type CloudpodsServerRestartTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerRestartTool 创建一个新的CloudpodsServerRestartTool实例
func NewCloudpodsServerRestartTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerRestartTool {
	return &CloudpodsServerRestartTool{
		adapter: adapter,
	}
}

// GetTool 返回重启虚拟机工具的定义，包括参数和描述
func (c *CloudpodsServerRestartTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_restart_server",
		mcp.WithDescription("重启指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("is_force", mcp.Description("是否强制重启，默认为false")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理重启虚拟机的请求，调用适配器执行重启操作并返回结果
func (c *CloudpodsServerRestartTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中获取必需的 server_id 参数
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 解析 is_force 参数，决定是否强制重启虚拟机
	isForce := false
	if isForceStr := req.GetString("is_force", "false"); isForceStr == "true" {
		isForce = true
	}

	// 构造重启虚拟机的请求参数
	restartReq := models.ServerRestartRequest{
		IsForce: isForce,
	}

	// 获取认证所需的 access key 和 secret key
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器的 RestartServer 方法执行重启操作
	response, err := c.adapter.RestartServer(ctx, serverID, restartReq, ak, sk)
	if err != nil {
		log.Errorf("Fail to query restart server: %s", err)
		return nil, fmt.Errorf("fail to restart server: %w", err)
	}

	// 构造返回结果，包含任务ID、成功状态和状态信息
	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "restart",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	// 如果有错误信息，则添加到结果中
	if response.Error != "" {
		result["error"] = response.Error
	}

	// 将结果序列化为 JSON 格式
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 返回序列化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回重启虚拟机工具的名称
func (c *CloudpodsServerRestartTool) GetName() string {
	return "cloudpods_restart_server"
}

// CloudpodsServerResetPasswordTool 用于重置指定Cloudpods虚拟机的登录密码
type CloudpodsServerResetPasswordTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerResetPasswordTool 创建一个新的CloudpodsServerResetPasswordTool实例
func NewCloudpodsServerResetPasswordTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerResetPasswordTool {
	return &CloudpodsServerResetPasswordTool{
		adapter: adapter,
	}
}

// GetTool 返回重置虚拟机密码工具的定义，包括参数和描述
func (c *CloudpodsServerResetPasswordTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_reset_server_password",
		mcp.WithDescription("重置指定Cloudpods虚拟机的登录密码"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("password", mcp.Required(), mcp.Description("新密码，长度8-30个字符")),
		mcp.WithString("reset_password", mcp.Description("是否重置密码，默认为true")),
		mcp.WithString("auto_start", mcp.Description("重置后是否自动启动，默认为true")),
		mcp.WithString("username", mcp.Description("用户名，可选，默认为空")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理重置虚拟机密码的请求，调用适配器执行密码重置操作并返回结果
func (c *CloudpodsServerResetPasswordTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中获取必需的 server_id 参数
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 从请求中获取必需的 password 参数，并验证其长度
	password, err := req.RequireString("password")
	if err != nil {
		return nil, err
	}

	if len(password) < 8 || len(password) > 30 {
		return nil, fmt.Errorf("密码长度必须在8-30个字符之间")
	}

	// 解析 reset_password 参数，决定是否重置密码
	resetPassword := true
	if resetPasswordStr := req.GetString("reset_password", "true"); resetPasswordStr == "false" {
		resetPassword = false
	}

	// 解析 auto_start 参数，决定重置密码后是否自动启动虚拟机
	autoStart := true
	if autoStartStr := req.GetString("auto_start", "true"); autoStartStr == "false" {
		autoStart = false
	}

	// 获取 username 参数，可选
	username := req.GetString("username", "")

	// 构造重置虚拟机密码的请求参数
	resetPasswordReq := models.ServerResetPasswordRequest{
		Password:      password,
		ResetPassword: resetPassword,
		AutoStart:     autoStart,
		Username:      username,
	}

	// 获取认证所需的 access key 和 secret key
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器的 ResetServerPassword 方法执行密码重置操作
	response, err := c.adapter.ResetServerPassword(ctx, serverID, resetPasswordReq, ak, sk)
	if err != nil {
		log.Errorf("Fail to reset server password: %s", err)
		return nil, fmt.Errorf("fail to reset server password: %w", err)
	}

	// 构造返回结果，包含任务ID、成功状态和状态信息
	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "reset-password",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	// 如果有错误信息，则添加到结果中
	if response.Error != "" {
		result["error"] = response.Error
	}

	// 将结果序列化为 JSON 格式
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 返回序列化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回重置虚拟机密码工具的名称
func (c *CloudpodsServerResetPasswordTool) GetName() string {
	return "cloudpods_reset_server_password"
}

// CloudpodsServerDeleteTool 用于删除指定的Cloudpods虚拟机实例
type CloudpodsServerDeleteTool struct {
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerDeleteTool 创建一个新的CloudpodsServerDeleteTool实例
func NewCloudpodsServerDeleteTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerDeleteTool {
	return &CloudpodsServerDeleteTool{
		adapter: adapter,
	}
}

// GetTool 返回删除虚拟机工具的定义，包括参数和描述
func (c *CloudpodsServerDeleteTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_delete_server",
		mcp.WithDescription("删除指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("override_pending_delete", mcp.Description("是否强制删除（包括在回收站中的实例），默认为false")),
		mcp.WithString("purge", mcp.Description("是否仅删除本地资源，默认为false")),
		mcp.WithString("delete_snapshots", mcp.Description("是否删除快照，默认为false")),
		mcp.WithString("delete_eip", mcp.Description("是否删除关联的EIP，默认为false")),
		mcp.WithString("delete_disks", mcp.Description("是否删除关联的数据盘，默认为false")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理删除虚拟机的请求，调用适配器执行删除操作并返回结果
func (c *CloudpodsServerDeleteTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中获取必需的 server_id 参数
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 解析 override_pending_delete 参数，决定是否强制删除（包括在回收站中的实例）
	overridePendingDelete := false
	if overrideStr := req.GetString("override_pending_delete", "false"); overrideStr == "true" {
		overridePendingDelete = true
	}

	// 解析 purge 参数，决定是否仅删除本地资源
	purge := false
	if purgeStr := req.GetString("purge", "false"); purgeStr == "true" {
		purge = true
	}

	// 解析 delete_snapshots 参数，决定是否删除快照
	deleteSnapshots := false
	if deleteSnapshotsStr := req.GetString("delete_snapshots", "false"); deleteSnapshotsStr == "true" {
		deleteSnapshots = true
	}

	// 解析 delete_eip 参数，决定是否删除关联的EIP
	deleteEip := false
	if deleteEipStr := req.GetString("delete_eip", "false"); deleteEipStr == "true" {
		deleteEip = true
	}

	// 解析 delete_disks 参数，决定是否删除关联的数据盘
	deleteDisks := false
	if deleteDisksStr := req.GetString("delete_disks", "false"); deleteDisksStr == "true" {
		deleteDisks = true
	}

	// 构造删除虚拟机的请求参数
	deleteReq := models.ServerDeleteRequest{
		OverridePendingDelete: overridePendingDelete,
		Purge:                 purge,
		DeleteSnapshots:       deleteSnapshots,
		DeleteEip:             deleteEip,
		DeleteDisks:           deleteDisks,
	}

	// 获取认证所需的 access key 和 secret key
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器的 DeleteServer 方法执行删除操作
	response, err := c.adapter.DeleteServer(ctx, serverID, deleteReq, ak, sk)
	if err != nil {
		log.Errorf("Fail to delete server: %s", err)
		return nil, fmt.Errorf("fail to delete server: %w", err)
	}

	// 构造返回结果，包含任务ID、成功状态和状态信息
	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "delete",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	// 如果有错误信息，则添加到结果中
	if response.Error != "" {
		result["error"] = response.Error
	}

	// 将结果序列化为 JSON 格式
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 返回序列化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回删除虚拟机工具的名称
func (c *CloudpodsServerDeleteTool) GetName() string {
	return "cloudpods_delete_server"
}
