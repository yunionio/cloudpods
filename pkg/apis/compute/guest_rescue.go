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

// Rescue constants are used for rescue mode
const (
	GUEST_RESCUE_RELATIVE_PATH = "rescue" //  serverxxx/rescue

	GUEST_RESCUE_INITRAMFS       = "initramfs"
	GUEST_RESCUE_KERNEL          = "kernel"
	GUEST_RESCUE_INITRAMFS_ARM64 = "initramfs_aarch64"
	GUEST_RESCUE_KERNEL_ARM64    = "kernel_aarch64"

	GUEST_RESCUE_SYS_DISK_NAME = "sys_img"
	GUEST_RESCUE_SYS_DISK_SIZE = 500 // MB
)
