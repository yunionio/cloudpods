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

package models

import "time"

type ListRegionsReq struct {
}

type CloudregionDetails struct {
	CanDelete           bool              `json:"can_delete"`
	CanUpdate           bool              `json:"can_update"`
	City                string            `json:"city"`
	CloudEnv            string            `json:"cloud_env"`
	CountryCode         string            `json:"country_code"`
	CreatedAt           *time.Time        `json:"created_at"`
	Deleted             bool              `json:"deleted"`
	DeletedAt           *time.Time        `json:"deleted_at"`
	Description         string            `json:"description"`
	Enabled             bool              `json:"enabled"`
	Environment         string            `json:"environment"`
	ExternalId          string            `json:"external_id"`
	GuestCount          int64             `json:"guest_count"`
	GuestIncrementCount int64             `json:"guest_increment_count"`
	Id                  string            `json:"id"`
	ImportedAt          *time.Time        `json:"imported_at"`
	IsEmulated          bool              `json:"is_emulated"`
	Latitude            float64           `json:"latitude"`
	Longitude           float64           `json:"longitude"`
	Metadata            map[string]string `json:"metadata"`
	Name                string            `json:"name"`
	NetworkCount        int64             `json:"network_count"`
	Progress            float64           `json:"progress"`
	Provider            string            `json:"provider"`
	Source              string            `json:"source"`
	Status              string            `json:"status"`
	UpdateVersion       int64             `json:"update_version"`
	UpdatedAt           *time.Time        `json:"updated_at"`
	VpcCount            int64             `json:"vpc_count"`
	ZoneCount           int64             `json:"zone_count"`
}

type CloudregionListResponse struct {
	Limit        int64                `json:"limit"`
	Offset       int64                `json:"offset"`
	Cloudregions []CloudregionDetails `json:"cloudregions"`
	Total        int64                `json:"total"`
}

type SharedDomain struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SharedProject struct {
	Domain   string `json:"domain"`
	DomainId string `json:"domain_id"`
	Id       string `json:"id"`
	Name     string `json:"name"`
}

type VpcDetails struct {
	Account             string            `json:"account"`
	AccountHealthStatus string            `json:"account_health_status"`
	AccountId           string            `json:"account_id"`
	AccountReadOnly     bool              `json:"account_read_only"`
	AccountStatus       string            `json:"account_status"`
	AcceptVpcPeerCount  int64             `json:"accpet_vpc_peer_count"`
	Brand               string            `json:"brand"`
	CanDelete           bool              `json:"can_delete"`
	CanUpdate           bool              `json:"can_update"`
	CidrBlock           string            `json:"cidr_block"`
	CidrBlock6          string            `json:"cidr_block6"`
	CloudEnv            string            `json:"cloud_env"`
	Cloudregion         string            `json:"cloudregion"`
	CloudregionId       string            `json:"cloudregion_id"`
	CreatedAt           *time.Time        `json:"created_at"`
	Deleted             bool              `json:"deleted"`
	DeletedAt           *time.Time        `json:"deleted_at"`
	Description         string            `json:"description"`
	Direct              bool              `json:"direct"`
	DnsZoneCount        int64             `json:"dns_zone_count"`
	DomainId            string            `json:"domain_id"`
	DomainSrc           string            `json:"domain_src"`
	Enabled             bool              `json:"enabled"`
	Environment         string            `json:"environment"`
	ExternalAccessMode  string            `json:"external_access_mode"`
	ExternalId          string            `json:"external_id"`
	Globalvpc           string            `json:"globalvpc"`
	GlobalvpcId         string            `json:"globalvpc_id"`
	Id                  string            `json:"id"`
	ImportedAt          *time.Time        `json:"imported_at"`
	IsDefault           bool              `json:"is_default"`
	IsEmulated          bool              `json:"is_emulated"`
	IsPublic            bool              `json:"is_public"`
	Manager             string            `json:"manager"`
	ManagerDomain       string            `json:"manager_domain"`
	ManagerDomainId     string            `json:"manager_domain_id"`
	ManagerId           string            `json:"manager_id"`
	ManagerProject      string            `json:"manager_project"`
	ManagerProjectId    string            `json:"manager_project_id"`
	Metadata            map[string]string `json:"metadata"`
	Name                string            `json:"name"`
	NatgatewayCount     int64             `json:"natgateway_count"`
	NetworkCount        int64             `json:"network_count"`
	Progress            float64           `json:"progress"`
	ProjectDomain       string            `json:"project_domain"`
	Provider            string            `json:"provider"`
	PublicScope         string            `json:"public_scope"`
	PublicSrc           string            `json:"public_src"`
	Region              string            `json:"region"`
	RegionExtId         string            `json:"region_ext_id"`
	RegionExternalId    string            `json:"region_external_id"`
	RegionId            string            `json:"region_id"`
	RequestVpcPeerCount int64             `json:"request_vpc_peer_count"`
	RoutetableCount     int64             `json:"routetable_count"`
	SharedDomains       []SharedDomain    `json:"shared_domains"`
	SharedProjects      []SharedProject   `json:"shared_projects"`
	Source              string            `json:"source"`
	Status              string            `json:"status"`
	UpdateVersion       int64             `json:"update_version"`
	UpdatedAt           *time.Time        `json:"updated_at"`
	WireCount           int64             `json:"wire_count"`
}

