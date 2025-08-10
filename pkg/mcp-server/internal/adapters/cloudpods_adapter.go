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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/config"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/models"
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

func (a *CloudpodsAdapter) authenticate() error {
	if a.session != nil {
		return nil
	}

	token, err := a.client.AuthenticateByAccessKey("", "", "")
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

func (a *CloudpodsAdapter) getSession() (*mcclient.ClientSession, error) {
	if err := a.authenticate(); err != nil {
		return nil, err
	}
	return a.session, nil
}

func (a CloudpodsAdapter) ListCloudRegions(ctx context.Context, limit int, offset int, search string, provider string) (*models.CloudregionListResponse, error) {
	session, err := a.getSession()
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
		// 根据接口文档，使用providers参数
		providers := jsonutils.NewArray()
		providers.Add(jsonutils.NewString(provider))
		params.Set("providers", providers)
	}

	result, err := compute.Cloudregions.List(session, params)
	if err != nil {
		return nil, err
	}

	// 构造响应结构体
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

func (a *CloudpodsAdapter) ListVPCs(limit int, offset int, search string, cloudregionId string) (*models.VpcListResponse, error) {
	session, err := a.getSession()
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

	// 构造响应结构体
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

// ListNetworks 查询网络列表（基于compute服务的networks接口）
func (a *CloudpodsAdapter) ListNetworks(limit int, offset int, search string, vpcId string) (*models.NetworkListResponse, error) {
	session, err := a.getSession()
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

// ListImages 查询镜像列表（基于image服务的images接口）
func (a *CloudpodsAdapter) ListImages(limit int, offset int, search string, osTypes []string) (*models.ImageListResponse, error) {
	session, err := a.getSession()
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
		// 根据接口文档，使用os_types参数过滤
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

	// 构造响应结构体
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

// ListServerSkus 查询主机套餐规格列表（基于compute服务的serverskus接口）
func (a *CloudpodsAdapter) ListServerSkus(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, cpuCoreCount []string, memorySizeMB []string, providers []string, cpuArch []string) (*models.ServerSkuListResponse, error) {
	session, err := a.getSession()
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
		// 根据接口文档，cloudregion_id使用数组格式
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		// 根据接口文档，zone_ids使用数组格式
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(cpuCoreCount) > 0 {
		// 根据接口文档，cpu_core_count使用数组格式
		cpuCoreArray := jsonutils.NewArray()
		for _, count := range cpuCoreCount {
			cpuCoreArray.Add(jsonutils.NewString(count))
		}
		params.Set("cpu_core_count", cpuCoreArray)
	}
	if len(memorySizeMB) > 0 {
		// 根据接口文档，memory_size_mb使用数组格式
		memoryArray := jsonutils.NewArray()
		for _, size := range memorySizeMB {
			memoryArray.Add(jsonutils.NewString(size))
		}
		params.Set("memory_size_mb", memoryArray)
	}
	if len(providers) > 0 {
		// 根据接口文档，providers使用数组格式
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(cpuArch) > 0 {
		// 根据接口文档，cpu_arch使用数组格式
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

	// 构造响应结构体
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

// ListStorages 查询块存储列表（基于compute服务的storages接口）
func (a *CloudpodsAdapter) ListStorages(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, providers []string, storageTypes []string, hostId string) (*models.StorageListResponse, error) {
	session, err := a.getSession()
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
		// 根据接口文档，cloudregion_id使用数组格式
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		// 根据接口文档，zone_ids使用数组格式
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(providers) > 0 {
		// 根据接口文档，providers使用数组格式
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(storageTypes) > 0 {
		// 根据接口文档，storage_type参数过滤存储类型
		for _, storageType := range storageTypes {
			params.Set("storage_type", jsonutils.NewString(storageType))
			break // 接口文档显示storage_type是单个值，不是数组
		}
	}
	if hostId != "" {
		// 根据接口文档，使用host_id参数过滤
		params.Set("host_id", jsonutils.NewString(hostId))
	}

	result, err := compute.Storages.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list storages: %w", err)
	}

	// 构造响应结构体
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

// ListServers 查询虚拟机列表
func (a *CloudpodsAdapter) ListServers(ctx context.Context, limit int, offset int, search string, status string) (*models.ServerListResponse, error) {
	session, err := a.getSession()
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

	// 构造响应结构体
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
