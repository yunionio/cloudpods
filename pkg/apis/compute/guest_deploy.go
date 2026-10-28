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
	"time"

	"yunion.io/x/jsonutils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
)

type TDeployAction string

const (
	TDeployActionCreate  TDeployAction = "create"
	TDeployActionRebuild TDeployAction = "rebuild"
	TDeployActionDeploy  TDeployAction = "deploy"
)

// SGuestDeployTaskParam 是 GuestDeployTask / HAGuestDeployTask 的任务入参。
// ResetPassword 为 nil 时表示调用方未指定，部署逻辑按重置密码处理。
type SGuestDeployTaskParam struct {
	DeployAction TDeployAction `json:"deploy_action"`

	Keypair         string          `json:"keypair"`
	KeypairId       string          `json:"keypair_id"`
	DeletePublicKey string          `json:"delete_public_key"`
	DeleteKeypair   bool            `json:"__delete_keypair__,omitfalse"`
	ResetPassword   *bool           `json:"reset_password"`
	Password        string          `json:"password"`
	UserData        string          `json:"user_data"`
	LoginAccount    string          `json:"login_account"`
	Restart         bool            `json:"restart,omitfalse"`
	AutoStart       bool            `json:"auto_start,omitfalse"`
	DeployConfigs   []*DeployConfig `json:"deploy_configs"`
	DeployTelegraf  bool            `json:"deploy_telegraf,omitfalse"`
	EnableCloudInit bool            `json:"enable_cloud_init"`

	FakeCreateFromBmImport bool   `json:"fake_create_from_bm_import,omitfalse"`
	TargetHostId           string `json:"target_host_id"`

	PublicIpBw         int                        `json:"public_ip_bw,omitzero"`
	PublicIpChargeType billing_api.TNetChargeType `json:"public_ip_charge_type"`
}

// SGuestDeployConfigOnHost 是 region 下发给宿主机的部署参数。
type SGuestDeployConfigOnHost struct {
	Desc             jsonutils.JSONObject `json:"desc"`
	Deploys          []*DeployConfig      `json:"deploys"`
	EnableCloudInit  bool                 `json:"enable_cloud_init"`
	LoginAccount     string               `json:"login_account"`
	ResetPassword    *bool                `json:"reset_password"`
	Password         string               `json:"password"`
	PublicKey        string               `json:"public_key"`
	KeypairName      string               `json:"keypair_name"`
	DeletePublicKey  string               `json:"delete_public_key"`
	AdminPublicKey   string               `json:"admin_public_key"`
	ProjectPublicKey string               `json:"project_public_key"`
	Action           TDeployAction        `json:"action"`
	OnFinish         string               `json:"on_finish"`
	DeployTelegraf   bool                 `json:"deploy_telegraf"`
	TelegrafConf     string               `json:"telegraf_conf"`

	FakeCreateFromBmImport bool `json:"fake_create_from_bm_import"`
	DisableImageCache      bool `json:"disable_image_cache"`
}

// SDeployGuestFsDisk 是部署完成回调里的磁盘信息。
// 裸金属回写 disk_id、size、pci_path、dev；公有云回写 uuid 等云上磁盘字段。
type SDeployGuestFsDisk struct {
	DiskId  string `json:"disk_id"`
	Size    int    `json:"size"`
	PciPath string `json:"pci_path"`
	Dev     string `json:"dev"`

	DiskType          string                   `json:"disk_type"`
	Uuid              string                   `json:"uuid"`
	BillingType       billing_api.TBillingType `json:"billing_type"`
	FsFromat          string                   `json:"fs_fromat"`
	AutoDelete        bool                     `json:"auto_delete"`
	TemplateId        string                   `json:"template_id"`
	DiskFormat        string                   `json:"disk_format"`
	Path              string                   `json:"path"`
	Driver            string                   `json:"driver"`
	CacheMode         string                   `json:"cache_mode"`
	ExpiredAt         time.Time                `json:"expired_at"`
	StorageExternalId string                   `json:"storage_external_id"`
	Device            string                   `json:"device"`
	Metadata          map[string]string        `json:"metadata"`
}

// SDeployGuestFsResponse 是宿主机部署文件系统后返回给 region 的结果，
// 也是 GuestDeployTask 完成时 OnGuestDeployTaskDataReceived 的入参。
type SDeployGuestFsResponse struct {
	Os               string `json:"os"`
	Account          string `json:"account"`
	Key              string `json:"key"`
	Distro           string `json:"distro"`
	Version          string `json:"version"`
	Arch             string `json:"arch"`
	Language         string `json:"language"`
	TelegrafDeployed bool   `json:"telegraf_deployed"`
	CurrentVersion   string `json:"current_version"`

	Uuid      string               `json:"uuid"`
	Disks     []SDeployGuestFsDisk `json:"disks"`
	Metadata  map[string]string    `json:"metadata"`
	ExpiredAt time.Time            `json:"expired_at"`
	AutoRenew bool                 `json:"auto_renew"`
	HostIp    string               `json:"host_ip"`
}