type VpcListResponse struct {
	Limit  int64        `json:"limit"`
	Offset int64        `json:"offset"`
	Vpcs   []VpcDetails `json:"vpcs"`
	Total  int64        `json:"total"`
}

type SchedtagShortDescDetails struct {
	Default string `json:"default"`
	Id      string `json:"id"`
	Name    string `json:"name"`
	ResName string `json:"res_name"`
}

type SRoute []string

type SSimpleWire struct {
	Wire   string `json:"Wire"`
	WireId string `json:"WireId"`
}

type NetworkDetails struct {
	Account               string                     `json:"account"`
	AccountHealthStatus   string                     `json:"account_health_status"`
	AccountId             string                     `json:"account_id"`
	AccountReadOnly       bool                       `json:"account_read_only"`
	AccountStatus         string                     `json:"account_status"`
	AdditionalWires       []SSimpleWire              `json:"additional_wires"`
	AllocPolicy           string                     `json:"alloc_policy"`
	AllocTimoutSeconds    int64                      `json:"alloc_timout_seconds"`
	BgpType               string                     `json:"bgp_type"`
	BmReusedVnics         int64                      `json:"bm_reused_vnics"`
	BmVnics               int64                      `json:"bm_vnics"`
	Brand                 string                     `json:"brand"`
	CanDelete             bool                       `json:"can_delete"`
	CanUpdate             bool                       `json:"can_update"`
	CloudEnv              string                     `json:"cloud_env"`
	Cloudregion           string                     `json:"cloudregion"`
	CloudregionId         string                     `json:"cloudregion_id"`
	CreatedAt             *time.Time                 `json:"created_at"`
	Deleted               bool                       `json:"deleted"`
	DeletedAt             *time.Time                 `json:"deleted_at"`
	Description           string                     `json:"description"`
	Dns                   string                     `json:"dns"`
	DomainId              string                     `json:"domain_id"`
	EipVnics              int64                      `json:"eip_vnics"`
	Environment           string                     `json:"environment"`
	Exit                  bool                       `json:"exit"`
	ExternalId            string                     `json:"external_id"`
	Freezed               bool                       `json:"freezed"`
	GroupVnics            int64                      `json:"group_vnics"`
	GuestDhcp             string                     `json:"guest_dhcp"`
	GuestDns              string                     `json:"guest_dns"`
	GuestDns6             string                     `json:"guest_dns6"`
	GuestDomain           string                     `json:"guest_domain"`
	GuestDomain6          string                     `json:"guest_domain6"`
	GuestGateway          string                     `json:"guest_gateway"`
	GuestGateway6         string                     `json:"guest_gateway6"`
	GuestIpEnd            string                     `json:"guest_ip_end"`
	GuestIpMask           uint8                      `json:"guest_ip_mask"`
	GuestIpStart          string                     `json:"guest_ip_start"`
	GuestIp6End           string                     `json:"guest_ip6_end"`
	GuestIp6Mask          uint8                      `json:"guest_ip6_mask"`
	GuestIp6Start         string                     `json:"guest_ip6_start"`
	GuestNtp              string                     `json:"guest_ntp"`
	Id                    string                     `json:"id"`
	IfnameHint            string                     `json:"ifname_hint"`
	ImportedAt            *time.Time                 `json:"imported_at"`
	IsAutoAlloc           bool                       `json:"is_auto_alloc"`
	IsClassic             bool                       `json:"is_classic"`
	IsDefaultVpc          bool                       `json:"is_default_vpc"`
	IsEmulated            bool                       `json:"is_emulated"`
	IsPublic              bool                       `json:"is_public"`
	IsSystem              bool                       `json:"is_system"`
	LbVnics               int64                      `json:"lb_vnics"`
	Manager               string                     `json:"manager"`
	ManagerDomain         string                     `json:"manager_domain"`
	ManagerDomainId       string                     `json:"manager_domain_id"`
	ManagerId             string                     `json:"manager_id"`
	ManagerProject        string                     `json:"manager_project"`
	ManagerProjectId      string                     `json:"manager_project_id"`
	Metadata              map[string]string          `json:"metadata"`
	Name                  string                     `json:"name"`
	NatVnics              int64                      `json:"nat_vnics"`
	NetworkinterfaceVnics int64                      `json:"networkinterface_vnics"`
	PendingDeleted        bool                       `json:"pending_deleted"`
	PendingDeletedAt      *time.Time                 `json:"pending_deleted_at"`
	Ports                 int64                      `json:"ports"`
	PortsUsed             int64                      `json:"ports_used"`
	Ports6Used            int64                      `json:"ports6_used"`
	Progress              float64                    `json:"progress"`
	Project               string                     `json:"project"`
	ProjectDomain         string                     `json:"project_domain"`
	ProjectId             string                     `json:"project_id"`
	ProjectMetadata       map[string]string          `json:"project_metadata"`
	ProjectSrc            string                     `json:"project_src"`
	Provider              string                     `json:"provider"`
	PublicScope           string                     `json:"public_scope"`
	PublicSrc             string                     `json:"public_src"`
	RdsVnics              int64                      `json:"rds_vnics"`
	Region                string                     `json:"region"`
	RegionExtId           string                     `json:"region_ext_id"`
	RegionExternalId      string                     `json:"region_external_id"`
	RegionId              string                     `json:"region_id"`
	ReserveVnics4         int64                      `json:"reserve_vnics4"`
	ReserveVnics6         int64                      `json:"reserve_vnics6"`
	Routes                []SRoute                   `json:"routes"`
	Schedtags             []SchedtagShortDescDetails `json:"schedtags"`
	ServerType            string                     `json:"server_type"`
	SharedDomains         []SharedDomain             `json:"shared_domains"`
	SharedProjects        []SharedProject            `json:"shared_projects"`
	Source                string                     `json:"source"`
	Status                string                     `json:"status"`
	Tenant                string                     `json:"tenant"`
	TenantId              string                     `json:"tenant_id"`
	Total                 int64                      `json:"total"`
	Total6                int64                      `json:"total6"`
	UpdateVersion         int64                      `json:"update_version"`
	UpdatedAt             *time.Time                 `json:"updated_at"`
	VlanId                int64                      `json:"vlan_id"`
	Vnics                 int64                      `json:"vnics"`
	Vnics4                int64                      `json:"vnics4"`
	Vnics6                int64                      `json:"vnics6"`
	Vpc                   string                     `json:"vpc"`
	VpcExtId              string                     `json:"vpc_ext_id"`
	VpcId                 string                     `json:"vpc_id"`
	Wire                  string                     `json:"wire"`
	WireId                string                     `json:"wire_id"`
	Zone                  string                     `json:"zone"`
	ZoneId                string                     `json:"zone_id"`
}

