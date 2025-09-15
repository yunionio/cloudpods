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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// ListCloudRegions 查询 Cloudpods 中的区域列表
func (a CloudpodsAdapter) ListCloudRegions(ctx context.Context, limit int, offset int, search string, provider string, ak string, sk string) (*models.CloudregionListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if provider != "" {
		// 设置云提供商过滤条件
		providers := jsonutils.NewArray()
		providers.Add(jsonutils.NewString(provider))
		params.Set("providers", providers)
	}

	// 调用 Cloudpods API 查询区域列表
	result, err := compute.Cloudregions.List(session, params)
	if err != nil {
		return nil, err
	}

	// 构造响应数据
	response := &models.CloudregionListResponse{
		Limit:        int64(limit),
		Offset:       int64(offset),
		Cloudregions: make([]models.CloudregionDetails, 0),
		Total:        int64(result.Total),
	}
	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		region := models.CloudregionDetails{}
		if err := data.Unmarshal(&region); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal cloudregion details: %s", err)
			continue
		}
		response.Cloudregions = append(response.Cloudregions, region)
	}

	return response, nil
}

// ListVPCs 查询 Cloudpods 中的 VPC 列表
func (a *CloudpodsAdapter) ListVPCs(limit int, offset int, search string, cloudregionId string, ak string, sk string) (*models.VpcListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if cloudregionId != "" {
		// 设置云区域 ID 过滤条件
		cloudregionIds := jsonutils.NewArray()
		cloudregionIds.Add(jsonutils.NewString(cloudregionId))
		params.Set("cloudregion_id", cloudregionIds)
	}

	// 调用 Cloudpods API 查询 VPC 列表
	result, err := compute.Vpcs.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list vpcs: %w", err)
	}

	// 构造响应数据
	response := &models.VpcListResponse{
		Limit:  int64(limit),
		Offset: int64(offset),
		Vpcs:   make([]models.VpcDetails, 0),
		Total:  int64(result.Total),
	}
	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		vpc := models.VpcDetails{}
		if err := data.Unmarshal(&vpc); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal vpc details: %s", err)
			continue
		}
		response.Vpcs = append(response.Vpcs, vpc)
	}

	return response, nil
}

// ListNetworks 查询 Cloudpods 中的网络列表
func (a *CloudpodsAdapter) ListNetworks(limit int, offset int, search string, vpcId string, ak string, sk string) (*models.NetworkListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if vpcId != "" {
		// 设置 VPC ID 过滤条件
		//vpcIds := jsonutils.NewArray()
		//vpcIds.Add(jsonutils.NewString(vpcId))
		//params.Set("vpc_id", vpcIds)
		params.Set("vpc_id", jsonutils.NewString(vpcId))
	}

	// 调用 Cloudpods API 查询网络列表
	result, err := compute.Networks.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	// 构造响应数据
	response := &models.NetworkListResponse{
		Limit:    int64(limit),
		Offset:   int64(offset),
		Networks: make([]models.NetworkDetails, 0),
		Total:    int64(result.Total),
	}
	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		network := models.NetworkDetails{}
		if err := data.Unmarshal(&network); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal network details: %s", err)
			continue
		}
		response.Networks = append(response.Networks, network)
	}

	return response, nil
}

// ListImages 查询 Cloudpods 中的镜像列表
func (a *CloudpodsAdapter) ListImages(limit int, offset int, search string, osTypes []string, ak string, sk string) (*models.ImageListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if len(osTypes) > 0 {
		// 设置操作系统类型过滤条件
		osTypesArray := jsonutils.NewArray()
		for _, osType := range osTypes {
			osTypesArray.Add(jsonutils.NewString(osType))
		}
		params.Set("os_types", osTypesArray)
	}

	// 调用 Cloudpods API 查询镜像列表
	result, err := image.Images.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// 构造响应数据
	response := &models.ImageListResponse{
		Limit:  int64(limit),
		Offset: int64(offset),
		Images: make([]models.ImageDetails, 0),
		Total:  int64(result.Total),
	}
	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		image := models.ImageDetails{}
		if err := data.Unmarshal(&image); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal image details: %s", err)
			continue
		}
		response.Images = append(response.Images, image)
	}

	return response, nil
}

