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
	EXTERNAL_PROJECT_STATUS_AVAILABLE   = "available"   // 可用
	EXTERNAL_PROJECT_STATUS_UNAVAILABLE = "unavailable" // 不可用
	EXTERNAL_PROJECT_STATUS_CREATING    = "creating"    // 创建中
	EXTERNAL_PROJECT_STATUS_DELETING    = "deleting"    // 删除中
	EXTERNAL_PROJECT_STATUS_UNKNOWN     = "unknown"     // 未知
)

var (
	MANGER_EXTERNAL_PROJECT_PROVIDERS = []string{
		CLOUD_PROVIDER_AZURE,
	}
)