type NetworkListResponse struct {
	Limit    int64            `json:"limit"`
	Offset   int64            `json:"offset"`
	Networks []NetworkDetails `json:"networks"`
	Total    int64            `json:"total"`
}

type ImageDetails struct {
	AutoDeleteAt           *time.Time        `json:"auto_delete_at"`
	CanDelete              bool              `json:"can_delete"`
	CanUpdate              bool              `json:"can_update"`
	Checksum               string            `json:"checksum"`
	CreatedAt              *time.Time        `json:"created_at"`
	Deleted                bool              `json:"deleted"`
	DeletedAt              *time.Time        `json:"deleted_at"`
	Description            string            `json:"description"`
	DisableDelete          bool              `json:"disable_delete"`
	DiskFormat             string            `json:"disk_format"`
	DomainId               string            `json:"domain_id"`
	EncryptAlg             string            `json:"encrypt_alg"`
	EncryptKey             string            `json:"encrypt_key"`
	EncryptKeyId           string            `json:"encrypt_key_id"`
	EncryptKeyUser         string            `json:"encrypt_key_user"`
	EncryptKeyUserDomain   string            `json:"encrypt_key_user_domain"`
	EncryptKeyUserDomainId string            `json:"encrypt_key_user_domain_id"`
	EncryptKeyUserId       string            `json:"encrypt_key_user_id"`
	EncryptStatus          string            `json:"encrypt_status"`
	FastHash               string            `json:"fast_hash"`
	Freezed                bool              `json:"freezed"`
	Id                     string            `json:"id"`
	IsData                 bool              `json:"is_data"`
	IsEmulated             bool              `json:"is_emulated"`
	IsGuestImage           bool              `json:"is_guest_image"`
	IsPublic               bool              `json:"is_public"`
	IsStandard             bool              `json:"is_standard"`
	IsSystem               bool              `json:"is_system"`
	Location               string            `json:"location"`
	Metadata               map[string]string `json:"metadata"`
	MinDisk                int32             `json:"min_disk"`
	MinRam                 int32             `json:"min_ram"`
	Name                   string            `json:"name"`
	OsArch                 string            `json:"os_arch"`
	OssChecksum            string            `json:"oss_checksum"`
	Owner                  string            `json:"owner"`
	PendingDeleted         bool              `json:"pending_deleted"`
	PendingDeletedAt       *time.Time        `json:"pending_deleted_at"`
	Progress               float64           `json:"progress"`
	Project                string            `json:"project"`
	ProjectDomain          string            `json:"project_domain"`
	ProjectId              string            `json:"project_id"`
	ProjectMetadata        map[string]string `json:"project_metadata"`
	ProjectSrc             string            `json:"project_src"`
	Properties             map[string]string `json:"properties"`
	Protected              bool              `json:"protected"`
	PublicScope            string            `json:"public_scope"`
	PublicSrc              string            `json:"public_src"`
	SharedDomains          []SharedDomain    `json:"shared_domains"`
	SharedProjects         []SharedProject   `json:"shared_projects"`
	Size                   int64             `json:"size"`
	Status                 string            `json:"status"`
	Tenant                 string            `json:"tenant"`
	TenantId               string            `json:"tenant_id"`
	UpdateVersion          int64             `json:"update_version"`
	UpdatedAt              *time.Time        `json:"updated_at"`
}

