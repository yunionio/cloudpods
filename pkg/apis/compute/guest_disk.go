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

type GuestDiskDetails struct {
	GuestJointResourceDetails

	SGuestdisk

	// 磁盘名称
	Disk string

	// 存储类型
	// example: local
	StorageType string `json:"storage_type"`
	// 磁盘大小, 单位Mb
	// example: 10240
	DiskSize int `json:"disk_size"`
	// 磁盘状态
	// example: ready
	Status string `json:"status"`
	// 磁盘类型
	// example: data
	DiskType string `json:"disk_type"`
	// 介质类型
	// example: ssd
	MediumType string `json:"medium_type"`
}

type GuestdiskListInput struct {
	GuestJointsListInput

	DiskFilterListInput

	Driver []string `json:"driver"`

	CacheMode []string `json:"cache_mode"`

	AioMode []string `json:"aio_mode"`
}

type GuestdiskUpdateInput struct {
	GuestJointBaseUpdateInput

	Driver string `json:"driver"`

	CacheMode string `json:"cache_mode"`

	AioMode string `json:"aio_mode"`

	Iops *int `json:"iops"`

	Bps *int `json:"bps"`

	Index *int8 `json:"index"`
}
