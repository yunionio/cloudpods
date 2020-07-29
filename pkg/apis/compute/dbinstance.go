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

import "yunion.io/x/onecloud/pkg/apis"

type DBInstanceCreateInput struct {
	apis.VirtualResourceCreateInput
	DeletePreventableCreateInput

	// Ip子网名称或Id,建议使用Id
	// 谷歌云并不实际使用Ip子网,仅仅通过Ip子网确定Vpc
	// required: true
	Network string `json:"network"`
	// swagger:ignore
	NetworkId string

	// Ip子网内的地址,不填则按照ip子网的地址分配策略分配一个ip
	// required: false
	Address string `json:"address"`

	// rds实例名称或Id,建议使用Id
	// 创建只读实例时此参数必传
	MasterInstance string `json:"master_instance"`
	// swagger:ignore
	MasterInstanceId string

	// 安全组名称或Id
	// default: default
	Secgroup string `json:"secgroup"`
	// swagger:ignore
	SecgroupId string

	// 主可用区名称或Id, 此参数从指定的套餐所在的可用区获取
	Zone1 string `json:"zone1"`

	// 次可用区名称或Id, 此参数从指定的套餐所在的可用区获取
	Zone2 string `json:"zone2"`

	// 三节点可用区名称或Id,, 此参数从指定的套餐所在的可用区获取
	Zone3 string `json:"zone3"`

	// swagger:ignore
	ZoneId string

	// 区域名称或Id，建议使用Id
	// swagger:ignore
	Cloudregion string `json:"cloudregion"`

	// swagger:ignore
	CloudregionId string

	// swagger:ignore
	VpcId string

	// swagger:ignore
	ManagerId string

	// swagger:ignore
	NetworkExternalId string

	// 包年包月时间周期
	Duration string `json:"duration"`

	// swagger:ignore
	BillingType string
	// swagger:ignore
	BillingCycle string

	// 套餐名称, 若此参数不填, 则必须有vmem_size_mb及vcpu_count参数
	// 套餐列表可以通过 dbinstancesku 获取
	InstanceType string `json:"instance_type"`

	// rds引擎
	// enum: MySQL, SQLServer, PostgreSQL, MariaDB, Oracle, PPAS
	// required: true
	Engine string `json:"engine"`

	// rds引擎版本
	// 根据各个引擎版本各不相同
	// required: true
	EngineVersion string `json:"engine_version"`

	// rds类型
	//
	//
	//
	// | 平台		| 支持类型	| 说明 |
	// | -----		| ------	| --- |
	// | 华为云		|ha, single, replica| |
	// | 阿里云		|basic, high_availability, always_on, finance||
	// | Google		|Zonal, Regional | |
	// 翻译:
	// basic: 基础版
	// high_availability: 高可用
	// always_on: 集群版
	// finance: 金融版, 三节点
	// ha: 高可用
	// single: 单机
	// replica: 只读
	// Zonal: 单区域
	// Regional: 区域级
	// required: true
	Category string `json:"category"`

	// rds存储类型
	//
	//
	//
	// | 平台	| 支持类型	|
	// | 华为云	|SSD, SAS, SATA|
	// | 阿里云	|local_ssd, cloud_essd, cloud_ssd|
	// | Google	|PD_SSD, PD_HDD|
	// 翻译:
	// PD_SSD: SSD
	// PD_HDD: HDD
	// required: true
	StorageType string `json:"storage_type"`

	// rds存储大小
	// 可参考rds套餐的大小范围和步长情况
	// required: true
	DiskSizeGB int `json:"disk_size_gb"`

	// rds初始化密码
	// 阿里云不需要此参数
	// 华为云会默认创建一个用户,若不传此参数, 则为随机密码
	// 谷歌云会默认创建一个用户,若不传此参数, 则为随机密码
	Password string `json:"password"`

	// 是否不设置初始密码
	// 华为云不支持此参数
	// 谷歌云仅mysql支持此参数
	ResetPassword *bool `json:"reset_password"`

	// rds实例cpu大小
	// 若指定实例套餐，此参数将根据套餐设置
	VcpuCount int `json:"vcpu_count"`

	// rds实例内存大小
	// 若指定实例套餐，此参数将根据套餐设置
	VmemSizeMb int `json:"vmem_size_mb"`

	// swagger:ignore
	Provider string
}

type SDBInstanceChangeConfigInput struct {
	apis.Meta

	InstanceType string
	VCpuCount    int
	VmemSizeMb   int
	StorageType  string
	DiskSizeGB   int
	Category     string
}