type ImageListResponse struct {
	Limit  int64          `json:"limit"`
	Offset int64          `json:"offset"`
	Images []ImageDetails `json:"images"`
	Total  int64          `json:"total"`
}

type ServerSkuDetails struct {
	AttachedDiskCount    int64             `json:"attached_disk_count"`
	AttachedDiskSizeGB   int64             `json:"attached_disk_size_gb"`
	AttachedDiskType     string            `json:"attached_disk_type"`
	CanDelete            bool              `json:"can_delete"`
	CanUpdate            bool              `json:"can_update"`
	CloudEnv             string            `json:"cloud_env"`
	Cloudregion          string            `json:"cloudregion"`
	CloudregionId        string            `json:"cloudregion_id"`
	CpuArch              string            `json:"cpu_arch"`
	CpuCoreCount         int64             `json:"cpu_core_count"`
	CreatedAt            *time.Time        `json:"created_at"`
	DataDiskMaxCount     int64             `json:"data_disk_max_count"`
	DataDiskTypes        string            `json:"data_disk_types"`
	Deleted              bool              `json:"deleted"`
	DeletedAt            *time.Time        `json:"deleted_at"`
	Description          string            `json:"description"`
	Enabled              bool              `json:"enabled"`
	ExternalId           string            `json:"external_id"`
	GpuAttachable        bool              `json:"gpu_attachable"`
	GpuCount             string            `json:"gpu_count"`
	GpuMaxCount          int64             `json:"gpu_max_count"`
	GpuSpec              string            `json:"gpu_spec"`
	Id                   string            `json:"id"`
	ImportedAt           *time.Time        `json:"imported_at"`
	InstanceTypeCategory string            `json:"instance_type_category"`
	InstanceTypeFamily   string            `json:"instance_type_family"`
	IsEmulated           bool              `json:"is_emulated"`
	LocalCategory        string            `json:"local_category"`
	Md5                  string            `json:"md5"`
	MemorySizeMB         int64             `json:"memory_size_mb"`
	Metadata             map[string]string `json:"metadata"`
	Name                 string            `json:"name"`
	NicMaxCount          int64             `json:"nic_max_count"`
	NicType              string            `json:"nic_type"`
	OsName               string            `json:"os_name"`
	PostpaidStatus       string            `json:"postpaid_status"`
	PrepaidStatus        string            `json:"prepaid_status"`
	Progress             float64           `json:"progress"`
	Provider             string            `json:"provider"`
	Region               string            `json:"region"`
	RegionExtId          string            `json:"region_ext_id"`
	RegionExternalId     string            `json:"region_external_id"`
	RegionId             string            `json:"region_id"`
	Source               string            `json:"source"`
	Status               string            `json:"status"`
	SysDiskMaxSizeGB     int64             `json:"sys_disk_max_size_gb"`
	SysDiskMinSizeGB     int64             `json:"sys_disk_min_size_gb"`
	SysDiskResizable     bool              `json:"sys_disk_resizable"`
	SysDiskType          string            `json:"sys_disk_type"`
	TotalGuestCount      int64             `json:"total_guest_count"`
	UpdateVersion        int64             `json:"update_version"`
	UpdatedAt            *time.Time        `json:"updated_at"`
	Zone                 string            `json:"zone"`
	ZoneExtId            string            `json:"zone_ext_id"`
	ZoneId               string            `json:"zone_id"`
}

