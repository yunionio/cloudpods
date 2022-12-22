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
	MIRROR_JOB            = "__mirror_job_status"
	MIRROR_JOB_READY      = "ready"
	MIRROR_JOB_FAILED     = "failed"
	MIRROR_JOB_INPROGRESS = "inprogress"
	QUORUM_CHILD_INDEX    = "__quorum_child_index"

	DISK_CLONE_TASK_ID = "__disk_clone_task_id"

	SSH_PORT = "__ssh_port"
)

const BASE_INSTANCE_SNAPSHOT_ID = "__base_instance_snapshot_id"
