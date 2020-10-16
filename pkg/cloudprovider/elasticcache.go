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

package cloudprovider

import "yunion.io/x/onecloud/pkg/util/billing"

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423019.html
// https://help.aliyun.com/document_detail/60873.html?spm=a2c4g.11174283.6.715.7412dce0qSYemb
type SCloudElasticCacheInput struct {
	RegionId         string                 // 地域
	InstanceType     string                 // 实例规格 redis.master.small.default
	CapacityGB       int64                  // 缓存容量 华为云此项参数必选
	InstanceName     string                 // 实例名称
	UserName         string                 // redis 用户名，可选
	Password         string                 // redis 用户密码，可选
	ZoneIds          []string               // 可用区， 可选
	ChargeType       string                 // 计费类型，可选
	NodeType         string                 // 节点类型，可选
	NetworkType      string                 // 网络类型 VPC|CLASSIC，可选
	VpcId            string                 // VPC ，可选
	NetworkId        string                 // 子网ID，可选
	Engine           string                 // Redis|Memcache
	EngineVersion    string                 // 版本类型
	PrivateIpAddress string                 // 指定新实例的内网IP地址。
	SecurityGroupIds []string               // 安全组ID
	EipId            string                 // 绑定弹性IP
	MaintainBegin    string                 // 维护时间窗开始时间，格式为HH:mm:ss
	MaintainEnd      string                 // 维护时间窗结束时间，格式为HH:mm:ss
	BC               *billing.SBillingCycle // 包年包月
	ProjectId        string
	Tags             map[string]string
}

type SCloudElasticCacheAccountInput struct {
	AccountName      string // 账号名称
	AccountPassword  string // 账号密码
	AccountPrivilege string // 账号权限
	Description      string // 账号描述
}

type SCloudElasticCacheAccountResetPasswordInput struct {
	NoPasswordAccess *bool   // 免密码访问
	NewPassword      string  // 新密码
	OldPassword      *string // 旧密码。required by huawei
}

type SCloudElasticCacheAccountUpdateInput struct {
	NoPasswordAccess *bool   // 免密码访问
	Password         *string // 新密码
	OldPassword      *string // 旧密码。required by huawei
	AccountPrivilege *string
	Description      *string
}

type SCloudElasticCacheBackupPolicyUpdateInput struct {
	BackupType            string // auto：自动备份 / manual：手动备份
	BackupReservedDays    int    // 1-7
	PreferredBackupPeriod string // Monday（周一） / Tuesday（周二） / Wednesday（周三） / Thursday（周四） / Friday（周五） / Saturday（周六） / Sunday（周日）
	PreferredBackupTime   string // 备份时间，格式：HH:mmZ-HH:mmZ
}

type SCloudElasticCacheFlushInstanceInput struct {
	Password string // root账号密码. requied by qcloud
}
