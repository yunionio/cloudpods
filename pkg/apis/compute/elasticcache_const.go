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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ELASTIC_CACHE_STATUS_RUNNING               = compute.ELASTIC_CACHE_STATUS_RUNNING               //（正常）
	ELASTIC_CACHE_STATUS_RESTARTING            = compute.ELASTIC_CACHE_STATUS_RESTARTING            //（重启中）
	ELASTIC_CACHE_STATUS_RESTART_FAILED        = "restart_failed"                                   //（重启失败）
	ELASTIC_CACHE_STATUS_DEPLOYING             = compute.ELASTIC_CACHE_STATUS_DEPLOYING             //（创建中）
	ELASTIC_CACHE_STATUS_CREATE_FAILED         = compute.ELASTIC_CACHE_STATUS_CREATE_FAILED         //（创建失败）
	ELASTIC_CACHE_STATUS_CHANGING              = compute.ELASTIC_CACHE_STATUS_CHANGING              //（修改中）
	ELASTIC_CACHE_STATUS_CHANGE_FAILED         = compute.ELASTIC_CACHE_STATUS_CHANGE_FAILED         //（修改失败）
	ELASTIC_CACHE_STATUS_INACTIVE              = compute.ELASTIC_CACHE_STATUS_INACTIVE              //（被禁用）
	ELASTIC_CACHE_STATUS_FLUSHING              = compute.ELASTIC_CACHE_STATUS_FLUSHING              //（清除中）
	ELASTIC_CACHE_STATUS_FLUSHING_FAILED       = "flushing_failed"                                  //（清除失败）
	ELASTIC_CACHE_STATUS_RELEASING             = compute.ELASTIC_CACHE_STATUS_RELEASING             //（释放中）
	ELASTIC_CACHE_STATUS_RELEASED              = compute.ELASTIC_CACHE_STATUS_RELEASED              //（已释放）
	ELASTIC_CACHE_STATUS_RELEASE_FAILED        = "release_failed"                                   //（释放失败）
	ELASTIC_CACHE_STATUS_TRANSFORMING          = compute.ELASTIC_CACHE_STATUS_TRANSFORMING          //（转换中）
	ELASTIC_CACHE_STATUS_UNAVAILABLE           = compute.ELASTIC_CACHE_STATUS_UNAVAILABLE           //（服务停止）
	ELASTIC_CACHE_STATUS_ERROR                 = compute.ELASTIC_CACHE_STATUS_ERROR                 //（删除失败）
	ELASTIC_CACHE_STATUS_MIGRATING             = compute.ELASTIC_CACHE_STATUS_MIGRATING             //（迁移中）
	ELASTIC_CACHE_STATUS_BACKUPRECOVERING      = compute.ELASTIC_CACHE_STATUS_BACKUPRECOVERING      //（备份恢复中）
	ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING = compute.ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING //（小版本升级中）
	ELASTIC_CACHE_STATUS_NETWORKMODIFYING      = compute.ELASTIC_CACHE_STATUS_NETWORKMODIFYING      //（网络变更中）
	ELASTIC_CACHE_STATUS_SSLMODIFYING          = compute.ELASTIC_CACHE_STATUS_SSLMODIFYING          //（SSL变更中）
	ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING = compute.ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING //（大版本升级中，可正常访问）
	ELASTIC_CACHE_STATUS_UNKNOWN               = compute.ELASTIC_CACHE_STATUS_UNKNOWN               //（未知状态）
	ELASTIC_CACHE_STATUS_DELETING              = compute.ELASTIC_CACHE_STATUS_DELETING              // (删除)
	ELASTIC_CACHE_STATUS_SNAPSHOTTING          = compute.ELASTIC_CACHE_STATUS_SNAPSHOTTING          //（快照）
	ELASTIC_CACHE_STATUS_SYNCING               = "syncing"                                          //（同步中）
	ELASTIC_CACHE_STATUS_SYNC_FAILED           = "sync_failed"                                      //（同步失败）
	ELASTIC_CACHE_RENEWING                     = "renewing"                                         //（续费中）
	ELASTIC_CACHE_RENEW_FAILED                 = "renew_failed"                                     //（续费失败）
	ELASTIC_CACHE_SET_AUTO_RENEW               = "set_auto_renew"                                   //（设置自动续费）
	ELASTIC_CACHE_SET_AUTO_RENEW_FAILED        = "set_auto_renew_failed"                            //（设置自动续费失败）

	ELASTIC_CACHE_ENGINE_REDIS     = compute.ELASTIC_CACHE_ENGINE_REDIS
	ELASTIC_CACHE_ENGINE_MEMCACHED = compute.ELASTIC_CACHE_ENGINE_MEMCACHED
)

const (
	ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE     = compute.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE   // 正常可用
	ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE   = compute.ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE // 不可用
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATING      = compute.ELASTIC_CACHE_ACCOUNT_STATUS_CREATING
	ELASTIC_CACHE_ACCOUNT_STATUS_MODIFYING     = compute.ELASTIC_CACHE_ACCOUNT_STATUS_MODIFYING // 修改中
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATE_FAILED = "create_failed"                                //（创建失败）
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETING      = compute.ELASTIC_CACHE_ACCOUNT_STATUS_DELETING  // 删除中
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETE_FAILED = "delete_failed"                                // 删除失败
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETED       = compute.ELASTIC_CACHE_ACCOUNT_STATUS_DELETED   // 已删除
)

