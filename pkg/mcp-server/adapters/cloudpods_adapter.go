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
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcp-server/config"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

type CloudpodsAdapter struct {
	config  *config.Config
	logger  *logrus.Logger
	client  *mcclient.Client
	session *mcclient.ClientSession
}

type CloudRegion struct {
	RegionId string `json:"region_id"`
}

func NewCloudpodsAdapter(cfg *config.Config, logger *logrus.Logger) *CloudpodsAdapter {

	client := mcclient.NewClient(
		cfg.External.Cloudpods.BaseURL,
		cfg.External.Cloudpods.Timeout,
		false,
		true,
		"",
		"",
	)

	return &CloudpodsAdapter{
		config: cfg,
		logger: logger,
		client: client,
	}
}

func (a *CloudpodsAdapter) authenticate(ak string, sk string) error {
	if a.session != nil {
		return nil
	}

	token, err := a.client.AuthenticateByAccessKey(ak, sk, "")
	if err != nil {
		return err
	}

	a.session = a.client.NewSession(
		context.Background(),
		"",
		"",
		"publicURL",
		token,
	)

	return nil
}

func (a *CloudpodsAdapter) getSession(ak string, sk string) (*mcclient.ClientSession, error) {
	if err := a.authenticate(ak, sk); err != nil {
		return nil, err
	}
	return a.session, nil
}

