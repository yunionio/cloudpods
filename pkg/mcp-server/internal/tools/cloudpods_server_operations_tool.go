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
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/models"
)

type CloudpodsServerStartTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerStartTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerStartTool {
	return &CloudpodsServerStartTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerStartTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_start_server",
		mcp.WithDescription("启动指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("auto_prepaid",
			mcp.Description("按量机器自动转换为包年包月，默认为false")),
		mcp.WithString("qemu_version",
			mcp.Description("指定启动虚拟机的Qemu版本，可选值：2.12.1, 4.2.0，仅适用于KVM虚拟机")),
	)
}

func (c *CloudpodsServerStartTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	autoPrepaid := false
	if autoPrepaidStr := req.GetString("auto_prepaid", "false"); autoPrepaidStr == "true" {
		autoPrepaid = true
	}

	qemuVersion := req.GetString("qemu_version", "")

	c.logger.WithFields(logrus.Fields{
		"server_id":    serverID,
		"auto_prepaid": autoPrepaid,
		"qemu_version": qemuVersion,
	}).Info("开始启动虚拟机")

	startReq := models.ServerStartRequest{
		AutoPrepaid: autoPrepaid,
		QemuVersion: qemuVersion,
	}

	response, err := c.adapter.StartServer(ctx, serverID, startReq)
	if err != nil {
		c.logger.WithError(err).Error("启动虚拟机失败")
		return nil, fmt.Errorf("启动虚拟机失败: %w", err)
	}

	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "start",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	if response.Error != "" {
		result["error"] = response.Error
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerStartTool) GetName() string {
	return "cloudpods_start_server"
}

type CloudpodsServerStopTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerStopTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerStopTool {
	return &CloudpodsServerStopTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerStopTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_stop_server",
		mcp.WithDescription("停止指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("is_force",
			mcp.Description("是否强制停止，默认为false")),
		mcp.WithString("stop_charging",
			mcp.Description("是否关机停止计费，默认为false")),
		mcp.WithString("timeout_secs",
			mcp.Description("关机等待时间，如果是强制关机，则等待时间为0，如果不设置，默认为30秒")),
	)
}

func (c *CloudpodsServerStopTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	isForce := false
	if isForceStr := req.GetString("is_force", "false"); isForceStr == "true" {
		isForce = true
	}

	stopCharging := false
	if stopChargingStr := req.GetString("stop_charging", "false"); stopChargingStr == "true" {
		stopCharging = true
	}

	timeoutSecs := int64(0)
	if timeoutSecsStr := req.GetString("timeout_secs", ""); timeoutSecsStr != "" {
		if parsed, err := strconv.ParseInt(timeoutSecsStr, 10, 64); err == nil && parsed > 0 {
			timeoutSecs = parsed
		}
	}

	c.logger.WithFields(logrus.Fields{
		"server_id":     serverID,
		"is_force":      isForce,
		"stop_charging": stopCharging,
		"timeout_secs":  timeoutSecs,
	}).Info("开始停止虚拟机")

	stopReq := models.ServerStopRequest{
		IsForce:      isForce,
		StopCharging: stopCharging,
		TimeoutSecs:  timeoutSecs,
	}

	response, err := c.adapter.StopServer(ctx, serverID, stopReq)
	if err != nil {
		c.logger.WithError(err).Error("停止虚拟机失败")
		return nil, fmt.Errorf("停止虚拟机失败: %w", err)
	}

	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "stop",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	if response.Error != "" {
		result["error"] = response.Error
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerStopTool) GetName() string {
	return "cloudpods_stop_server"
}

type CloudpodsServerRestartTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerRestartTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerRestartTool {
	return &CloudpodsServerRestartTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerRestartTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_restart_server",
		mcp.WithDescription("重启指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("is_force",
			mcp.Description("是否强制重启，默认为false")),
	)
}