type ServerSkuListResponse struct {
	Limit      int64              `json:"limit"`
	Offset     int64              `json:"offset"`
	Serverskus []ServerSkuDetails `json:"serverskus"`
	Total      int64              `json:"total"`
}

type StorageHost struct {
	HostStatus string `json:"HostStatus"`
	Id         string `json:"Id"`
	Name       string `json:"Name"`
	Status     string `json:"Status"`
}

type StorageDetails struct {
	DiskCount            int64                      `json:"DiskCount"`
	HostCount            int64                      `json:"HostCount"`
	SnapshotCount        int64                      `json:"SnapshotCount"`
	Used                 int64                      `json:"Used"`
	Wasted               int64                      `json:"Wasted"`
	Account              string                     `json:"account"`
	AccountHealthStatus  string                     `json:"account_health_status"`
	AccountId            string                     `json:"account_id"`
	AccountReadOnly      bool                       `json:"account_read_only"`
	AccountStatus        string                     `json:"account_status"`
	ActualCapacityUsed   int64                      `json:"actual_capacity_used"`
	Brand                string                     `json:"brand"`
	CanDelete            bool                       `json:"can_delete"`
	CanUpdate            bool                       `json:"can_update"`
	Capacity             int64                      `json:"capacity"`
	CloudEnv             string                     `json:"cloud_env"`
	Cloudregion          string                     `json:"cloudregion"`
	CloudregionId        string                     `json:"cloudregion_id"`
	Cmtbound             float64                    `json:"cmtbound"`
	CommitBound          float64                    `json:"commit_bound"`
	CommitRate           float64                    `json:"commit_rate"`
	CreatedAt            *time.Time                 `json:"created_at"`
	Deleted              bool                       `json:"deleted"`
	DeletedAt            *time.Time                 `json:"deleted_at"`
	Description          string                     `json:"description"`
	DomainId             string                     `json:"domain_id"`
	DomainSrc            string                     `json:"domain_src"`
	Enabled              bool                       `json:"enabled"`
	Environment          string                     `json:"environment"`
	ExternalId           string                     `json:"external_id"`
	FreeCapacity         int64                      `json:"free_capacity"`
	Hosts                []StorageHost              `json:"hosts"`
	Id                   string                     `json:"id"`
	ImportedAt           *time.Time                 `json:"imported_at"`
	IsEmulated           bool                       `json:"is_emulated"`
	IsPublic             bool                       `json:"is_public"`
	IsSysDiskStore       bool                       `json:"is_sys_disk_store"`
	Manager              string                     `json:"manager"`
	ManagerDomain        string                     `json:"manager_domain"`
	ManagerDomainId      string                     `json:"manager_domain_id"`
	ManagerId            string                     `json:"manager_id"`
	ManagerProject       string                     `json:"manager_project"`
	ManagerProjectId     string                     `json:"manager_project_id"`
	MasterHost           string                     `json:"master_host"`
	MasterHostName       string                     `json:"master_host_name"`
	MediumType           string                     `json:"medium_type"`
	Metadata             map[string]string          `json:"metadata"`
	Name                 string                     `json:"name"`
	Progress             float64                    `json:"progress"`
	ProjectDomain        string                     `json:"project_domain"`
	Provider             string                     `json:"provider"`
	PublicScope          string                     `json:"public_scope"`
	PublicSrc            string                     `json:"public_src"`
	RealTimeUsedCapacity int64                      `json:"real_time_used_capacity"`
	Region               string                     `json:"region"`
	RegionExtId          string                     `json:"region_ext_id"`
	RegionExternalId     string                     `json:"region_external_id"`
	RegionId             string                     `json:"region_id"`
	Reserved             int64                      `json:"reserved"`
	Schedtags            []SchedtagShortDescDetails `json:"schedtags"`
	SharedDomains        []SharedDomain             `json:"shared_domains"`
	SharedProjects       []SharedProject            `json:"shared_projects"`
	Source               string                     `json:"source"`
	Status               string                     `json:"status"`
	StorageConf          map[string]interface{}     `json:"storage_conf"`
	StorageType          string                     `json:"storage_type"`
	StoragecacheId       string                     `json:"storagecache_id"`
	UpdateVersion        int64                      `json:"update_version"`
	UpdatedAt            *time.Time                 `json:"updated_at"`
	UsedCapacity         int64                      `json:"used_capacity"`
	VirtualCapacity      int64                      `json:"virtual_capacity"`
	WasteCapacity        int64                      `json:"waste_capacity"`
	Zone                 string                     `json:"zone"`
	ZoneExtId            string                     `json:"zone_ext_id"`
	ZoneId               string                     `json:"zone_id"`
}

