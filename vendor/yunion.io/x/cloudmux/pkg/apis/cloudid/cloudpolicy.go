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

package cloudid

const (
	CLOUD_POLICY_STATUS_AVAILABLE     = "available"
	CLOUD_POLICY_STATUS_SYNCING       = "syncing"
	CLOUD_POLICY_STATUS_SYNC_FAILE    = "sync_failed"
	CLOUD_POLICY_STATUS_DELETING      = "deleting"
	CLOUD_POLICY_STATUS_DELETE_FAILED = "delete_failed"

	CLOUD_POLICY_TYPE_SYSTEM = "system"
	CLOUD_POLICY_TYPE_CUSTOM = "custom"
)
