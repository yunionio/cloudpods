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

	"yunion.io/x/onecloud/pkg/apis"
)

type ImageListInput struct {
	apis.SharableVirtualResourceListInput

	// 以镜像的格式过滤，可能值为：qcow2, iso, vmdk, vhd, raw等
	DiskFormats []string `json:"disk_formats"`
	// 列出是否支持UEFI启动的镜像
	Uefi *bool `json:"uefi"`

	// 是否为标准镜像
	IsStandard *bool `json:"is_standard"`

	// 是否删除保护
	Protected *bool `json:"protected"`

	// 是否为主机镜像的子镜像
	IsGuestImage *bool `json:"is_guest_image"`

	// 是否为数据盘
	IsData *bool `json:"is_data"`
}

type GuestImageListInput struct {
	apis.SharableVirtualResourceListInput

	// 是否删除保护
	Protected *bool `json:"protected"`
}

type ImageDetails struct {
	apis.SharableVirtualResourceDetails

	SImage

	// 镜像属性信息
	Properties map[string]string `json:"properties"`
	// 自动清除时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 删除保护
	DisableDelete bool `json:"disable_delete"`
	//OssChecksum   string    `json:"oss_checksum"`
}
