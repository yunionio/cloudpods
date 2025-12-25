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

package adapters

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// StartServer 启动 Cloudpods 中的服务器
func (a *CloudpodsAdapter) StartServer(ctx context.Context, serverId string, req models.ServerStartRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造启动参数
	params := jsonutils.NewDict()

	// 如果需要自动续费预付费实例，则设置相应参数
	if req.AutoPrepaid {
		params.Set("auto_prepaid", jsonutils.NewBool(true))
	}

	// 如果指定了 QEMU 版本，则设置相应参数
	if req.QemuVersion != "" {
		params.Set("qemu_version", jsonutils.NewString(req.QemuVersion))
	}

	// 调用 Cloudpods API 启动服务器
	result, err := compute.Servers.PerformAction(session, serverId, "start", params)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	// 构造响应数据
	response := &models.ServerOperationResponse{
		Operation: "start",
	}

	// 尝试将结果解析到响应结构体中
	if err := result.Unmarshal(response); err != nil {
		// 如果解析失败，则尝试获取任务 ID
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		// 如果任务 ID 不为空，则认为操作成功
		response.Success = taskId != ""
	}

	return response, nil
}

// StopServer 停止 Cloudpods 中的服务器
func (a *CloudpodsAdapter) StopServer(ctx context.Context, serverId string, req models.ServerStopRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造停止参数
	params := jsonutils.NewDict()

	// 如果需要强制停止，则设置相应参数
	if req.IsForce {
		params.Set("is_force", jsonutils.NewBool(true))
	}

	// 如果需要停止计费，则设置相应参数
	if req.StopCharging {
		params.Set("stop_charging", jsonutils.NewBool(true))
	}

	// 如果设置了超时时间，则设置相应参数
	if req.TimeoutSecs > 0 {
		params.Set("timeout_secs", jsonutils.NewInt(req.TimeoutSecs))
	}

	// 调用 Cloudpods API 停止服务器
	result, err := compute.Servers.PerformAction(session, serverId, "stop", params)
	if err != nil {
		return nil, fmt.Errorf("failed to stop server: %w", err)
	}

	// 构造响应数据
	response := &models.ServerOperationResponse{
		Operation: "stop",
	}

	// 尝试将结果解析到响应结构体中
	if err := result.Unmarshal(response); err != nil {
		// 如果解析失败，则尝试获取任务 ID
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		// 如果任务 ID 不为空，则认为操作成功
		response.Success = taskId != ""
	}

	return response, nil
}

// RestartServer 重启 Cloudpods 中的服务器
func (a *CloudpodsAdapter) RestartServer(ctx context.Context, serverId string, req models.ServerRestartRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造重启参数
	params := jsonutils.NewDict()

	// 如果需要强制重启，则设置相应参数
	if req.IsForce {
		params.Set("is_force", jsonutils.NewBool(true))
	}

	// 调用 Cloudpods API 重启服务器
	result, err := compute.Servers.PerformAction(session, serverId, "restart", params)
	if err != nil {
		return nil, fmt.Errorf("failed to restart server: %w", err)
	}

	// 构造响应数据
	response := &models.ServerOperationResponse{
		Operation: "restart",
	}

	// 尝试将结果解析到响应结构体中
	if err := result.Unmarshal(response); err != nil {
		// 如果解析失败，则尝试获取任务 ID
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		// 如果任务 ID 不为空，则认为操作成功
		response.Success = taskId != ""
	}

	return response, nil
}

// ResetServerPassword 重置 Cloudpods 中服务器的密码
func (a *CloudpodsAdapter) ResetServerPassword(ctx context.Context, serverId string, req models.ServerResetPasswordRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造密码重置参数
	params := jsonutils.NewDict()
	// 设置新密码
	params.Set("password", jsonutils.NewString(req.Password))

	if req.ResetPassword {
		params.Set("reset_password", jsonutils.NewBool(true))
	}

	if req.AutoStart {
		params.Set("auto_start", jsonutils.NewBool(true))
	}

	if req.Username != "" {
		params.Set("username", jsonutils.NewString(req.Username))
	}

	// 调用 Cloudpods API 重置服务器密码
	result, err := compute.Servers.PerformAction(session, serverId, "reset-password", params)
	if err != nil {
		return nil, fmt.Errorf("failed to reset server password: %w", err)
	}

	// 构造响应数据
	response := &models.ServerOperationResponse{
		Operation: "reset-password",
	}

	// 尝试将结果解析到响应结构体中
	if err := result.Unmarshal(response); err != nil {
		// 如果解析失败，则尝试获取任务 ID
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		// 如果任务 ID 不为空，则认为操作成功
		response.Success = taskId != ""
	}

	return response, nil
}

