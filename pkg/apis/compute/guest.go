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
	"yunion.io/x/onecloud/pkg/apis"
)

type Server struct {
	apis.SVirtualResourceBase
	apis.SExternalizedResourceBase
	apis.SBillingResourceBase

	VcpuCount int `json:"vcpu_count"`
	VmemSize  int `json:"vmem_size"`

	BootOrder string `json:"boot_order"`

	DisableDelete    bool   `json:"disable_delete"`
	ShutdownBehavior string `json:"shutdown_behavior"`

	KeypairId string `json:"keypair_id"`

	HostId       string `json:"host_id"`
	BackupHostId string `json:"backup_host_id"`

	Vga     string `json:"vga"`
	Vdi     string `json:"vdi"`
	Machine string `json:"machine"`
	Bios    string `json:"bios"`
	OsType  string `json:"os_type"`

	FlavorId string `json:"flavor_id"`

	SecgrpId      string `json:"secgrp_id"`
	AdminSecgrpId string `json:"admin_secgrp_id"`

	Hypervisor string `json:"hypervisor"`

	InstanceType string `json:"instance_type"`
}

type ServerListInput struct {
	apis.BaseListOptions
}

type ServerDeleteInput struct {
	apis.OverridePendingDelete
}

type ServerUpdateInput struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	ShutdownBehavior string `json:"shutdown_behavior"`
}

type ServerStopInput struct {
	// Force stop server
	IsForce bool `json:"is_force"`
}
