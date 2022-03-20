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

package host

type GuestMemorySnapshotRequest struct {
	InstanceSnapshotId string `json:"instance_snapshot_id"`
}

type GuestMemorySnapshotResponse struct {
	MemorySnapshotPath string `json:"memory_snapshot_path"`
	SizeMB             int64  `json:"size_mb"`
	Checksum           string `json:"checksum"`
}

type GuestMemorySnapshotDeleteRequest struct {
	InstanceSnapshotId string `json:"instance_snapshot_id"`
	Path               string `json:"path"`
}

type GuestMemorySnapshotResetRequest struct {
	InstanceSnapshotId string `json:"instance_snapshot_id"`
	Path               string `json:"path"`
	Checksum           string `json:"checksum"`
}
