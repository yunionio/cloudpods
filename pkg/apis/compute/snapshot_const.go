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
	// create by
	SNAPSHOT_MANUAL = "manual"
	SNAPSHOT_AUTO   = "auto"

	SNAPSHOT_CREATING    = "creating"
	SNAPSHOT_ROLLBACKING = "rollbacking"
	SNAPSHOT_FAILED      = "create_failed"
	SNAPSHOT_READY       = "ready"
	SNAPSHOT_DELETING    = "deleting"
	SNAPSHOT_UNKNOWN     = "unknown"

	SNAPSHOT_POLICY_CREATING      = "creating"
	SNAPSHOT_POLICY_READY         = "ready"
	SNAPSHOT_POLICY_CREATE_FAILED = "create_failed"
	SNAPSHOT_POLICY_UPDATING      = "updating"
	SNAPSHOT_POLICY_UPDATE_FAILED = "update_failed"
	SNAPSHOT_POLICY_UNKNOWN       = "unknown"
	SNAPSHOT_POLICY_DELETING      = "deleting"
	SNAPSHOT_POLICY_DELETE_FAILED = "delete_failed"

	SNAPSHOT_POLICY_DISK_CREATING      = "creating"
	SNAPSHOT_POLICY_DISK_CREATE_FAILED = "create_failed"
	SNAPSHOT_POLICY_DISK_READY         = "ready"
	SNAPSHOT_POLICY_DISK_DELETING      = "deleting"
	SNAPSHOT_POLICY_DISK_DELETE_FAILED = "delete_failed"

	SNAPSHOT_POLICY_CACHE_READY         = "ready"
	SNAPSHOT_POLICY_CACHE_UPDATE_FAILED = "updating"
	SNAPSHOT_POLICY_CACHE_UPDATING      = "update_failed"
	SNAPSHOT_POLICY_CACHE_UNKNOWN       = "unknown"
	SNAPSHOT_POLICY_CACHE_DELETING      = "deleting"
	SNAPSHOT_POLICY_CACHE_DELETE_FAILED = "delete_failed"
)
