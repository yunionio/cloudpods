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
	"yunion.io/x/onecloud/pkg/apis"
)

type StoragecachedimageDetails struct {
	SStoragecachedimage

	apis.JointResourceBaseDetails

	StoragecacheResourceInfo

	// 缓存镜像名称
	Cachedimage string `json:"cachedimage"`

	// 存储列表
	// Storages []string `json:"storages"`
	// 通过一致性哈希获取的一个宿主机详情
	// Host *jsonutils.JSONDict `json:"host"`

	// 镜像名称
	Image string `json:"image"`
	// 镜像大小
	Size int64 `json:"size"`
	// 引用次数
	Reference int `json:"reference"`
	// Disk引用次数
	DiskReference int `json:"disk_reference"`
	// Cdrom引用次数
	CdromReference int `json:"cdrom_reference"`
}

type StoragecachedimageListInput struct {
	apis.JointResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	StoragecacheFilterListInput

	// 以镜像缓存过滤
	CachedimageId string `json:"cachedimage_id"`
	// Deprecated
	// swagger:ignore
	Cachedimage string `json:"cachedimage" yunion-deprecated-by:"cachedimage_id"`

	// 镜像状态
	Status []string `json:"status"`
}
