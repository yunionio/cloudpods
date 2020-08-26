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

package compute

import (
	"yunion.io/x/jsonutils"
)

type HoststorageDetails struct {
	HostJointResourceDetails

	SHoststorage

	// 存储名称
	Storage string `json:"storage"`
	// 存储大小
	Capacity int64 `json:"capacity"`
	// 存储配置信息
	StorageConf jsonutils.JSONObject `json:"storage_conf"`
	// 已使用存储大小
	UsedCapacity int64 `json:"used_capacity"`
	// 浪费存储大小(异常磁盘总大小)
	WasteCapacity int64 `json:"waste_capacity"`
	// 可用存储大小
	FreeCapacity int64 `json:"free_capacity"`
	// 存储类型
	// example: local
	StorageType string `json:"storage_type"`
	// 介质类型
	// example: ssd
	MediumType string `json:"medium_type"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 超售比
	Cmtbound float32 `json:"cmtbound"`
	// 镜像缓存路径
	ImagecachePath string `json:"imagecache_path"`
	// 存储缓存Id
	StoragecacheId string `json:"storagecache_id"`

	GuestDiskCount int `json:"guest_disk_count,allowempty"`
}

type HoststorageListInput struct {
	HostJointsListInput

	StorageFilterListInput
}
