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

import "yunion.io/x/onecloud/pkg/apis"

type CachedImageUncacheImageInput struct {
	// 存储缓存名Id
	// required: true
	StoragecacheId string `json:"storagecache_id"`

	// 是否强制清除缓存
	// default: false
	IsForce bool `json:"is_force"`
}

type CachedimageDetails struct {
	apis.SharableVirtualResourceDetails

	SCachedimage

	// 镜像状态, 和info里面的status一致
	// example: active
	Status string `json:"status"`

	// 操作系统类型
	// example: FreeBSD
	OsType string `json:"os_type"`

	// 操作系统发行版
	// example: FreeBSD
	OsDistribution string `json:"os_distribution"`

	// 操作系统版本
	// example: 11
	OsVersion string `json:"os_version"`

	// 虚拟化类型
	Hypervisor string `json:"hypervisor"`

	// 此镜像被使用次数
	// example: 0
	CachedCount int `json:"cached_count"`
}
