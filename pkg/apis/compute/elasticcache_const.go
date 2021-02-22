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

const (
	ELASTIC_CACHE_STATUS_RUNNING               = "running"               //（正常）
	ELASTIC_CACHE_STATUS_RESTARTING            = "restarting"            //（重启中）
	ELASTIC_CACHE_STATUS_RESTART_FAILED        = "restart_failed"        //（重启失败）
	ELASTIC_CACHE_STATUS_DEPLOYING             = "deploying"             //（创建中）
	ELASTIC_CACHE_STATUS_CREATE_FAILED         = "create_failed"         //（创建失败）
	ELASTIC_CACHE_STATUS_CHANGING              = "changing"              //（修改中）
	ELASTIC_CACHE_STATUS_CHANGE_FAILED         = "change_failed"         //（修改失败）
	ELASTIC_CACHE_STATUS_INACTIVE              = "inactive"              //（被禁用）
	ELASTIC_CACHE_STATUS_FLUSHING              = "flushing"              //（清除中）
	ELASTIC_CACHE_STATUS_FLUSHING_FAILED       = "flushing_failed"       //（清除失败）
	ELASTIC_CACHE_STATUS_RELEASING             = "releasing"             //（释放中）
	ELASTIC_CACHE_STATUS_RELEASED              = "released"              //（已释放）
	ELASTIC_CACHE_STATUS_RELEASE_FAILED        = "release_failed"        //（释放失败）
	ELASTIC_CACHE_STATUS_TRANSFORMING          = "transforming"          //（转换中）
	ELASTIC_CACHE_STATUS_UNAVAILABLE           = "unavailable"           //（服务停止）
	ELASTIC_CACHE_STATUS_ERROR                 = "error"                 //（删除失败）
	ELASTIC_CACHE_STATUS_MIGRATING             = "migrating"             //（迁移中）
	ELASTIC_CACHE_STATUS_BACKUPRECOVERING      = "backuprecovering"      //（备份恢复中）
	ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING = "minorversionupgrading" //（小版本升级中）
	ELASTIC_CACHE_STATUS_NETWORKMODIFYING      = "networkmodifying"      //（网络变更中）
	ELASTIC_CACHE_STATUS_SSLMODIFYING          = "sslmodifying"          //（SSL变更中）
	ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING = "majorversionupgrading" //（大版本升级中，可正常访问）
	ELASTIC_CACHE_STATUS_UNKNOWN               = "unknown"               //（未知状态）
	ELASTIC_CACHE_STATUS_DELETING              = "deleting"              // (删除)
	ELASTIC_CACHE_STATUS_SNAPSHOTTING          = "snapshotting"          //（快照）
	ELASTIC_CACHE_STATUS_SYNCING               = "syncing"               //（同步中）
	ELASTIC_CACHE_STATUS_SYNC_FAILED           = "sync_failed"           //（同步失败）
	ELASTIC_CACHE_RENEWING                     = "renewing"              //（续费中）
	ELASTIC_CACHE_RENEW_FAILED                 = "renew_failed"          //（续费失败）
	ELASTIC_CACHE_SET_AUTO_RENEW               = "set_auto_renew"        //（设置自动续费）
	ELASTIC_CACHE_SET_AUTO_RENEW_FAILED        = "set_auto_renew_failed" //（设置自动续费失败）

)

const (
	ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE     = "available"     // 正常可用
	ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE   = "unavailable"   // 不可用
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATING      = "creating"      // 创建中
	ELASTIC_CACHE_ACCOUNT_STATUS_MODIFYING     = "modifying"     // 修改中
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATE_FAILED = "create_failed" //（创建失败）
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETING      = "deleting"      // 删除中
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETE_FAILED = "delete_failed" // 删除失败
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETED       = "deleted"       // 已删除
)

const (
	ELASTIC_CACHE_UPDATE_TAGS        = "update_tags"
	ELASTIC_CACHE_UPDATE_TAGS_FAILED = "update_tags_fail"
)

const (
	ELASTIC_CACHE_ACCOUNT_TYPE_NORMAL = "normal" // 普通账号
	ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN  = "admin"  // 管理账号
)

const (
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ  = "read"  // 只读
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE = "write" // 读写
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_REPL  = "repl"  // 复制，复制权限支持读写，且支持使用SYNC/PSYNC命令。
)

const (
	ELASTIC_CACHE_BACKUP_STATUS_CREATING       = "creating" // 备份中
	ELASTIC_CACHE_BACKUP_STATUS_RESTORING      = "restoring"
	ELASTIC_CACHE_BACKUP_STATUS_COPYING        = "copying"
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED = "expired"  //（备份文件已过期）
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED = "deleted"  //（备份文件已删除）
	ELASTIC_CACHE_BACKUP_STATUS_DELETING       = "deleting" // 删除中
	ELASTIC_CACHE_BACKUP_STATUS_SUCCESS        = "success"  // 备份成功
	ELASTIC_CACHE_BACKUP_STATUS_FAILED         = "failed"   // 备份失败
	ELASTIC_CACHE_BACKUP_STATUS_UNKNOWN        = "unknown"  // 未知
)

