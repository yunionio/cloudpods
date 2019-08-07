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
	DBINSTANCE_DEPLOYING  = "deploying"  //部署中
	DBINSTANCE_RUNNING    = "running"    //使用中
	DBINSTANCE_REBOOTING  = "rebooting"  //重启中
	DBINSTANCE_MIGRATING  = "migrating"  //迁移中
	DBINSTANCE_BACKING_UP = "backing_up" //备份中
	DBINSTANCE_RESTORING  = "restoring"  //备份恢复中
	DBINSTANCE_IMPORTING  = "importing"  //数据导入中
	DBINSTANCE_CLONING    = "cloning"    //克隆中
	DBINSTANCE_DELETING   = "deleting"   //删除中
	DBINSTANCE_UNKNOWN    = "unknown"

	DBINSTANCE_BACKUP_READY    = "ready"
	DBINSTANCE_BACKUP_CREATING = "creating"
	DBINSTANCE_BACKUP_DELETING = "deleting"
	DBINSTANCE_BACKUP_FAILED   = "failed"
	DBINSTANCE_BACKUP_UNKNOWN  = "unknown"

	BACKUP_MODE_AUTOMATED = "automated"
	BACKUP_MODE_MANUAL    = "manual"

	DBINSTANCE_DATABASE_CREATING = "creating"
	DBINSTANCE_DATABASE_RUNNING  = "running"
	DBINSTANCE_DATABASE_DELETING = "deleting"

	DBINSTANCE_USER_UNAVAILABLE = "unavailable"
	DBINSTANCE_USER_AVAILABLE   = "available"

	DATABASE_PRIVILEGE_RW     = "rw"
	DATABASE_PRIVILEGE_R      = "r"
	DATABASE_PRIVILEGE_DDL    = "ddl"
	DATABASE_PRIVILEGE_DML    = "dml"
	DATABASE_PRIVILEGE_OWNER  = "owner"
	DATABASE_PRIVILEGE_CUSTOM = "custom"

	DBINSTANCE_TYPE_MYSQL      = "MySQL"
	DBINSTANCE_TYPE_SQLSERVER  = "SQLServer"
	DBINSTANCE_TYPE_POSTGRESQL = "PostgreSQL"
	DBINSTANCE_TYPE_MARIADB    = "MariaDB"
	DBINSTANCE_TYPE_ORACLE     = "Oracle"
	DBINSTANCE_TYPE_PPAS       = "PPAS"

	DBINSTANCE_CATEGORY_BASIC    = "basic"             //基础版
	DBINSTANCE_CATEGORY_HA       = "high_availability" //高可用或主备
	DBINSTANCE_CATEGORY_ALWAYSON = "always_on"         //集群版
	DBINSTANCE_CATEGORY_FINANCE  = "finance"
	DBINSTANCE_CATEGORY_SINGLE   = "single"  //单机
	DBINSTANCE_CATEGORY_Replica  = "replica" //只读
)