// DeleteServer 删除 Cloudpods 中的服务器
func (a *CloudpodsAdapter) DeleteServer(ctx context.Context, serverId string, req models.ServerDeleteRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造删除参数
	params := jsonutils.NewDict()
	// 如果需要覆盖待删除状态，则设置相应参数
	if req.OverridePendingDelete {
		params.Set("override_pending_delete", jsonutils.NewBool(true))
	}
	// 如果需要彻底删除，则设置相应参数
	if req.Purge {
		params.Set("purge", jsonutils.NewBool(true))
	}
	// 如果需要删除快照，则设置相应参数
	if req.DeleteSnapshots {
		params.Set("delete_snapshots", jsonutils.NewBool(true))
	}
	// 如果需要删除弹性 IP，则设置相应参数
	if req.DeleteEip {
		params.Set("delete_eip", jsonutils.NewBool(true))
	}
	// 如果需要删除磁盘，则设置相应参数
	if req.DeleteDisks {
		params.Set("delete_disks", jsonutils.NewBool(true))
	}

	// 调用 Cloudpods API 删除服务器
	result, err := compute.Servers.Delete(session, serverId, params)
	if err != nil {
		return nil, fmt.Errorf("failed to delete server: %w", err)
	}

	// 构造响应数据
	response := &models.ServerOperationResponse{
		Operation: "delete",
	}

	// 尝试将结果解析到响应结构体中
	if err := result.Unmarshal(response); err != nil {
		// 如果解析失败，则尝试获取任务 ID
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		// 如果任务 ID 不为空，则认为操作成功
		response.Success = taskId != ""
	}

	return response, nil
}

