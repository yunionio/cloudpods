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

package image

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type GuestImageDetails struct {
	apis.SharableVirtualResourceDetails
	apis.EncryptedResourceDetails

	SGuestImage

	//Status     string               `json:"status"`
	Size       int64          `json:"size"`
	MinRamMb   int32          `json:"min_ram_mb"`
	DiskFormat string         `json:"disk_format"`
	RootImage  SubImageInfo   `json:"root_image"`
	DataImages []SubImageInfo `json:"data_images"`

	Properties *jsonutils.JSONDict `json:"properties"`

	DisableDelete bool `json:"disable_delete"`
}

type SubImageInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	MinDiskMB  int32     `json:"min_disk_mb"`
	DiskFormat string    `json:"disk_format"`
	Size       int64     `json:"size"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`

	EncryptKeyId  string `json:"encrypt_key_id"`
	DisableDelete bool   `json:"disable_delete"`
}

type GuestImageCreateInputBase struct {
	apis.SharableVirtualResourceCreateInput
	apis.EncryptedResourceCreateInput

	// 备注
	Notes string `json:"notes"`
	// 镜像格式
	DiskFormat string `json:"disk_format"`
	// 是否有删除保护
	Protected *bool `json:"protected"`
}

type GuestImageCreateInputSubimage struct {
	// Id
	Id string `json:"id"`
	// 磁盘格式
	DiskFormat string `json:"disk_format"`
	// 磁盘大小
	VirtualSize int `json:"virtual_size"`
}

type GuestImageCreateInput struct {
	GuestImageCreateInputBase

	// 镜像列表
	Images []GuestImageCreateInputSubimage `json:"images"`

	// 镜像大小, 单位Byte
	Size *int64 `json:"size"`
	// CPU架构 x86_64 or aarch64
	OsArch string `json:"os_arch"`

	// 镜像属性
	Properties map[string]string `json:"properties"`
}
