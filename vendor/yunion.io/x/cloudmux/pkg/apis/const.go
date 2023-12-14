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

package apis

const (
	STATUS_DELETING      = "deleting"
	STATUS_DELETE_FAILED = "delete_failed"
	STATUS_CREATING      = "creating"
	STATUS_CREATE_FAILED = "create_failed"
	STATUS_AVAILABLE     = "available"
	STATUS_UNKNOWN       = "unknown"

	USER_TAG_PREFIX = "user:"

	SKU_STATUS_AVAILABLE = "available"
	SKU_STATUS_SOLDOUT   = "soldout"
)

const (
	OS_ARCH_X86 = "x86"
	OS_ARCH_ARM = "arm"

	OS_ARCH_I386    = "i386"
	OS_ARCH_X86_32  = "x86_32"
	OS_ARCH_X86_64  = "x86_64"
	OS_ARCH_AARCH32 = "aarch32"
	OS_ARCH_AARCH64 = "aarch64"
)

const (
	PUBLIC_CLOUD_ANSIBLE_USER = "cloudroot"
)
