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

package scheduler

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

type ScheduleBaseConfig struct {
	BestEffort      bool     `json:"best_effort"`
	SuggestionLimit int64    `json:"suggestion_limit"`
	SuggestionAll   bool     `json:"suggestion_all"`
	IgnoreFilters   []string `json:"ignore_filters"`
	SessionId       string   `json:"session_id"`

	// usedby test api
	RecordLog bool `json:"record_to_history"`
	Details   bool `json:"details"`
}

type ForGuest struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type GroupRelation struct {
	GroupId  string `json:"group_id"`
	Strategy string `json:"strategy"`
	Scope    string `json:"scope"`
}

type ServerConfig struct {
	*compute.ServerConfigs

	Memory      int    `json:"vmem_size"`
	Ncpu        int    `json:"vcpu_count"`
	Name        string `json:"name"`
	GuestStatus string `json:"guest_status"`
	Cdrom       string `json:"cdrom"`

	// owner project id
	Project string `json:"project_id"`
	// owner domain id
	Domain string `json:"domain_id"`

	// Deprecated
	Metadata       map[string]string `json:"__meta__"`
	ForGuests      []*ForGuest       `json:"for_guests"`
	GroupRelations []*GroupRelation  `json:"group_releations"`
	Groups         interface{}       `json:"groups"`
	Id             string            `json:"id"`
}

// ScheduleInput used by scheduler sync-schedule/test/forecast api
type ScheduleInput struct {
	apis.Meta

	ScheduleBaseConfig

	ServerConfig

	// HostId used by migrate
	HostId       string `json:"host_id"`
	LiveMigrate  bool   `json:"live_migrate"`
	CpuDesc      string `json:"cpu_desc"`
	CpuMicrocode string `json:"cpu_microcode"`
	CpuMode      string `json:"cpu_mode"`
	OsArch       string `json:"os_arch"`

	// In the migrate and create backup cases
	// we don't need reallocate network
	ReuseNetwork bool `json:"reuse_network"`

	// Change config
	ChangeConfig bool
	// guest who change config has isolated device
	HasIsolatedDevice bool

	PendingUsages []jsonutils.JSONObject
}

func (input ScheduleInput) ToConditionInput() *jsonutils.JSONDict {
	ret := input.JSON(input)
	// old condition compatible
	ret.Add(jsonutils.NewString(input.Project), "owner_tenant_id")
	return ret
}

type CandidateDisk struct {
	Index      int      `json:"index"`
	StorageIds []string `json:"storage_ids"`
}

type CandidateDiskV2 struct {
	Index    int                 `json:"index"`
	Storages []*CandidateStorage `json:"storages"`
}

type CandidateStorage struct {
	Id           string
	Name         string
	FreeCapacity int64
}

type CandidateNet struct {
	Index      int      `json:"index"`
	NetworkIds []string `json:"network_ids"`
}

type CandidateResource struct {
	SessionId string           `json:"session_id"`
	HostId    string           `json:"host_id"`
	Name      string           `json:"name"`
	Disks     []*CandidateDisk `json:"disks"`
	Nets      []*CandidateNet  `json:"nets"`

	// used by backup schedule
	BackupCandidate *CandidateResource `json:"backup_candidate"`

	// Error means no candidate found, include reasons
	Error string `json:"error"`
}

type ScheduleOutput struct {
	apis.Meta

	Candidates []*CandidateResource `json:"candidates"`
}