type StorageListResponse struct {
	Limit    int64            `json:"limit"`
	Offset   int64            `json:"offset"`
	Storages []StorageDetails `json:"storages"`
	Total    int64            `json:"total"`
}

type ServerDetails struct {
	Account                string                     `json:"account"`
	AccountHealthStatus    string                     `json:"account_health_status"`
	AccountId              string                     `json:"account_id"`
	AccountReadOnly        bool                       `json:"account_read_only"`
	AccountStatus          string                     `json:"account_status"`
	BackupGuestSync        string                     `json:"backup_guest_sync"`
	BackupGuestSyncStatus  string                     `json:"backup_guest_sync_status"`
	BackupHostId           string                     `json:"backup_host_id"`
	BackupHostName         string                     `json:"backup_host_name"`
	BackupHostStatus       string                     `json:"backup_host_status"`
	BillingCycle           string                     `json:"billing_cycle"`
	BillingType            string                     `json:"billing_type"`
	Bios                   string                     `json:"bios"`
	BootOrder              string                     `json:"boot_order"`
	Brand                  string                     `json:"brand"`
	CanDelete              bool                       `json:"can_delete"`
	CanRecycle             bool                       `json:"can_recycle"`
	CanUpdate              bool                       `json:"can_update"`
	Cdrom                  interface{}                `json:"cdrom"`
	CdromSupport           bool                       `json:"cdrom_support"`
	CloudEnv               string                     `json:"cloud_env"`
	Cloudregion            string                     `json:"cloudregion"`
	CloudregionId          string                     `json:"cloudregion_id"`
	Containers             interface{}                `json:"containers"`
	CpuNumaPin             map[string]interface{}     `json:"cpu_numa_pin"`
	CpuSockets             int64                      `json:"cpu_sockets"`
	CreatedAt              *time.Time                 `json:"created_at"`
	DeleteFailReason       interface{}                `json:"delete_fail_reason"`
	Deleted                bool                       `json:"deleted"`
	DeletedAt              *time.Time                 `json:"deleted_at"`
	Description            string                     `json:"description"`
	DisableDelete          bool                       `json:"disable_delete"`
	DiskSizeMb             int64                      `json:"disk"`
	DiskCount              int64                      `json:"disk_count"`
	Disks                  string                     `json:"disks"`
	DisksInfo              interface{}                `json:"disks_info"`
	DomainId               string                     `json:"domain_id"`
	Eip                    string                     `json:"eip"`
	EipMode                string                     `json:"eip_mode"`
	EncryptAlg             string                     `json:"encrypt_alg"`
	EncryptKey             string                     `json:"encrypt_key"`
	EncryptKeyId           string                     `json:"encrypt_key_id"`
	EncryptKeyUser         string                     `json:"encrypt_key_user"`
	EncryptKeyUserDomain   string                     `json:"encrypt_key_user_domain"`
	EncryptKeyUserDomainId string                     `json:"encrypt_key_user_domain_id"`
	EncryptKeyUserId       string                     `json:"encrypt_key_user_id"`
	Environment            string                     `json:"environment"`
	ExpiredAt              *time.Time                 `json:"expired_at"`
	ExternalId             string                     `json:"external_id"`
	ExtraCpuCount          int64                      `json:"extra_cpu_count"`
	FlavorId               string                     `json:"flavor_id"`
	Floppy                 interface{}                `json:"floppy"`
	FloppySupport          bool                       `json:"floppy_support"`
	Freezed                bool                       `json:"freezed"`
	GpuCount               string                     `json:"gpu_count"`
	GpuModel               string                     `json:"gpu_model"`
	Host                   string                     `json:"host"`
	HostAccessIp           string                     `json:"host_access_ip"`
	HostAccessMac          string                     `json:"host_access_mac"`
	HostBillingType        string                     `json:"host_billing_type"`
	HostEIP                string                     `json:"host_eip"`
	HostEnabled            bool                       `json:"host_enabled"`
	HostId                 string                     `json:"host_id"`
	HostStatus             string                     `json:"host_status"`
	Hostname               string                     `json:"hostname"`
	Hypervisor             string                     `json:"hypervisor"`
	Id                     string                     `json:"id"`
	ImportedAt             *time.Time                 `json:"imported_at"`
	Ips                    []string                   `json:"ips"`
	IsBaremetal            bool                       `json:"is_baremetal"`
	IsDefer                bool                       `json:"is_defer"`
	IsEmulated             bool                       `json:"is_emulated"`
	IsMerge                bool                       `json:"is_merge"`
	IsMirror               bool                       `json:"is_mirror"`
	IsPublic               bool                       `json:"is_public"`
	IsSystem               bool                       `json:"is_system"`
	KeypairId              string                     `json:"keypair_id"`
	Manager                string                     `json:"manager"`
	ManagerDomain          string                     `json:"manager_domain"`
	ManagerDomainId        string                     `json:"manager_domain_id"`
	ManagerId              string                     `json:"manager_id"`
	ManagerProject         string                     `json:"manager_project"`
	ManagerProjectId       string                     `json:"manager_project_id"`
	MemoryPinned           bool                       `json:"memory_pinned"`
	Metadata               map[string]string          `json:"metadata"`
	Mmemc                  interface{}                `json:"mmemc"`
	Name                   string                     `json:"name"`
	NicType                string                     `json:"nic_type"`
	Nics                   interface{}                `json:"nics"`
	NSPSConfig             map[string]interface{}     `json:"nsps_config"`
	OsArch                 string                     `json:"os_arch"`
	OsFullName             string                     `json:"os_full_name"`
	OsName                 string                     `json:"os_name"`
	OsType                 string                     `json:"os_type"`
	PendingDeleted         bool                       `json:"pending_deleted"`
	PendingDeletedAt       *time.Time                 `json:"pending_deleted_at"`
	PowerStates            string                     `json:"power_states"`
	Progress               float64                    `json:"progress"`
	Project                string                     `json:"project"`
	ProjectDomain          string                     `json:"project_domain"`
	ProjectId              string                     `json:"project_id"`
	ProjectMetadata        map[string]string          `json:"project_metadata"`
	ProjectSrc             string                     `json:"project_src"`
	Provider               string                     `json:"provider"`
	PublicIp               string                     `json:"public_ip"`
	PublicScope            string                     `json:"public_scope"`
	PublicSrc              string                     `json:"public_src"`
	Rds                    bool                       `json:"rds"`
	RecoveryMode           string                     `json:"recovery_mode"`
	ReorderMaster          bool                       `json:"reorder_master"`
	Schedtags              []SchedtagShortDescDetails `json:"schedtags"`
	SecurityGroup          string                     `json:"security_group"`
	SecurityGroupId        string                     `json:"security_group_id"`
	SecurityGroups         interface{}                `json:"security_groups"`
	SharedDomains          []SharedDomain             `json:"shared_domains"`
	SharedProjects         []SharedProject            `json:"shared_projects"`
	ShutdownBehavior       string                     `json:"shutdown_behavior"`
	SourceOsDist           string                     `json:"source_os_dist"`
	Source                 string                     `json:"source"`
	Status                 string                     `json:"status"`
	StorageId              string                     `json:"storage_id"`
	StorageType            string                     `json:"storage_type"`
	SystemVmtypeName       string                     `json:"system_vmtype_name"`
	Tenant                 string                     `json:"tenant"`
	TenantId               string                     `json:"tenant_id"`
	UpdateVersion          int64                      `json:"update_version"`
	UpdatedAt              *time.Time                 `json:"updated_at"`
	UpgradeStatus          string                     `json:"upgrade_status"`
	UpdateFailReason       interface{}                `json:"update_fail_reason"`
	UserData               string                     `json:"user_data"`
	VcpuCount              int64                      `json:"vcpu_count"`
	VdiBrokerStuff         map[string]interface{}     `json:"vdi_broker_stuff"`
	VdiConfig              map[string]interface{}     `json:"vdi_config"`
	VditConfig             map[string]interface{}     `json:"vdit_config"`
	VmemSize               int64                      `json:"vmem_size"`
	VMEMSizeMb             int64                      `json:"vmem_size_mb"`
	Vpc                    string                     `json:"vpc"`
	VpcId                  string                     `json:"vpc_id"`
	Zone                   string                     `json:"zone"`
	ZoneId                 string                     `json:"zone_id"`
}

