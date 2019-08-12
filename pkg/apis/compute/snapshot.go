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

import "yunion.io/x/onecloud/pkg/apis"

type SSnapshotCreateInput struct {
	apis.Meta

	Name      string `json:"name"`
	ProjectId string `json:"project_id"`
	DomainId  string `json:"domain_id"`

	DiskId        string `json:"disk_id"`
	StorageId     string `json:"storage_id"`
	CreatedBy     string `json:"created_by"`
	Location      string `json:"location"`
	Size          int    `json:"size"`
	DiskType      string `json:"disk_type"`
	CloudregionId string `json:"cloudregion_id"`
	OutOfChain    bool   `json:"out_of_chain"`
	ManagerId     string `json:"manager_id"`
}

type SSnapshotPolicyCreateInput struct {
	apis.Meta

	Name      string `json:"name"`
	ProjectId string `json:"project_id"`
	DomainId  string `json:"domain_id"`

	RetentionDays  int   `json:"retention_days"`
	RepeatWeekdays []int `json:"repeat_weekdays"`
	TimePoints     []int `json:"time_points"`
}

type SSnapshotPolicyCreateInternalInput struct {
	apis.Meta

	Name      string
	ProjectId string
	DomainId  string

	RetentionDays  int
	RepeatWeekdays uint8
	TimePoints     uint32
}
