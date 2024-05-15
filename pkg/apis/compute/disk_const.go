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

import "yunion.io/x/cloudmux/pkg/apis/compute"

const (
	DISK_INIT                = compute.DISK_INIT
	DISK_REBUILD             = compute.DISK_REBUILD
	DISK_ALLOC_FAILED        = compute.DISK_ALLOC_FAILED
	DISK_STARTALLOC          = "start_alloc"
	DISK_BACKUP_STARTALLOC   = compute.DISK_BACKUP_STARTALLOC
	DISK_BACKUP_ALLOC_FAILED = compute.DISK_BACKUP_ALLOC_FAILED
	DISK_ALLOCATING          = compute.DISK_ALLOCATING
	DISK_READY               = compute.DISK_READY
	DISK_RESET               = compute.DISK_RESET
	DISK_RESET_FAILED        = compute.DISK_RESET_FAILED
	DISK_DEALLOC             = compute.DISK_DEALLOC
	DISK_DEALLOC_FAILED      = compute.DISK_DEALLOC_FAILED
	DISK_UNKNOWN             = compute.DISK_UNKNOWN
	DISK_DETACHING           = compute.DISK_DETACHING
	DISK_ATTACHING           = compute.DISK_ATTACHING
	DISK_CLONING             = compute.DISK_CLONING // 硬盘克隆

	DISK_START_SAVE = "start_save"
	DISK_SAVING     = compute.DISK_SAVING

	DISK_START_RESIZE  = "start_resize"
	DISK_RESIZING      = compute.DISK_RESIZING
	DISK_RESIZE_FAILED = compute.DISK_RESIZE_FAILED

	DISK_START_MIGRATE = "start_migrate"
	DISK_POST_MIGRATE  = "post_migrate"
	DISK_MIGRATING     = "migrating"
	DISK_MIGRATE_FAIL  = "migrate_failed"
	DISK_IMAGE_CACHING = "image_caching" // 缓存镜像中

	DISK_CLONE      = "clone"
	DISK_CLONE_FAIL = "clone_failed"

	DISK_START_SNAPSHOT       = "start_snapshot"
	DISK_SNAPSHOTING          = "snapshoting"
	DISK_APPLY_SNAPSHOT_FAIL  = "apply_snapshot_failed"
	DISK_CALCEL_SNAPSHOT_FAIL = "cancel_snapshot_failed"

	DISK_TYPE_SYS    = compute.DISK_TYPE_SYS
	DISK_TYPE_SWAP   = compute.DISK_TYPE_SWAP
	DISK_TYPE_DATA   = compute.DISK_TYPE_DATA
	DISK_TYPE_VOLUME = "volume"

	DISK_BACKING_IMAGE = "image"

	DISK_SIZE_AUTOEXTEND = -1

	DISK_NOT_EXIST = "not_exist"
	DISK_EXIST     = "exist"

	DISK_PREALLOCATION_OFF = "off"
	// 精简置备
	DISK_PREALLOCATION_METADATA = "metadata"
	// 厚置备延迟归零
	DISK_PREALLOCATION_FALLOC = "falloc"
	// 厚置备快速归零
	DISK_PREALLOCATION_FULL = "full"
)

var DISK_PREALLOCATIONS = []string{
	DISK_PREALLOCATION_OFF,
	DISK_PREALLOCATION_METADATA,
	DISK_PREALLOCATION_FALLOC,
	DISK_PREALLOCATION_FULL,
}

const (
	DISK_META_EXISTING_PATH      = "disk_existing_path"
	DISK_META_LAST_ATTACHED_HOST = "__disk_last_attached_host"
)

const (
	DISK_DRIVER_VIRTIO = "virtio"
	DISK_DRIVER_SCSI   = "scsi"
	DISK_DRIVER_PVSCSI = "pvscsi"
	DISK_DRIVER_IDE    = "ide"
	DISK_DRIVER_SATA   = "sata"
	DISK_DRIVER_VFIO   = "vfio-pci"
)
