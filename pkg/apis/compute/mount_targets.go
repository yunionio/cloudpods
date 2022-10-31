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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	MOUNT_TARGET_STATUS_AVAILABLE     = compute.MOUNT_TARGET_STATUS_AVAILABLE
	MOUNT_TARGET_STATUS_UNAVAILABLE   = compute.MOUNT_TARGET_STATUS_UNAVAILABLE
	MOUNT_TARGET_STATUS_CREATING      = compute.MOUNT_TARGET_STATUS_CREATING
	MOUNT_TARGET_STATUS_CREATE_FAILED = "create_failed"
	MOUNT_TARGET_STATUS_DELETING      = compute.MOUNT_TARGET_STATUS_DELETING
	MOUNT_TARGET_STATUS_DELETE_FAILED = "delete_failed"
	MOUNT_TARGET_STATUS_UNKNOWN       = "unknown"
)

type MountTargetListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	AccessGroupFilterListInput
	VpcFilterListInput
	NetworkFilterListInput

	// 文件系统Id
	FileSystemId string `json:"file_system_id"`
}

type MountTargetDetails struct {
	apis.StatusStandaloneResourceDetails
	AccessGroupResourceInfo
	VpcResourceInfo
	NetworkResourceInfo

	FileSystem string
}

type MountTargetCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// 网络类型
	// enmu: vpc, classic
	// default: vpc
	NetworkType string `json:"network_type"`

	// 文件系统Id
	// required: true
	FileSystemId string `json:"file_system_id"`

	// Ip子网名称或Id, network_type == vpc时有效
	// required: true
	NetworkId string `json:"network_id"`
	// swagger:ignore
	Network string `json:"network" yunion-deprecated-by:"network_id"`

	// swagger:ignore
	VpcId string `json:"vpc_id"`

	// 权限组Id
	AccessGroupId string `json:"access_group_id"`
}

type MountTargetSyncstatusInput struct {
}
