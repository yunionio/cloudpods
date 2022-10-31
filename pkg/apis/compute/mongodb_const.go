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
	MONGO_DB_STATUS_CREATING      = compute.MONGO_DB_STATUS_CREATING
	MONGO_DB_STATUS_RUNNING       = compute.MONGO_DB_STATUS_RUNNING
	MONGO_DB_STATUS_DEPLOY        = compute.MONGO_DB_STATUS_DEPLOY
	MONGO_DB_STATUS_CHANGE_CONFIG = compute.MONGO_DB_STATUS_CHANGE_CONFIG
	MONGO_DB_STATUS_DELETING      = compute.MONGO_DB_STATUS_DELETING
	MONGO_DB_STATUS_DELETE_FAILED = "delete_failed"
	MONGO_DB_STATUS_REBOOTING     = compute.MONGO_DB_STATUS_REBOOTING
	MONGO_DB_STATUS_UNKNOWN       = "unknown"

	MONGO_DB_ENGINE_WIRED_TIGER = compute.MONGO_DB_ENGINE_WIRED_TIGER
	MONGO_DB_ENGINE_ROCKS       = compute.MONGO_DB_ENGINE_ROCKS

	MONGO_DB_ENGINE_VERSION_40 = "4.0"
	MONGO_DB_ENGINE_VERSION_36 = "3.6"
	MONGO_DB_ENGINE_VERSION_32 = "3.2"

	// 分片
	MONGO_DB_CATEGORY_SHARDING = compute.MONGO_DB_CATEGORY_SHARDING
	// 副本集
	MONGO_DB_CATEGORY_REPLICATE = compute.MONGO_DB_CATEGORY_REPLICATE
)
