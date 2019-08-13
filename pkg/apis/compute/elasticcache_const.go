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
	ELASTIC_CACHE_STATUS_DEPLOYING             = "deploying"             //（创建中）
	ELASTIC_CACHE_STATUS_CHANGING              = "changing"              //（修改中）
	ELASTIC_CACHE_STATUS_INACTIVE              = "inactive"              //（被禁用）
	ELASTIC_CACHE_STATUS_FLUSHING              = "flushing"              //（清除中）
	ELASTIC_CACHE_STATUS_RELEASED              = "released"              //（已释放）
	ELASTIC_CACHE_STATUS_TRANSFORMING          = "transforming"          //（转换中）
	ELASTIC_CACHE_STATUS_UNAVAILABLE           = "unavailable"           //（服务停止）
	ELASTIC_CACHE_STATUS_ERROR                 = "error"                 //（创建失败）
	ELASTIC_CACHE_STATUS_MIGRATING             = "migrating"             //（迁移中）
	ELASTIC_CACHE_STATUS_BACKUPRECOVERING      = "backuprecovering"      //（备份恢复中）
	ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING = "minorversionupgrading" //（小版本升级中）
	ELASTIC_CACHE_STATUS_NETWORKMODIFYING      = "networkmodifying"      //（网络变更中）
	ELASTIC_CACHE_STATUS_SSLMODIFYING          = "sslmodifying"          //（SSL变更中）
	ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING = "majorversionupgrading" //（大版本升级中，可正常访问）
)

const (
	ELASTIC_CACHE_ARCH_TYPE_STAND_ALONE  = "standalone"   //
	ELASTIC_CACHE_ARCH_TYPE_MASTER_SLAVE = "master_slave" //
	ELASTIC_CACHE_ARCH_TYPE_CLUSTER      = "cluster"      // 集群
	ELASTIC_CACHE_ARCH_TYPE_PROXY        = "proxy"        // 代理集群
)