func (a CloudpodsAdapter) ListCloudRegions(ctx context.Context, limit int, offset int, search string, provider string, ak string, sk string) (*models.CloudregionListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if provider != "" {
		providers := jsonutils.NewArray()
		providers.Add(jsonutils.NewString(provider))
		params.Set("providers", providers)
	}

	result, err := compute.Cloudregions.List(session, params)
	if err != nil {
		return nil, err
	}

	response := &models.CloudregionListResponse{
		Limit:        int64(limit),
		Offset:       int64(offset),
		Cloudregions: make([]models.CloudregionDetails, 0),
		Total:        int64(result.Total),
	}
	for _, data := range result.Data {
		region := models.CloudregionDetails{}
		if err := data.Unmarshal(&region); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal cloudregion details")
			continue
		}
		response.Cloudregions = append(response.Cloudregions, region)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListVPCs(limit int, offset int, search string, cloudregionId string, ak string, sk string) (*models.VpcListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if cloudregionId != "" {
		cloudregionIds := jsonutils.NewArray()
		cloudregionIds.Add(jsonutils.NewString(cloudregionId))
		params.Set("cloudregion_id", cloudregionIds)
	}

	result, err := compute.Vpcs.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list vpcs: %w", err)
	}

	response := &models.VpcListResponse{
		Limit:  int64(limit),
		Offset: int64(offset),
		Vpcs:   make([]models.VpcDetails, 0),
		Total:  int64(result.Total),
	}

	for _, data := range result.Data {
		vpc := models.VpcDetails{}
		if err := data.Unmarshal(&vpc); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal vpc details")
			continue
		}
		response.Vpcs = append(response.Vpcs, vpc)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListNetworks(limit int, offset int, search string, vpcId string, ak string, sk string) (*models.NetworkListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if vpcId != "" {
		vpcIds := jsonutils.NewArray()
		vpcIds.Add(jsonutils.NewString(vpcId))
		params.Set("vpc_id", vpcIds)
	}

	result, err := compute.Networks.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	response := &models.NetworkListResponse{
		Limit:    int64(limit),
		Offset:   int64(offset),
		Networks: make([]models.NetworkDetails, 0),
		Total:    int64(result.Total),
	}

	for _, data := range result.Data {
		network := models.NetworkDetails{}
		if err := data.Unmarshal(&network); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal network details")
			continue
		}
		response.Networks = append(response.Networks, network)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListImages(limit int, offset int, search string, osTypes []string, ak string, sk string) (*models.ImageListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if len(osTypes) > 0 {
		osTypesArray := jsonutils.NewArray()
		for _, osType := range osTypes {
			osTypesArray.Add(jsonutils.NewString(osType))
		}
		params.Set("os_types", osTypesArray)
	}

	result, err := image.Images.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	response := &models.ImageListResponse{
		Limit:  int64(limit),
		Offset: int64(offset),
		Images: make([]models.ImageDetails, 0),
		Total:  int64(result.Total),
	}

	for _, data := range result.Data {
		image := models.ImageDetails{}
		if err := data.Unmarshal(&image); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal image details")
			continue
		}
		response.Images = append(response.Images, image)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListServerSkus(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, cpuCoreCount []string, memorySizeMB []string, providers []string, cpuArch []string, ak string, sk string) (*models.ServerSkuListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if len(cloudregionIds) > 0 {
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(cpuCoreCount) > 0 {
		cpuCoreArray := jsonutils.NewArray()
		for _, count := range cpuCoreCount {
			cpuCoreArray.Add(jsonutils.NewString(count))
		}
		params.Set("cpu_core_count", cpuCoreArray)
	}
	if len(memorySizeMB) > 0 {
		memoryArray := jsonutils.NewArray()
		for _, size := range memorySizeMB {
			memoryArray.Add(jsonutils.NewString(size))
		}
		params.Set("memory_size_mb", memoryArray)
	}
	if len(providers) > 0 {
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(cpuArch) > 0 {
		cpuArchArray := jsonutils.NewArray()
		for _, arch := range cpuArch {
			cpuArchArray.Add(jsonutils.NewString(arch))
		}
		params.Set("cpu_arch", cpuArchArray)
	}

	result, err := compute.ServerSkus.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list server skus: %w", err)
	}

	response := &models.ServerSkuListResponse{
		Limit:      int64(limit),
		Offset:     int64(offset),
		Serverskus: make([]models.ServerSkuDetails, 0),
		Total:      int64(result.Total),
	}

	for _, data := range result.Data {
		sku := models.ServerSkuDetails{}
		if err := data.Unmarshal(&sku); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal server sku details")
			continue
		}
		response.Serverskus = append(response.Serverskus, sku)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListStorages(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, providers []string, storageTypes []string, hostId string, ak string, sk string) (*models.StorageListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if len(cloudregionIds) > 0 {
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(providers) > 0 {
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(storageTypes) > 0 {
		for _, storageType := range storageTypes {
			params.Set("storage_type", jsonutils.NewString(storageType))
			break
		}
	}
	if hostId != "" {
		params.Set("host_id", jsonutils.NewString(hostId))
	}

	result, err := compute.Storages.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list storages: %w", err)
	}

	response := &models.StorageListResponse{
		Limit:    int64(limit),
		Offset:   int64(offset),
		Storages: make([]models.StorageDetails, 0),
		Total:    int64(result.Total),
	}

	for _, data := range result.Data {
		storage := models.StorageDetails{}
		if err := data.Unmarshal(&storage); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal storage details")
			continue
		}
		response.Storages = append(response.Storages, storage)
	}

	return response, nil
}

func (a *CloudpodsAdapter) ListServers(ctx context.Context, limit int, offset int, search string, status string, ak string, sk string) (*models.ServerListResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if limit > 0 {
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		params.Set("search", jsonutils.NewString(search))
	}
	if status != "" {
		params.Set("status", jsonutils.NewString(status))
	}

	result, err := compute.Servers.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	response := &models.ServerListResponse{
		Limit:   int64(limit),
		Offset:  int64(offset),
		Servers: make([]models.ServerDetails, 0),
		Total:   int64(result.Total),
	}

	for _, data := range result.Data {
		server := models.ServerDetails{}
		if err := data.Unmarshal(&server); err != nil {
			a.logger.WithError(err).Warn("Failed to unmarshal server details")
			continue
		}
		response.Servers = append(response.Servers, server)
	}

	return response, nil
}

func (a *CloudpodsAdapter) StartServer(ctx context.Context, serverId string, req models.ServerStartRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()

	if req.AutoPrepaid {
		params.Set("auto_prepaid", jsonutils.NewBool(true))
	}

	if req.QemuVersion != "" {
		params.Set("qemu_version", jsonutils.NewString(req.QemuVersion))
	}

	result, err := compute.Servers.PerformAction(session, serverId, "start", params)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	response := &models.ServerOperationResponse{
		Operation: "start",
	}

	if err := result.Unmarshal(response); err != nil {
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		response.Success = taskId != ""
	}

	return response, nil
}

func (a *CloudpodsAdapter) StopServer(ctx context.Context, serverId string, req models.ServerStopRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()

	if req.IsForce {
		params.Set("is_force", jsonutils.NewBool(true))
	}

	if req.StopCharging {
		params.Set("stop_charging", jsonutils.NewBool(true))
	}

	if req.TimeoutSecs > 0 {
		params.Set("timeout_secs", jsonutils.NewInt(req.TimeoutSecs))
	}

	result, err := compute.Servers.PerformAction(session, serverId, "stop", params)
	if err != nil {
		return nil, fmt.Errorf("failed to stop server: %w", err)
	}

	response := &models.ServerOperationResponse{
		Operation: "stop",
	}

	if err := result.Unmarshal(response); err != nil {
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		response.Success = taskId != ""
	}

	return response, nil
}

func (a *CloudpodsAdapter) RestartServer(ctx context.Context, serverId string, req models.ServerRestartRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if req.IsForce {
		params.Set("is_force", jsonutils.NewBool(true))
	}

	result, err := compute.Servers.PerformAction(session, serverId, "restart", params)
	if err != nil {
		return nil, fmt.Errorf("failed to restart server: %w", err)
	}

	response := &models.ServerOperationResponse{
		Operation: "restart",
	}

	if err := result.Unmarshal(response); err != nil {
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		response.Success = taskId != ""
	}

	return response, nil
}

func (a *CloudpodsAdapter) ResetServerPassword(ctx context.Context, serverId string, req models.ServerResetPasswordRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()

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

	result, err := compute.Servers.PerformAction(session, serverId, "reset-password", params)
	if err != nil {
		return nil, fmt.Errorf("failed to reset server password: %w", err)
	}

	response := &models.ServerOperationResponse{
		Operation: "reset-password",
	}

	if err := result.Unmarshal(response); err != nil {
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		response.Success = taskId != ""
	}

	return response, nil
}

func (a *CloudpodsAdapter) DeleteServer(ctx context.Context, serverId string, req models.ServerDeleteRequest, ak string, sk string) (*models.ServerOperationResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	if req.OverridePendingDelete {
		params.Set("override_pending_delete", jsonutils.NewBool(true))
	}
	if req.Purge {
		params.Set("purge", jsonutils.NewBool(true))
	}
	if req.DeleteSnapshots {
		params.Set("delete_snapshots", jsonutils.NewBool(true))
	}
	if req.DeleteEip {
		params.Set("delete_eip", jsonutils.NewBool(true))
	}
	if req.DeleteDisks {
		params.Set("delete_disks", jsonutils.NewBool(true))
	}

	result, err := compute.Servers.Delete(session, serverId, params)
	if err != nil {
		return nil, fmt.Errorf("failed to delete server: %w", err)
	}

	response := &models.ServerOperationResponse{
		Operation: "delete",
	}

	if err := result.Unmarshal(response); err != nil {
		taskId, _ := result.GetString("task_id")
		response.TaskId = taskId
		response.Success = taskId != ""
	}

	return response, nil
}

func (a *CloudpodsAdapter) CreateServer(ctx context.Context, req models.CreateServerRequest, ak string, sk string) (*models.CreateServerResponse, error) {
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(req.Name))
	params.Set("vcpu_count", jsonutils.NewInt(req.VcpuCount))
	params.Set("vmem_size", jsonutils.NewInt(req.VmemSize))

	if req.Count > 1 {
		params.Set("count", jsonutils.NewInt(int64(req.Count)))
	}

	if req.AutoStart {
		params.Set("auto_start", jsonutils.NewBool(req.AutoStart))
	}

	if req.Password != "" {
		params.Set("password", jsonutils.NewString(req.Password))
	}

	if req.BillingType != "" {
		params.Set("billing_type", jsonutils.NewString(req.BillingType))
	}

	if req.Duration != "" {
		params.Set("duration", jsonutils.NewString(req.Duration))
	}

	if req.Description != "" {
		params.Set("description", jsonutils.NewString(req.Description))
	}

	if req.Hostname != "" {
		params.Set("hostname", jsonutils.NewString(req.Hostname))
	}

	if req.Hypervisor != "" {
		params.Set("hypervisor", jsonutils.NewString(req.Hypervisor))
	}

	if req.UserData != "" {
		params.Set("user_data", jsonutils.NewString(req.UserData))
	}

	if req.KeypairId != "" {
		params.Set("keypair_id", jsonutils.NewString(req.KeypairId))
	}

	if req.ProjectId != "" {
		params.Set("project_id", jsonutils.NewString(req.ProjectId))
	}

	if req.ZoneId != "" {
		params.Set("prefer_zone_id", jsonutils.NewString(req.ZoneId))
	}

	if req.RegionId != "" {
		params.Set("prefer_region_id", jsonutils.NewString(req.RegionId))
	}

	if req.DisableDelete {
		params.Set("disable_delete", jsonutils.NewBool(req.DisableDelete))
	}

	if req.BootOrder != "" {
		params.Set("boot_order", jsonutils.NewString(req.BootOrder))
	}

	if len(req.Metadata) > 0 {
		metaDict := jsonutils.NewDict()
		for k, v := range req.Metadata {
			metaDict.Set(k, jsonutils.NewString(v))
		}
		params.Set("__meta__", metaDict)
	}

	disks := jsonutils.NewArray()

	if req.ImageId != "" {
		diskDict := jsonutils.NewDict()
		diskDict.Set("image_id", jsonutils.NewString(req.ImageId))
		diskDict.Set("disk_type", jsonutils.NewString("sys"))
		if req.DiskSize > 0 {
			diskDict.Set("size", jsonutils.NewInt(req.DiskSize))
		}
		disks.Add(diskDict)
	}

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

	if disks.Length() > 0 {
		params.Set("disks", disks)
	}

	if req.NetworkId != "" {
		networks := jsonutils.NewArray()
		netDict := jsonutils.NewDict()
		netDict.Set("network", jsonutils.NewString(req.NetworkId))
		networks.Add(netDict)
		params.Set("nets", networks)
	}

	if req.SecgroupId != "" {
		params.Set("secgrp_id", jsonutils.NewString(req.SecgroupId))
	}

	if len(req.Secgroups) > 0 {
		secgroups := jsonutils.NewArray()
		for _, sg := range req.Secgroups {
			secgroups.Add(jsonutils.NewString(sg))
		}
		params.Set("secgroups", secgroups)
	}

	if req.ServerskuId != "" {
		params.Set("instance_type", jsonutils.NewString(req.ServerskuId))
	}

	result, err := compute.Servers.Create(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	response := &models.CreateServerResponse{}
	if err := result.Unmarshal(response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create server response: %w", err)
	}

	return response, nil
}

func (a *CloudpodsAdapter) GetServerMonitor(ctx context.Context, serverId string, startTime, endTime int64, metrics []string, ak string, sk string) (*models.MonitorResponse, error) {
	session, err := a.getSession(ak, sk)
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

func (a *CloudpodsAdapter) GetServerStats(ctx context.Context, serverId string, ak string, sk string) (*models.ServerStatsResponse, error) {
	session, err := a.getSession(ak, sk)
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