type SDBInstanceRecoveryConfigInput struct {
	apis.Meta

	// swagger:ignore
	DBInstancebackup string `json:"dbinstancebackup" "yunion:deprecated-by":"dbinstancebackup_id"`

	// 备份Id
	//
	//
	// | 平台		| 支持引擎								| 说明		|
	// | -----		| ------								| ---		|
	// | 华为云		|MySQL, SQL Server						| 仅SQL Server支持恢复到当前实例			|
	// | 阿里云		|MySQL, SQL Server						| MySQL要求必须开启单库单表恢复功能 并且只能是MySQL 8.0 高可用版（本地SSD盘）MySQL 5.7 高可用版（本地SSD盘）或MySQL 5.6 高可用版, MySQL仅支持恢复到当前实例|
	// | Google		|MySQL, PostgreSQL, SQL Server			| PostgreSQL备份恢复时，要求实例不能有副本			|
	DBInstancebackupId string `json:"dbinstancebackup_id"`

	// 数据库信息, 例如 {"src":"dest"} 是将备份中的src数据库恢复到目标实例的dest数据库中, 阿里云此参数为必传
	// example: {"sdb1":"ddb1"}
	Databases map[string]string `json:"databases,allowempty"`
}

type DBInstanceListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	VpcFilterListInput

	ZoneResourceInput

	MasterInstance string `json:"master_instance"`

	VcpuCount int `json:"vcpu_count"`

	VmemSizeMb int `json:"vmem_size_mb"`

	StorageType string `json:"storage_type"`

	Category string `json:"category"`

	Engine string `json:"engine"`

	EngineVersion string `json:"engine_version"`

	InstanceType string `json:"instance_type"`
}

type DBInstanceBackupListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	RegionalFilterListInput

	DBInstanceFilterListInputBase

	// RDS引擎
	// example: MySQL
	Engine []string `json:"engine"`

	// RDS引擎版本
	// example: 5.7
	EngineVersion []string `json:"engine_version"`

	// 备份模式
	BackupMode []string `json:"backup_mode"`

	// 数据库名称
	DBNames string `json:"db_names"`
}

type DBInstancePrivilegeListInput struct {
	apis.ResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	// filter by dbinstanceaccount
	Dbinstanceaccount string `json:"dbinstanceaccount"`
	// filter by dbinstancedatabase
	Dbinstancedatabase string `json:"dbinstancedatabase"`

	// 权限
	Privilege []string `json:"privilege"`
}

type DBInstanceParameterListInput struct {
	apis.StandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	DBInstanceFilterListInput

	// 参数名称
	Key []string `json:"key"`

	// 参数值
	Value []string `json:"value"`
}

type DBInstanceDatabaseListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	DBInstanceFilterListInput

	// 数据库字符集
	CharacterSet []string `json:"character_set"`
}

type DBInstanceAccountListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	DBInstanceFilterListInput
}

type DBInstanceDetails struct {
	apis.VirtualResourceDetails
	CloudregionResourceInfo
	ManagedResourceInfo

	VpcResourceInfoBase

	SDBInstance

	// 安全组名称
	// example: Default
	Secgroup string `json:"secgroup"`
	// iops
	// example: 0
	Iops int `json:"iops"`
	// IP子网名称
	// example: test-network
	Network string `json:"network"`

	// Zone1名称
	Zone1Name string `json:"zone1_name"`
	// Zone2名称
	Zone2Name string `json:"zone2_name"`
	// Zone3名称
	Zone3Name string `json:"zone3_name"`
}

type DBInstanceResourceInfoBase struct {
	// RDS实例名称
	DBInstance string `json:"dbinstance"`
}

type DBInstanceResourceInfo struct {
	DBInstanceResourceInfoBase

	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo
}

type DBInstanceResourceInput struct {
	// RDS实例(ID or Name)
	DBInstance string `json:"dbinstance"`

	// swagger:ignore
	// Deprecated
	DBInstanceId string `json:"dbinstance_id" "yunion:deprecated-by":"dbinstance"`
}

type DBInstanceFilterListInputBase struct {
	DBInstanceResourceInput

	// 以RDS实例名字排序
	OrderByDBInstance string `json:"order_by_dbinstance"`
}

type DBInstanceFilterListInput struct {
	DBInstanceFilterListInputBase

	VpcFilterListInput
}