func (c *CloudpodsServerRestartTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	isForce := false
	if isForceStr := req.GetString("is_force", "false"); isForceStr == "true" {
		isForce = true
	}

	c.logger.WithFields(logrus.Fields{
		"server_id": serverID,
		"is_force":  isForce,
	}).Info("开始重启虚拟机")

	restartReq := models.ServerRestartRequest{
		IsForce: isForce,
	}

	response, err := c.adapter.RestartServer(ctx, serverID, restartReq)
	if err != nil {
		c.logger.WithError(err).Error("重启虚拟机失败")
		return nil, fmt.Errorf("重启虚拟机失败: %w", err)
	}

	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "restart",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	if response.Error != "" {
		result["error"] = response.Error
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerRestartTool) GetName() string {
	return "cloudpods_restart_server"
}

type CloudpodsServerResetPasswordTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerResetPasswordTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerResetPasswordTool {
	return &CloudpodsServerResetPasswordTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerResetPasswordTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_reset_server_password",
		mcp.WithDescription("重置指定Cloudpods虚拟机的登录密码"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("password",
			mcp.Required(),
			mcp.Description("新密码，长度8-30个字符")),
		mcp.WithString("reset_password",
			mcp.Description("是否重置密码，默认为true")),
		mcp.WithString("auto_start",
			mcp.Description("重置后是否自动启动，默认为true")),
		mcp.WithString("username",
			mcp.Description("用户名，可选，默认为空")),
	)
}

func (c *CloudpodsServerResetPasswordTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	password, err := req.RequireString("password")
	if err != nil {
		return nil, err
	}

	if len(password) < 8 || len(password) > 30 {
		return nil, fmt.Errorf("密码长度必须在8-30个字符之间")
	}

	resetPassword := true
	if resetPasswordStr := req.GetString("reset_password", "true"); resetPasswordStr == "false" {
		resetPassword = false
	}

	autoStart := true
	if autoStartStr := req.GetString("auto_start", "true"); autoStartStr == "false" {
		autoStart = false
	}

	username := req.GetString("username", "")

	c.logger.WithFields(logrus.Fields{
		"server_id":      serverID,
		"reset_password": resetPassword,
		"auto_start":     autoStart,
		"username":       username,
	}).Info("开始重置虚拟机密码")

	resetPasswordReq := models.ServerResetPasswordRequest{
		Password:      password,
		ResetPassword: resetPassword,
		AutoStart:     autoStart,
		Username:      username,
	}

	response, err := c.adapter.ResetServerPassword(ctx, serverID, resetPasswordReq)
	if err != nil {
		c.logger.WithError(err).Error("重置虚拟机密码失败")
		return nil, fmt.Errorf("重置虚拟机密码失败: %w", err)
	}

	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "reset-password",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	if response.Error != "" {
		result["error"] = response.Error
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerResetPasswordTool) GetName() string {
	return "cloudpods_reset_server_password"
}

type CloudpodsServerDeleteTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerDeleteTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerDeleteTool {
	return &CloudpodsServerDeleteTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerDeleteTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_delete_server",
		mcp.WithDescription("删除指定的Cloudpods虚拟机实例"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("override_pending_delete",
			mcp.Description("是否强制删除（包括在回收站中的实例），默认为false")),
		mcp.WithString("purge",
			mcp.Description("是否仅删除本地资源，默认为false")),
		mcp.WithString("delete_snapshots",
			mcp.Description("是否删除快照，默认为false")),
		mcp.WithString("delete_eip",
			mcp.Description("是否删除关联的EIP，默认为false")),
		mcp.WithString("delete_disks",
			mcp.Description("是否删除关联的数据盘，默认为false")),
	)
}

func (c *CloudpodsServerDeleteTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	overridePendingDelete := false
	if overrideStr := req.GetString("override_pending_delete", "false"); overrideStr == "true" {
		overridePendingDelete = true
	}

	purge := false
	if purgeStr := req.GetString("purge", "false"); purgeStr == "true" {
		purge = true
	}

	deleteSnapshots := false
	if deleteSnapshotsStr := req.GetString("delete_snapshots", "false"); deleteSnapshotsStr == "true" {
		deleteSnapshots = true
	}

	deleteEip := false
	if deleteEipStr := req.GetString("delete_eip", "false"); deleteEipStr == "true" {
		deleteEip = true
	}

	deleteDisks := false
	if deleteDisksStr := req.GetString("delete_disks", "false"); deleteDisksStr == "true" {
		deleteDisks = true
	}

	c.logger.WithFields(logrus.Fields{
		"server_id":               serverID,
		"override_pending_delete": overridePendingDelete,
		"purge":                   purge,
		"delete_snapshots":        deleteSnapshots,
		"delete_eip":              deleteEip,
		"delete_disks":            deleteDisks,
	}).Info("开始删除虚拟机")

	deleteReq := models.ServerDeleteRequest{
		OverridePendingDelete: overridePendingDelete,
		Purge:                 purge,
		DeleteSnapshots:       deleteSnapshots,
		DeleteEip:             deleteEip,
		DeleteDisks:           deleteDisks,
	}

	response, err := c.adapter.DeleteServer(ctx, serverID, deleteReq)
	if err != nil {
		c.logger.WithError(err).Error("删除虚拟机失败")
		return nil, fmt.Errorf("删除虚拟机失败: %w", err)
	}

	result := map[string]interface{}{
		"server_id": serverID,
		"operation": "delete",
		"task_id":   response.TaskId,
		"success":   response.Success,
		"status":    response.Status,
	}

	if response.Error != "" {
		result["error"] = response.Error
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerDeleteTool) GetName() string {
	return "cloudpods_delete_server"
}