const (
	ELASTIC_CACHE_UPDATE_TAGS        = "update_tags"
	ELASTIC_CACHE_UPDATE_TAGS_FAILED = "update_tags_fail"
)

const (
	ELASTIC_CACHE_ACCOUNT_TYPE_NORMAL = compute.ELASTIC_CACHE_ACCOUNT_TYPE_NORMAL // 普通账号
	ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN  = compute.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN  // 管理账号
)

const (
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ  = compute.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ  // 只读
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE = compute.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE // 读写
	ELASTIC_CACHE_ACCOUNT_PRIVILEGE_REPL  = compute.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_REPL  // 复制，复制权限支持读写，且支持使用SYNC/PSYNC命令。
)

const (
	ELASTIC_CACHE_BACKUP_STATUS_CREATING       = compute.ELASTIC_CACHE_BACKUP_STATUS_CREATING // 备份中
	ELASTIC_CACHE_BACKUP_STATUS_RESTORING      = compute.ELASTIC_CACHE_BACKUP_STATUS_RESTORING
	ELASTIC_CACHE_BACKUP_STATUS_COPYING        = compute.ELASTIC_CACHE_BACKUP_STATUS_COPYING
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED = compute.ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED //（备份文件已过期）
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED = compute.ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED //（备份文件已删除）
	ELASTIC_CACHE_BACKUP_STATUS_DELETING       = compute.ELASTIC_CACHE_BACKUP_STATUS_DELETING       // 删除中
	ELASTIC_CACHE_BACKUP_STATUS_SUCCESS        = compute.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS        // 备份成功
	ELASTIC_CACHE_BACKUP_STATUS_FAILED         = compute.ELASTIC_CACHE_BACKUP_STATUS_FAILED         // 备份失败
	ELASTIC_CACHE_BACKUP_STATUS_UNKNOWN        = compute.ELASTIC_CACHE_BACKUP_STATUS_UNKNOWN        // 未知
)

const (
	ELASTIC_CACHE_BACKUP_TYPE_FULL        = compute.ELASTIC_CACHE_BACKUP_TYPE_FULL        // 全量备份
	ELASTIC_CACHE_BACKUP_TYPE_INCREMENTAL = compute.ELASTIC_CACHE_BACKUP_TYPE_INCREMENTAL // 增量备份
)

const (
	ELASTIC_CACHE_BACKUP_MODE_AUTOMATED = compute.ELASTIC_CACHE_BACKUP_MODE_AUTOMATED // 自动备份
	ELASTIC_CACHE_BACKUP_MODE_MANUAL    = compute.ELASTIC_CACHE_BACKUP_MODE_MANUAL    // 手动触发备份
)

const (
	ELASTIC_CACHE_ACL_STATUS_AVAILABLE     = compute.ELASTIC_CACHE_ACL_STATUS_AVAILABLE // 正常可用
	ELASTIC_CACHE_ACL_STATUS_CREATING      = "creating"                                 // 创建中
	ELASTIC_CACHE_ACL_STATUS_CREATE_FAILED = "create_failed"                            //（创建失败）
	ELASTIC_CACHE_ACL_STATUS_DELETING      = "deleting"                                 // 删除中
	ELASTIC_CACHE_ACL_STATUS_DELETE_FAILED = "delete_failed"                            // 删除失败
	ELASTIC_CACHE_ACL_STATUS_UPDATING      = "updating"                                 // 更新中
	ELASTIC_CACHE_ACL_STATUS_UPDATE_FAILED = "update_failed"                            // 更新失败
)

const (
	ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE     = compute.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE // 正常可用
	ELASTIC_CACHE_PARAMETER_STATUS_UPDATING      = "updating"                                       // 更新中
	ELASTIC_CACHE_PARAMETER_STATUS_UPDATE_FAILED = "update_failed"                                  // 更新失败
)

const (
	ELASTIC_CACHE_ARCH_TYPE_SINGLE  = compute.ELASTIC_CACHE_ARCH_TYPE_SINGLE  // 单副本
	ELASTIC_CACHE_ARCH_TYPE_MASTER  = compute.ELASTIC_CACHE_ARCH_TYPE_MASTER  // 主备
	ELASTIC_CACHE_ARCH_TYPE_CLUSTER = compute.ELASTIC_CACHE_ARCH_TYPE_CLUSTER // 集群
	ELASTIC_CACHE_ARCH_TYPE_RWSPLIT = compute.ELASTIC_CACHE_ARCH_TYPE_RWSPLIT // 读写分离
)

const (
	ELASTIC_CACHE_NODE_TYPE_SINGLE = compute.ELASTIC_CACHE_NODE_TYPE_SINGLE
	ELASTIC_CACHE_NODE_TYPE_DOUBLE = compute.ELASTIC_CACHE_NODE_TYPE_DOUBLE
	ELASTIC_CACHE_NODE_TYPE_THREE  = compute.ELASTIC_CACHE_NODE_TYPE_THREE
	ELASTIC_CACHE_NODE_TYPE_FOUR   = compute.ELASTIC_CACHE_NODE_TYPE_FOUR
	ELASTIC_CACHE_NODE_TYPE_FIVE   = compute.ELASTIC_CACHE_NODE_TYPE_FIVE
	ELASTIC_CACHE_NODE_TYPE_SIX    = compute.ELASTIC_CACHE_NODE_TYPE_SIX
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

	// 通过安全组Id过滤缓存实例
	SecgroupId string `json:"secgroup_id"`
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
