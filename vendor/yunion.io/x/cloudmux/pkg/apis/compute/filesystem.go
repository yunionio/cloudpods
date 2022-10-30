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
	// 可用
	NAS_STATUS_AVAILABLE = "available"
	// 不可用
	NAS_STATUS_UNAVAILABLE = "unavailable"
	// 扩容中
	NAS_STATUS_EXTENDING = "extending"
	// 创建中
	NAS_STATUS_CREATING = "creating"
	// 创建失败
	NAS_STATUS_CREATE_FAILED = "create_failed"
	// 未知
	NAS_STATUS_UNKNOWN = "unknown"
	// 删除中
	NAS_STATUS_DELETING = "deleting"
)
