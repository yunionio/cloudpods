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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SchedtagConfig struct {
	apis.Meta

	Id       string `json:"id"`
	Strategy string `json:"strategy"`
	Weight   int    `json:"weight"`
}

type NetworkConfig struct {
	apis.Meta

	Index    int    `json:"index"`
	Network  string `json:"network"`
	Wire     string `json:"wire"`
	Exit     bool   `json:"exit"`
	Private  bool   `json:"private"`
	Mac      string `json:"mac"`
	Address  string `json:"address"`
	Address6 string `json:"address6"`
	Driver   string `json:"driver"`
	BwLimit  int    `json:"bw_limit"`
	Vip      bool   `json:"vip"`
	Reserved bool   `json:"reserved"`
	NetType  string `json:"net_type"`

	RequireDesignatedIP bool `json:"require_designated_ip"`

	RequireTeaming bool `json:"require_teaming"`
	TryTeaming     bool `json:"try_teaming"`

	StandbyPortCount int `json:"standby_port_count"`
	StandbyAddrCount int `json:"standby_addr_count"`

	Project   string            `json:"project_id"`
	Domain    string            `json:"domain_id"`
	Ifname    string            `json:"ifname"`
	Schedtags []*SchedtagConfig `json:"schedtags"`
}

type DiskConfig struct {
	apis.Meta

	Index      int               `json:"index"` // -1 means not attach to any guests
	ImageId    string            `json:"image_id"`
	SnapshotId string            `json:"snapshot_id"`
	DiskType   string            `json:"disk_type"`
	Schedtags  []*SchedtagConfig `json:"schedtags"`

	SizeMb          int               `json:"size"`
	Fs              string            `json:"fs"`
	Format          string            `json:"format"`
	Driver          string            `json:"driver"`
	Cache           string            `json:"cache"`
	Mountpoint      string            `json:"mountpoint"`
	Backend         string            `json:"backend"` // stroageType
	Medium          string            `json:"medium"`
	ImageProperties map[string]string `json:"image_properties"`

	Storage string `json:"storage_id"`
	DiskId  string `json:"disk_id"`
}

type IsolatedDeviceConfig struct {
	Index   int    `json:"index"`
	Id      string `json:"id"`
	DevType string `json:"dev_type"`
	Model   string `json:"model"`
	Vendor  string `json:"vendor"`
}

type BaremetalDiskConfig struct {
	//Index int `json:"index"`
	// disk type
	Type string `json:"type"`
	// raid config
	Conf         string  `json:"conf"`
	Count        int64   `json:"count"`
	Range        []int64 `json:"range"`
	Splits       string  `json:"splits"`
	Size         []int64 `json:"size"`
	Adapter      *int    `json:"adapter,omitempty"`
	Driver       string  `json:"driver"`
	Cachedbadbbu *bool   `json:"cachedbadbbu,omitempty"`
	Strip        *int64  `json:"strip,omitempty"`
	RA           *bool   `json:"ra,omitempty"`
	WT           *bool   `json:"wt,omitempty"`
	Direct       *bool   `json:"direct,omitempty"`
}

type ServerConfigs struct {
	// prefer options
	PreferRegion     string `json:"prefer_region_id"`
	PreferZone       string `json:"prefer_zone_id"`
	PreferWire       string `json:"prefer_wire_id"`
	PreferHost       string `json:"prefer_host_id"`
	PreferBackupHost string `json:"prefer_backup_host"`

	Hypervisor string `json:"hypervisor"`
	// ResourceType "shared|prepaid|dedicated"`
	ResourceType string `json:"resource_type"`
	InstanceType string `json:"instance_type"`
	Project      string `json:"project_id"`
	Domain       string `json:"domain_id"`
	Backup       bool   `json:"backup"`
	Count        int    `json:"count"`

	Disks                []*DiskConfig           `json:"disks"`
	Networks             []*NetworkConfig        `json:"nets"`
	Schedtags            []*SchedtagConfig       `json:"schedtags"`
	IsolatedDevices      []*IsolatedDeviceConfig `json:"isolated_devices"`
	BaremetalDiskConfigs []*BaremetalDiskConfig  `json:"baremetal_disk_configs"`

	InstanceGroupIds []string `json:"groups"`

	// DEPRECATE
	Suggestion bool `json:"suggestion"`
}

