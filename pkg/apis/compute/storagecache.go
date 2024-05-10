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

	"yunion.io/x/onecloud/pkg/apis"
)

type StoragecacheDetails struct {
	apis.StandaloneResourceDetails
	ManagedResourceInfo

	SStoragecache

	// 存储列表
	Storages []string `json:"storages"`
	// 缓存镜像总大小
	Size int64 `json:"size"`
	// 缓存镜像个数
	Count int `json:"count"`
	// 通过一致性哈希获取的一个管理宿主机信息
	Host *jsonutils.JSONDict `json:"host"`
}

type StoragecacheListInput struct {
	apis.StandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput

	// 路径过滤
	Path []string `json:"path"`
}

type CacheImageInput struct {
	Image   string `json:"image" yunion-deprecated-by:"image_id"`
	ImageId string `json:"image_id"`
	IsForce bool   `json:"is_force"`
	Format  string `json:"format"`

	// swagger: ignore
	Zone string `json:"zone"`
	// swagger: ignore
	OsType string `json:"os_type"`
	// swagger: ignore
	OsArch string `json:"os_arch"`
	// swagger: ignore
	OsDistribution string `json:"os_distribution"`
	// swagger: ignore
	OsFullVersion string `json:"os_full_version"`
	// swagger: ignore
	OsVersion string `json:"os_version"`
	// swagger: ignore
	ImageName string `json:"image_name"`

	// swagger: ignore
	ServerId string `json:"server_id"`
	// swagger: ignore
	ParentTaskId string `json:"parent_task_id"`
	// swagger: ignore
	SourceHostId string `json:"source_host_id"`
	// swagger: ignore
	SrcUrl string `json:"src_url"`
	// swagger: ignore
	StoragecacheId string `json:"storagecache_id"`
	// swagger: ignore
	Checksum string `json:"checksum"`
	// swagger: ignore
	SkipChecksumIfExists bool `json:"skip_checksum_if_exists"`
}

type StoragecacheResourceInput struct {
	// 存储缓存（ID或Name）
	StoragecacheId string `json:"storagecache_id"`
	// swagger:ignore
	// Deprecated
	// filter by storagecache_id
	Storagecache string `json:"storagecache" yunion-deprecated-by:"storagecache_id"`
}

type StoragecacheResourceInfo struct {
	// 归属云订阅ID
	ManagerId string `json:"manager_id"`

	ManagedResourceInfo

	// 存储缓存名称
	Storagecache string `json:"storagecache"`

	// 关联存储名称
	Storages []string `json:"storages"`

	// 关联存储信息
	StorageInfo []StorageInfo `json:"storage_info"`
}

type StorageInfo struct {
	Id string `json:"id"`

	Name string `json:"name"`

	StorageType string `json:"storage_type"`

	MediumType string `json:"medium_type"`

	ZoneId string `json:"zone_id"`

	Zone string `json:"zone"`
}

type StoragecacheFilterListInputBase struct {
	StoragecacheResourceInput

	// 以存储缓存名称排序
	// pattern:asc|desc
	OrderByStoragecache string `json:"order_by_storagecache"`
}

type StoragecacheFilterListInput struct {
	StoragecacheFilterListInputBase

	ManagedResourceListInput
}