const (
	ELASTIC_CACHE_BACKUP_TYPE_FULL        = "full"        // 全量备份
	ELASTIC_CACHE_BACKUP_TYPE_INCREMENTAL = "incremental" // 增量备份
)

const (
	ELASTIC_CACHE_BACKUP_MODE_AUTOMATED = "automated" // 自动备份
	ELASTIC_CACHE_BACKUP_MODE_MANUAL    = "manual"    // 手动触发备份
)

const (
	ELASTIC_CACHE_ACL_STATUS_AVAILABLE     = "available"     // 正常可用
	ELASTIC_CACHE_ACL_STATUS_CREATING      = "creating"      // 创建中
	ELASTIC_CACHE_ACL_STATUS_CREATE_FAILED = "create_failed" //（创建失败）
	ELASTIC_CACHE_ACL_STATUS_DELETING      = "deleting"      // 删除中
	ELASTIC_CACHE_ACL_STATUS_DELETE_FAILED = "delete_failed" // 删除失败
	ELASTIC_CACHE_ACL_STATUS_UPDATING      = "updating"      // 更新中
	ELASTIC_CACHE_ACL_STATUS_UPDATE_FAILED = "update_failed" // 更新失败
)

const (
	ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE     = "available"     // 正常可用
	ELASTIC_CACHE_PARAMETER_STATUS_UPDATING      = "updating"      // 更新中
	ELASTIC_CACHE_PARAMETER_STATUS_UPDATE_FAILED = "update_failed" // 更新失败
)

const (
	ELASTIC_CACHE_ARCH_TYPE_SINGLE  = "single"  // 单副本
	ELASTIC_CACHE_ARCH_TYPE_MASTER  = "master"  // 主备
	ELASTIC_CACHE_ARCH_TYPE_CLUSTER = "cluster" // 集群
	ELASTIC_CACHE_ARCH_TYPE_RWSPLIT = "rwsplit" // 读写分离
)

const (
	ELASTIC_CACHE_NODE_TYPE_SINGLE = "single"
	ELASTIC_CACHE_NODE_TYPE_DOUBLE = "double"
	ELASTIC_CACHE_NODE_TYPE_THREE  = "three"
	ELASTIC_CACHE_NODE_TYPE_FOUR   = "four"
	ELASTIC_CACHE_NODE_TYPE_FIVE   = "five"
	ELASTIC_CACHE_NODE_TYPE_SIX    = "six"
)

type ElasticcacheListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput
	VpcFilterListInput
	ZonalFilterListBase

	// 实例规格
	// example: redis.master.micro.default
	InstanceType []string `json:"instance_type"`

	// 对应Sku
	LocalCategory []string `json:"local_category"`

	// 类型
	// single（单副本） | double（双副本) | readone (单可读) | readthree （3可读） | readfive（5只读）
	NodeType []string `json:"node_type"`

	// 后端存储引擎
	// Redis | Memcache
	// example: redis
	Engine []string `json:"engine"`

	// 后端存储引擎版本
	// example: 4.0
	EngineVersion []string `json:"engine_version"`

	// 网络类型, CLASSIC（经典网络）  VPC（专有网络）
	// example: CLASSIC
	NetworkType []string `json:"network_type"`

	NetworkFilterListBase

	//  内网DNS
	PrivateDNS []string `json:"private_dns"`

	//  内网IP地址
	PrivateIpAddr []string `json:"private_ip_addr"`

	// 内网访问端口
	PrivateConnectPort []int `json:"private_connect_port"`

	// 公网DNS
	PublicDNS []string `json:"public_dns"`

	// 公网IP地址
	PublicIpAddr []string `json:"public_ip_addr"`

	// 外网访问端口
	PublicConnectPort []int `json:"public_connect_port"`

	// 访问密码？ on （开启密码）|off （免密码访问）
	AuthMode []string `json:"auth_mode"`
}

type ElasticcacheAccountListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ElasticcacheFilterListInput

	// 账号类型 normal |admin
	AccountType []string `json:"account_type"`

	// 账号权限 read | write | repl（复制, 复制权限支持读写，且开放SYNC/PSYNC命令）
	AccountPrivilege []string `json:"account_privilege"`
}

type ElasticcacheAclListInput struct {
	apis.StandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ElasticcacheFilterListInput

	// Ip地址白名单列表
	IpList string `json:"ip_list"`
}

type ElasticcacheBackupListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ElasticcacheFilterListInput

	// 备份类型, 全量|增量额
	BackupType []string `json:"backup_type"`

	// 备份模式，自动|手动
	BackupMode []string `json:"backup_mode"`
}

type ElasticcacheParameterListInput struct {
	apis.StandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ElasticcacheFilterListInput

	// 参数名称
	Key []string `json:"key"`

	// 参数值
	Value []string `json:"value"`
}