func NewServerConfigs() *ServerConfigs {
	return &ServerConfigs{
		Disks:                make([]*DiskConfig, 0),
		Networks:             make([]*NetworkConfig, 0),
		Schedtags:            make([]*SchedtagConfig, 0),
		IsolatedDevices:      make([]*IsolatedDeviceConfig, 0),
		BaremetalDiskConfigs: make([]*BaremetalDiskConfig, 0),
		InstanceGroupIds:     make([]string, 0),
	}
}

type DeployConfig struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ServerCreateInput struct {
	apis.Meta

	*ServerConfigs

	Name         string `json:"name"`
	GenerateName string `json:"generate_name"`
	VmemSize     int    `json:"vmem_size"`
	VcpuCount    int    `json:"vcpu_count"`
	UserData     string `json:"user_data"`

	KeypairId          string          `json:"keypair_id"`
	Password           string          `json:"password"`
	Cdrom              string          `json:"cdrom"`
	Vga                string          `json:"vga"`
	Vdi                string          `json:"vdi"`
	Bios               string          `json:"bios"`
	Description        string          `json:"description"`
	BootOrder          string          `json:"boot_order"`
	EnableCloudInit    bool            `json:"enable_cloud_init"`
	ResetPassword      *bool           `json:"reset_password"`
	DisableDelete      *bool           `json:"disable_delete"`
	ShutdownBehavior   string          `json:"shutdown_behavior"`
	AutoStart          bool            `json:"auto_start"`
	DeployConfigs      []*DeployConfig `json:"deploy_configs"`
	IsSystem           bool            `json:"is_system"`
	Duration           string          `json:"duration"`
	AutoPrepaidRecycle bool            `json:"auto_prepaid_recycle,omitfalse"`
	SecgroupId         string          `json:"secgrp_id"`
	EipBw              int             `json:"eip_bw,omitzero"`
	EipChargeType      string          `json:"eip_charge_type,omitempty"`
	Eip                string          `json:"eip,omitempty"`
	InstanceSnapshotId string          `json:"instance_snapshot_id,omitempty"`
	Secgroups          []string        `json:"secgroups"`

	OsType string `json:"os_type"`
	// Fill by server
	OsProfile    jsonutils.JSONObject `json:"__os_profile__"`
	BillingType  string               `json:"billing_type"`
	BillingCycle string               `json:"billing_cycle"`

	// DEPRECATE or not used fields
	Baremetal bool `json:"baremetal"`

	// Used to store BaremetalConvertHypervisorTaskId
	ParentTaskId string `json:"__parent_task_id,omitempty"`
	// default stroage type if host is given
	DefaultStorageType string `json:"default_storage_type,omitempty"`

	// Guest Image ID
	GuestImageID string
}

type ServerCloneInput struct {
	apis.Meta

	Name      string `json:"name"`
	AutoStart bool   `json:"auto_start"`

	EipBw         int    `json:"eip_bw,omitzero"`
	EipChargeType string `json:"eip_charge_type,omitempty"`
	Eip           string `json:"eip,omitempty"`

	PreferHost string `json:"prefer_host_id"`
}

type ServerDeployInput struct {
	apis.Meta

	Id string

	Keypair       string          `json:"keypair"`
	DeleteKeypair *bool           `json:"__delete_keypair__"`
	DeployConfigs []*DeployConfig `json:"deploy_configs"`
	ResetPassword *bool           `json:"reset_password"`
	Password      string          `json:"password"`
	AutoStart     *bool           `json:"auto_start"`
}

type GuestBatchMigrateRequest struct {
	apis.Meta

	GuestIds   []string
	PreferHost string
}

type GuestBatchMigrateParams struct {
	Id          string
	LiveMigrate bool
	RescueMode  bool
	OldStatus   string
}

type HostLoginInfo struct {
	apis.Meta

	Username string `json:"username"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}
