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
	// 关机自动重置
	AutoReset bool `json:"auto_reset"`
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

type GuestdiskJsonDesc struct {
	DiskId        string `json:"disk_id"`
	Driver        string `json:"driver"`
	CacheMode     string `json:"cache_mode"`
	AioMode       string `json:"aio_mode"`
	Iops          int    `json:"iops"`
	Throughput    int    `json:"throughput"`
	Bps           int    `json:"bps"`
	Size          int    `json:"size"`
	TemplateId    string `json:"template_id"`
	ImagePath     string `json:"image_path"`
	StorageId     string `json:"storage_id"`
	StorageType   string `json:"storage_type"`
	Migrating     bool   `json:"migrating"`
	Path          string `json:"path"`
	Format        string `json:"format"`
	Index         int8   `json:"index"`
	BootIndex     *int8  `json:"boot_index"`
	MergeSnapshot bool   `json:"merge_snapshot"`
	Fs            string `json:"fs"`
	Mountpoint    string `json:"mountpoint"`
	Dev           string `json:"dev"`
	IsSSD         bool   `json:"is_ssd"`
	NumQueues     uint8  `json:"num_queues"`
	AutoReset     bool   `json:"auto_reset"`

	// esxi
	ImageInfo struct {
		ImageType          string `json:"image_type"`
		ImageExternalId    string `json:"image_external_id"`
		StorageCacheHostIp string `json:"storage_cache_host_ip"`
	} `json:"image_info"`
	Preallocation string `json:"preallocation"`

	TargetStorageId string `json:"target_storage_id"`

	EsxiFlatFilePath string `json:"esxi_flat_file_path"`
	Url              string `json:"url"`
}
