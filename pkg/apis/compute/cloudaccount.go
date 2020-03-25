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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type CloudaccountCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	// 指定云平台
	// Qcloud: 腾讯云
	// Ctyun: 天翼云
	// enum: VMware, Aliyun, Qcloud, Azure, Aws, Huawei, OpenStack, Ucloud, ZStack, Google, Ctyun
	Provider string `json:"provider"`
	// swagger:ignore
	AccountId string

	// 指定云平台品牌, 此参数默认和provider相同
	// requried: false
	//
	//
	//
	// | provider | 支持的参数 |
	// | -------- | ---------- |
	// | VMware | VMware |
	// | Aliyun | Aliyun |
	// | Qcloud | Qcloud |
	// | Azure | Azure |
	// | Aws | Aws |
	// | Huawei | Huawei |
	// | OpenStack | OpenStack |
	// | Ucloud | Ucloud |
	// | ZStack | ZStack, DStack |
	// | Google | Google |
	// | Ctyun | Ctyun |
	Brand string `json:"brand"`

	// swagger:ignore
	IsPublicCloud bool
	// swagger:ignore
	IsOnPremise bool

	// 指定云账号所属的项目
	Tenant string `json:"tenant"`

	// swagger:ignore
	TenantId string

	// 启用自动同步
	// default: false
	EnableAutoSync bool `json:"enable_auto_sync"`

	// 自动同步间隔时间
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// 自动根据云上项目或订阅创建本地项目
	// default: false
	AutoCreateProject bool `json:"auto_create_project"`

	// 额外信息,例如账单的access key
	Options *jsonutils.JSONObject `json:"options"`

	// 代理配置
	ProxySettingId string `json:"proxy_setting_id"`

	cloudprovider.SCloudaccount
	cloudprovider.SCloudaccountCredential
}

type CloudaccountShareModeInput struct {
	apis.Meta

	ShareMode string
}

func (i CloudaccountShareModeInput) Validate() error {
	if len(i.ShareMode) == 0 {
		return httperrors.NewMissingParameterError("share_mode")
	}
	if !utils.IsInStringArray(i.ShareMode, CLOUD_ACCOUNT_SHARE_MODES) {
		return httperrors.NewInputParameterError("invalid share_mode %s", i.ShareMode)
	}
	return nil
}

type CloudaccountListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	ManagedResourceListInput

	CapabilityListInput
}

type ProviderProject struct {
	// 子订阅项目名称
	// example: system
	Tenant string `json:"tenant"`

	// 子订阅项目Id
	// 9a48383a-467a-4542-8b50-4e15b0a8715f
	TenantId string `json:"tenant_id"`
}

type CloudaccountDetail struct {
	apis.StandaloneResourceDetails
	SCloudaccount

	// 子订阅项目信息
	Projects []ProviderProject `json:"projects"`

	// 同步时间间隔
	// example: 3600
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// 同步状态
	SyncStatus2 string `json:"sync_stauts2"`

	// 云账号环境类型
	// public: 公有云
	// private: 私有云
	// onpremise: 本地IDC
	// example: public
	CloudEnv string `json:"cloud_env"`

	// 云账号项目名称
	// example: system
	Tenant string `json:"tenant"`

	// 弹性公网Ip数量
	// example: 2
	EipCount int `json:"eip_count,allowempty"`

	// 虚拟私有网络数量
	// example: 4
	VpcCount int `json:"vpc_count,allowempty"`

	// 云盘数量
	// example: 12
	DiskCount int `json:"disk_count,allowempty"`

	// 宿主机数量(不计算虚拟机宿主机数量)
	// example: 0
	HostCount int `json:"host_count,allowempty"`

	// 云主机数量
	// example: 4
	GuestCount int `json:"guest_count,allowempty"`

	// 块存储数量
	// example: 12
	StorageCount int `json:"storage_count,allowempty"`

	// 子订阅数量
	// example: 1
	ProviderCount int `json:"provider_count,allowempty"`

	// 路由表数量
	// example: 0
	RoutetableCount int `json:"routetable_count,allowempty"`

	// 存储缓存数量
	// example: 10
	StoragecacheCount int `json:"storagecache_count,allowempty"`
}
