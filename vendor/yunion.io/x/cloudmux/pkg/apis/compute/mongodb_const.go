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
	MONGO_DB_STATUS_CREATING      = "creating"
	MONGO_DB_STATUS_RUNNING       = "running"
	MONGO_DB_STATUS_DEPLOY        = "deploy"
	MONGO_DB_STATUS_CHANGE_CONFIG = "change_config"
	MONGO_DB_STATUS_DELETING      = "deleting"
	MONGO_DB_STATUS_REBOOTING     = "rebooting"

	MONGO_DB_ENGINE_WIRED_TIGER = "WiredTiger"
	MONGO_DB_ENGINE_ROCKS       = "Rocks"

	// 分片
	MONGO_DB_CATEGORY_SHARDING = "sharding"
	// 副本集
	MONGO_DB_CATEGORY_REPLICATE = "replicate"
)