// ListServerSkus 查询 Cloudpods 中的服务器规格列表
func (a *CloudpodsAdapter) ListServerSkus(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, cpuCoreCount []string, memorySizeMB []string, providers []string, cpuArch []string, ak string, sk string) (*models.ServerSkuListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if len(cloudregionIds) > 0 {
		// 设置云区域 ID 过滤条件
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		// 设置可用区 ID 过滤条件
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(cpuCoreCount) > 0 {
		// 设置 CPU 核心数过滤条件
		cpuCoreArray := jsonutils.NewArray()
		for _, count := range cpuCoreCount {
			cpuCoreArray.Add(jsonutils.NewString(count))
		}
		params.Set("cpu_core_count", cpuCoreArray)
	}
	if len(memorySizeMB) > 0 {
		// 设置内存大小过滤条件
		memoryArray := jsonutils.NewArray()
		for _, size := range memorySizeMB {
			memoryArray.Add(jsonutils.NewString(size))
		}
		params.Set("memory_size_mb", memoryArray)
	}
	if len(providers) > 0 {
		// 设置提供商过滤条件
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(cpuArch) > 0 {
		// 设置 CPU 架构过滤条件
		cpuArchArray := jsonutils.NewArray()
		for _, arch := range cpuArch {
			cpuArchArray.Add(jsonutils.NewString(arch))
		}
		params.Set("cpu_arch", cpuArchArray)
	}

	// 调用 Cloudpods API 查询服务器规格列表
	result, err := compute.ServerSkus.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list server skus: %w", err)
	}

	// 构造响应数据
	response := &models.ServerSkuListResponse{
		Limit:      int64(limit),
		Offset:     int64(offset),
		Serverskus: make([]models.ServerSkuDetails, 0),
		Total:      int64(result.Total),
	}
	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		sku := models.ServerSkuDetails{}
		if err := data.Unmarshal(&sku); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal server sku details: %s", err)
			continue
		}
		response.Serverskus = append(response.Serverskus, sku)
	}

	return response, nil
}

// ListStorages 查询 Cloudpods 中的存储列表
func (a *CloudpodsAdapter) ListStorages(limit int, offset int, search string, cloudregionIds []string, zoneIds []string, providers []string, storageTypes []string, hostId string, ak string, sk string) (*models.StorageListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if len(cloudregionIds) > 0 {
		// 设置云区域 ID 过滤条件
		cloudregionIdArray := jsonutils.NewArray()
		for _, id := range cloudregionIds {
			cloudregionIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("cloudregion_id", cloudregionIdArray)
	}
	if len(zoneIds) > 0 {
		// 设置可用区 ID 过滤条件
		zoneIdArray := jsonutils.NewArray()
		for _, id := range zoneIds {
			zoneIdArray.Add(jsonutils.NewString(id))
		}
		params.Set("zone_ids", zoneIdArray)
	}
	if len(providers) > 0 {
		// 设置提供商过滤条件
		providerArray := jsonutils.NewArray()
		for _, provider := range providers {
			providerArray.Add(jsonutils.NewString(provider))
		}
		params.Set("providers", providerArray)
	}
	if len(storageTypes) > 0 {
		// 设置存储类型过滤条件
		for _, storageType := range storageTypes {
			params.Set("storage_type", jsonutils.NewString(storageType))
			break
		}
	}
	if hostId != "" {
		// 设置主机 ID 过滤条件
		params.Set("host_id", jsonutils.NewString(hostId))
	}

	// 调用 Cloudpods API 查询存储列表
	result, err := compute.Storages.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list storages: %w", err)
	}

	// 构造响应数据
	response := &models.StorageListResponse{
		Limit:    int64(limit),
		Offset:   int64(offset),
		Storages: make([]models.StorageDetails, 0),
		Total:    int64(result.Total),
	}

	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		storage := models.StorageDetails{}
		if err := data.Unmarshal(&storage); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal storage details: %s", err)
			continue
		}
		response.Storages = append(response.Storages, storage)
	}

	return response, nil
}

// ListServers 查询 Cloudpods 中的服务器列表
func (a *CloudpodsAdapter) ListServers(ctx context.Context, limit int, offset int, search string, status string, ak string, sk string) (*models.ServerListResponse, error) {
	// 获取 Cloudpods 会话
	session, err := a.getSession(ak, sk)
	if err != nil {
		return nil, err
	}

	// 构造查询参数
	params := jsonutils.NewDict()
	if limit > 0 {
		// 设置查询结果数量限制
		params.Set("limit", jsonutils.NewInt(int64(limit)))
	}
	if offset > 0 {
		// 设置查询偏移量
		params.Set("offset", jsonutils.NewInt(int64(offset)))
	}
	if search != "" {
		// 设置搜索关键字
		params.Set("search", jsonutils.NewString(search))
	}
	if status != "" {
		// 设置服务器状态过滤条件
		params.Set("status", jsonutils.NewString(status))
	}

	// 调用 Cloudpods API 查询服务器列表
	result, err := compute.Servers.List(session, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	// 构造响应数据
	response := &models.ServerListResponse{
		Limit:   int64(limit),
		Offset:  int64(offset),
		Servers: make([]models.ServerDetails, 0),
		Total:   int64(result.Total),
	}

	// 遍历查询结果，将数据转换为响应格式
	for _, data := range result.Data {
		server := models.ServerDetails{}
		if err := data.Unmarshal(&server); err != nil {
			// 如果数据转换失败，记录警告日志并跳过该条数据
			log.Warningf("Failed to unmarshal server details: %s", err)
			continue
		}
		response.Servers = append(response.Servers, server)
	}

	return response, nil
}
