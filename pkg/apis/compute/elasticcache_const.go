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
	ELASTIC_CACHE_STATUS_SYNCING               = "syncing"               //（同步中）
	ELASTIC_CACHE_STATUS_SYNC_FAILED           = "sync_failed"           //（同步失败）
)

const (
	ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE     = "available"     // 正常可用
	ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE   = "unavailable"   // 不可用
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATING      = "creating"      // 创建中
	ELASTIC_CACHE_ACCOUNT_STATUS_CREATE_FAILED = "create_failed" //（创建失败）
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETING      = "deleting"      // 删除中
	ELASTIC_CACHE_ACCOUNT_STATUS_DELETE_FAILED = "delete_failed" // 删除失败
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
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED = "expired"  //（备份文件已过期）
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED = "deleted"  //（备份文件已删除）
	ELASTIC_CACHE_BACKUP_STATUS_DELETING       = "deleting" // 删除中
	ELASTIC_CACHE_BACKUP_STATUS_SUCCESS        = "success"  // 备份成功
	ELASTIC_CACHE_BACKUP_STATUS_FAILED         = "failed"   // 备份失败
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