type ServerListResponse struct {
	Limit   int64           `json:"limit"`
	Offset  int64           `json:"offset"`
	Servers []ServerDetails `json:"servers"`
	Total   int64           `json:"total"`
}

type ServerStartRequest struct {
	AutoPrepaid bool
	QemuVersion string
}

type ServerStopRequest struct {
	IsForce      bool
	StopCharging bool
	TimeoutSecs  int64
}

type ServerRestartRequest struct {
	IsForce bool
}

type ServerOperationResponse struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	TaskId    string `json:"task_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Operation string `json:"operation,omitempty"`
}

type ServerResetPasswordRequest struct {
	Password      string
	ResetPassword bool
	AutoStart     bool
	Username      string
}

type ServerDeleteRequest struct {
	OverridePendingDelete bool
	Purge                 bool
	DeleteSnapshots       bool
	DeleteEip             bool
	DeleteDisks           bool
}

type CreateServerRequest struct {
	Name          string
	VcpuCount     int64
	VmemSize      int64
	ImageId       string
	DiskSize      int64
	NetworkId     string
	ServerskuId   string
	Count         int
	Password      string
	AutoStart     bool
	BillingType   string
	Duration      string
	Description   string
	Hostname      string
	Hypervisor    string
	Metadata      map[string]string
	SecgroupId    string
	Secgroups     []string
	UserData      string
	KeypairId     string
	ProjectId     string
	ZoneId        string
	RegionId      string
	DisableDelete bool
	BootOrder     string
	DataDisks     []DiskConfig
}

type DiskConfig struct {
	ImageId  string
	Size     int64
	DiskType string
}

type ServerCreateResponseData struct {
	Servers []ServerCreateInfo `json:"servers"`
}

type ServerCreateInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	TaskID string `json:"task_id"`
}

type CreateServerResponse struct {
	Status  int                      `json:"status"`
	Message string                   `json:"msg"`
	Data    ServerCreateResponseData `json:"data"`
}

type MonitorResponse struct {
	Status int                 `json:"status"`
	Data   MonitorResponseData `json:"data"`
}

type MonitorResponseData struct {
	Metrics []MetricData `json:"metrics"`
}

type MetricData struct {
	Metric string        `json:"metric"`
	Unit   string        `json:"unit"`
	Values []MetricValue `json:"values"`
}

type MetricValue struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type ServerStatsResponse struct {
	Status int             `json:"status"`
	Data   ServerStatsData `json:"data"`
}

type ServerStatsData struct {
	CPUUsage  float64 `json:"cpu_usage"`
	MemUsage  float64 `json:"mem_usage"`
	DiskUsage float64 `json:"disk_usage"`
	NetBpsRx  int64   `json:"net_bps_rx"`
	NetBpsTx  int64   `json:"net_bps_tx"`
	UpdatedAt string  `json:"updated_at"`
}
