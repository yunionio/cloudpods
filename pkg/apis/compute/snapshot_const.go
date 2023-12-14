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

import (
	"yunion.io/x/cloudmux/pkg/apis/compute"
)

const (
	// create by
	SNAPSHOT_MANUAL = "manual"
	SNAPSHOT_AUTO   = "auto"

	SNAPSHOT_CREATING      = compute.SNAPSHOT_CREATING
	SNAPSHOT_ROLLBACKING   = compute.SNAPSHOT_ROLLBACKING
	SNAPSHOT_FAILED        = compute.SNAPSHOT_FAILED
	SNAPSHOT_READY         = compute.SNAPSHOT_READY
	SNAPSHOT_DELETE_FAILED = compute.SNAPSHOT_DELETE_FAILED
	SNAPSHOT_DELETING      = compute.SNAPSHOT_DELETING
	SNAPSHOT_UNKNOWN       = compute.SNAPSHOT_UNKNOWN

	INSTANCE_SNAPSHOT_READY         = compute.INSTANCE_SNAPSHOT_READY
	INSTANCE_SNAPSHOT_UNKNOWN       = "unknown"
	INSTANCE_SNAPSHOT_FAILED        = "instance_snapshot_create_failed"
	INSTANCE_SNAPSHOT_START_DELETE  = "instance_snapshot_start_delete"
	INSTANCE_SNAPSHOT_DELETE_FAILED = "instance_snapshot_delete_failed"
	INSTANCE_SNAPSHOT_RESET         = "instance_snapshot_reset"

	SNAPSHOT_EXIST     = "exist"
	SNAPSHOT_NOT_EXIST = "not_exist"
)