// CreateServer 在 Cloudpods 中创建服务器
func (a *CloudpodsAdapter) CreateServer(ctx context.Context, req models.CreateServerRequest, ak string, sk string) (*models.CreateServerResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造创建服务器的参数
	params := jsonutils.NewDict()
	// 设置服务器名称
	params.Set("name", jsonutils.NewString(req.Name))
	// 设置 CPU 核心数
	params.Set("vcpu_count", jsonutils.NewInt(req.VcpuCount))
	// 设置内存大小
	params.Set("vmem_size", jsonutils.NewInt(req.VmemSize))

	// 如果创建数量大于1，则设置相应参数
	if req.Count > 1 {
		params.Set("count", jsonutils.NewInt(int64(req.Count)))
	}

	// 如果需要自动启动，则设置相应参数
	if req.AutoStart {
		params.Set("auto_start", jsonutils.NewBool(req.AutoStart))
	}

	// 如果设置了密码，则设置相应参数
	if req.Password != "" {
		params.Set("password", jsonutils.NewString(req.Password))
	}

	// 如果设置了计费类型，则设置相应参数
	if req.BillingType != "" {
		params.Set("billing_type", jsonutils.NewString(req.BillingType))
	}

	// 如果设置了计费时长，则设置相应参数
	if req.Duration != "" {
		params.Set("duration", jsonutils.NewString(req.Duration))
	}

	// 如果设置了描述，则设置相应参数
	if req.Description != "" {
		params.Set("description", jsonutils.NewString(req.Description))
	}

	// 如果设置了主机名，则设置相应参数
	if req.Hostname != "" {
		params.Set("hostname", jsonutils.NewString(req.Hostname))
	}

	// 如果设置了虚拟化类型，则设置相应参数
	if req.Hypervisor != "" {
		params.Set("hypervisor", jsonutils.NewString(req.Hypervisor))
	}

	// 如果设置了用户数据，则设置相应参数
	if req.UserData != "" {
		params.Set("user_data", jsonutils.NewString(req.UserData))
	}

	// 如果设置了密钥对 ID，则设置相应参数
	if req.KeypairId != "" {
		params.Set("keypair_id", jsonutils.NewString(req.KeypairId))
	}

	// 如果设置了项目 ID，则设置相应参数
	if req.ProjectId != "" {
		params.Set("project_id", jsonutils.NewString(req.ProjectId))
	}

	// 如果设置了可用区 ID，则设置相应参数
	if req.ZoneId != "" {
		params.Set("prefer_zone_id", jsonutils.NewString(req.ZoneId))
	}

	// 如果设置了区域 ID，则设置相应参数
	if req.RegionId != "" {
		params.Set("prefer_region_id", jsonutils.NewString(req.RegionId))
	}

	// 如果需要禁用删除，则设置相应参数
	if req.DisableDelete {
		params.Set("disable_delete", jsonutils.NewBool(req.DisableDelete))
	}

	// 如果设置了启动顺序，则设置相应参数
	if req.BootOrder != "" {
		params.Set("boot_order", jsonutils.NewString(req.BootOrder))
	}

	// 如果设置了元数据，则设置相应参数
	if len(req.Metadata) > 0 {
		metaDict := jsonutils.NewDict()
		for k, v := range req.Metadata {
			metaDict.Set(k, jsonutils.NewString(v))
		}
		params.Set("__meta__", metaDict)
	}

	// 构造磁盘参数
	disks := jsonutils.NewArray()

	// 如果设置了镜像 ID，则构造系统磁盘参数
	if req.ImageId != "" {
		diskDict := jsonutils.NewDict()
		diskDict.Set("image_id", jsonutils.NewString(req.ImageId))
		diskDict.Set("disk_type", jsonutils.NewString("sys"))
		if req.DiskSize > 0 {
			diskDict.Set("size", jsonutils.NewInt(req.DiskSize))
		}
		disks.Add(diskDict)
	}

	// 构造数据磁盘参数
	for _, disk := range req.DataDisks {
		diskDict := jsonutils.NewDict()
		if disk.ImageId != "" {
			diskDict.Set("image_id", jsonutils.NewString(disk.ImageId))
		}
		if disk.Size > 0 {
			diskDict.Set("size", jsonutils.NewInt(disk.Size))
		}
		diskDict.Set("disk_type", jsonutils.NewString(disk.DiskType))
		disks.Add(diskDict)
	}

	// 如果有磁盘参数，则设置相应参数
	if disks.Length() > 0 {
		params.Set("disks", disks)
	}

	// 如果设置了网络 ID，则构造网络参数
	if req.NetworkId != "" {
		networks := jsonutils.NewArray()
		netDict := jsonutils.NewDict()
		netDict.Set("network", jsonutils.NewString(req.NetworkId))
		networks.Add(netDict)
		params.Set("nets", networks)
	}

	// 如果设置了安全组 ID，则设置相应参数
	if req.SecgroupId != "" {
		params.Set("secgrp_id", jsonutils.NewString(req.SecgroupId))
	}

	// 如果设置了安全组列表，则设置相应参数
	if len(req.Secgroups) > 0 {
		secgroups := jsonutils.NewArray()
		for _, sg := range req.Secgroups {
			secgroups.Add(jsonutils.NewString(sg))
		}
		params.Set("secgroups", secgroups)
	}

	// 如果设置了服务器规格 ID，则设置相应参数
	if req.ServerskuId != "" {
		params.Set("instance_type", jsonutils.NewString(req.ServerskuId))
	}

	// 调用 Cloudpods API 创建服务器
	result, err := compute.Servers.Create(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// 构造响应数据
	response := &models.CreateServerResponse{}
	if err := result.Unmarshal(response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create server response: %w", err)
	}

	return response, nil
}

// GetServerMonitor 获取 Cloudpods 中服务器的监控数据
func (a *CloudpodsAdapter) GetServerMonitor(ctx context.Context, serverId string, startTime, endTime int64, metrics []string, ak string, sk string) (*models.MonitorResponse, error) {
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()

	metricQuery := jsonutils.NewArray()

	for _, metric := range metrics {

		modelDict := jsonutils.NewDict()

		modelDict.Set("database", jsonutils.NewString("telegraf"))
		modelDict.Set("measurement", jsonutils.NewString("vm_cpu"))

		switch metric {
		case "cpu_usage":
			modelDict.Set("measurement", jsonutils.NewString("vm_cpu"))
		case "mem_usage":
			modelDict.Set("measurement", jsonutils.NewString("vm_mem"))
		case "disk_usage":
			modelDict.Set("measurement", jsonutils.NewString("vm_disk"))
		case "net_bps_rx", "net_bps_tx":
			modelDict.Set("measurement", jsonutils.NewString("vm_netio"))
		}

		tagsArray := jsonutils.NewArray()
		tagDict := jsonutils.NewDict()
		tagDict.Set("key", jsonutils.NewString("vm_id"))
		tagDict.Set("operator", jsonutils.NewString("="))
		tagDict.Set("value", jsonutils.NewString(serverId))
		tagsArray.Add(tagDict)
		modelDict.Set("tags", tagsArray)

		queryDict := jsonutils.NewDict()
		queryDict.Set("model", modelDict)

		if startTime > 0 {
			queryDict.Set("from", jsonutils.NewString(fmt.Sprintf("%d", startTime)))
		}
		if endTime > 0 {
			queryDict.Set("to", jsonutils.NewString(fmt.Sprintf("%d", endTime)))
		}

		metricQuery.Add(queryDict)
	}

	params.Set("metric_query", metricQuery)
	params.Set("scope", jsonutils.NewString("system"))

	params.Set("interval", jsonutils.NewString("60s"))

	result, err := monitor.UnifiedMonitorManager.PerformAction(session, "query", "", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get server monitor data: %w", err)
	}

	response := &models.MonitorResponse{
		Status: 200,
		Data: models.MonitorResponseData{
			Metrics: []models.MetricData{},
		},
	}

	unifiedmonitor, err := result.Get("unifiedmonitor")
	if err != nil {
		return nil, fmt.Errorf("failed to get unifiedmonitor data: %w", err)
	}

	series, err := unifiedmonitor.Get("Series")
	if err != nil {
		return nil, fmt.Errorf("failed to get series data: %w", err)
	}

	seriesArray, ok := series.(*jsonutils.JSONArray)
	if !ok {
		return nil, fmt.Errorf("invalid series data format")
	}

	for i := 0; i < seriesArray.Length(); i++ {
		seriesObj, err := seriesArray.GetAt(i)
		if err != nil {
			continue
		}

		name, _ := seriesObj.GetString("name")

		metricData := models.MetricData{
			Metric: name,
			Unit:   "%",
			Values: []models.MetricValue{},
		}

		if strings.Contains(name, "net_bps") {
			metricData.Unit = "bps"
		} else if strings.Contains(name, "disk_io") {
			metricData.Unit = "iops"
		}

		points, err := seriesObj.Get("points")
		if err != nil {
			continue
		}

		pointsArray, ok := points.(*jsonutils.JSONArray)
		if !ok {
			continue
		}

		for j := 0; j < pointsArray.Length(); j++ {
			pointObj, err := pointsArray.GetAt(j)
			if err != nil {
				continue
			}

			pointArray, ok := pointObj.(*jsonutils.JSONArray)
			if !ok || pointArray.Length() < 2 {
				continue
			}

			timestamp, err := pointArray.GetAt(0)
			if err != nil {
				continue
			}

			value, err := pointArray.GetAt(1)
			if err != nil {
				continue
			}

			timestampStr, _ := timestamp.GetString()
			valueStr, _ := value.GetString()

			timestampInt, _ := strconv.ParseInt(timestampStr, 10, 64)
			valueFloat, _ := strconv.ParseFloat(valueStr, 64)

			metricData.Values = append(metricData.Values, models.MetricValue{
				Timestamp: timestampInt,
				Value:     valueFloat,
			})
		}

		response.Data.Metrics = append(response.Data.Metrics, metricData)
	}

	return response, nil
}

// GetServerStats 获取 Cloudpods 中服务器的实时统计数据
func (a *CloudpodsAdapter) GetServerStats(ctx context.Context, serverId string, ak string, sk string) (*models.ServerStatsResponse, error) {
	session, err := a.getSession(ctx, ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	result, err := compute.Servers.GetSpecific(session, serverId, "stats", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get server stats: %w", err)
	}

	statsData := models.ServerStatsData{}

	cpuUsed, _ := result.Float("cpu_used")
	statsData.CPUUsage = cpuUsed * 100

	memSize, _ := result.Int("mem_size")
	memUsed, _ := result.Int("mem_used")
	if memSize > 0 {
		statsData.MemUsage = float64(memUsed) / float64(memSize) * 100
	}

	diskSize, _ := result.Int("disk_size")
	diskUsed, _ := result.Int("disk_used")
	if diskSize > 0 {
		statsData.DiskUsage = float64(diskUsed) / float64(diskSize) * 100
	}

	netInRate, _ := result.Float("net_in_rate")
	netOutRate, _ := result.Float("net_out_rate")
	statsData.NetBpsRx = int64(netInRate)
	statsData.NetBpsTx = int64(netOutRate)

	statsData.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	response := &models.ServerStatsResponse{
		Status: 200,
		Data:   statsData,
	}

	return response, nil
}
