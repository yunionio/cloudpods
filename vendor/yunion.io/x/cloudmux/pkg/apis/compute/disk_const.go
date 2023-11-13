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

const (
	DISK_INIT                = "init"
	DISK_REBUILD             = "rebuild"
	DISK_ALLOC_FAILED        = "alloc_failed"
	DISK_BACKUP_STARTALLOC   = "backup_start_alloc"
	DISK_BACKUP_ALLOC_FAILED = "backup_alloc_failed"
	DISK_ALLOCATING          = "allocating"
	DISK_READY               = "ready"
	DISK_RESET               = "reset"
	DISK_RESET_FAILED        = "reset_failed"
	DISK_DEALLOC             = "deallocating"
	DISK_DEALLOC_FAILED      = "dealloc_failed"
	DISK_UNKNOWN             = "unknown"
	DISK_DETACHING           = "detaching"
	DISK_ATTACHING           = "attaching"
	DISK_CLONING             = "cloning" // 硬盘克隆

	DISK_SAVING = "saving"

	DISK_RESIZING      = "resizing"
	DISK_RESIZE_FAILED = "resize_failed"

	DISK_TYPE_SYS  = "sys"
	DISK_TYPE_SWAP = "swap"
	DISK_TYPE_DATA = "data"

	DISK_PREALLOCATION_OFF = "off"
	// 精简置备
	DISK_PREALLOCATION_METADATA = "metadata"
	// 厚置备延迟归零
	DISK_PREALLOCATION_FALLOC = "falloc"
	// 厚置备快速归零
	DISK_PREALLOCATION_FULL = "full"
)
