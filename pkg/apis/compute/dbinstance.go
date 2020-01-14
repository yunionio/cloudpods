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
	// | 平台		| 支持类型	|
	// | -----		| ------	|
	// | 华为云		|ha, single, replica|
	// | 阿里云		|basic, high_availability, always_on, finance|
	// 翻译:
	// basic: 基础版
	// high_availability: 高可用
	// always_on: 集群版
	// finance: 金融版, 三节点
	// ha: 高可用
	// single: 单机
	// replica: 只读
	// required: true
	Category string `json:"category"`

	// rds存储类型
	//
	//
	//
	// | 平台	| 支持类型	|
	// | 华为云	|SSD, SAS, SATA|
	// | 阿里云	|local_ssd, cloud_essd, cloud_ssd|
	// required: true
	StorageType string `json:"storage_type"`

	// rds存储大小
	// 可参考rds套餐的大小范围和步长情况
	// required: true
	DiskSizeGB int `json:"disk_size_gb"`

	// rds初始化密码
	// 阿里云不需要此参数
	// 华为云会默认创建一个用户,若不传此参数, 则为随机密码
	Password string `json:"password"`

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

	DBInstancebackup   string
	DBInstancebackupId string            `json:"dbinstancebackup_id"`
	Databases          map[string]string `json:"databases,allowempty"`
}

type DBInstanceListInput struct {
	apis.VirtualResourceListInput

	ZonalFilterListInput
	ManagedResourceListInput
	VpcFilterListInput
}

type DBInstanceBackupListInput struct {
	apis.VirtualResourceListInput

	ManagedResourceListInput
	RegionalFilterListInput

	DbinstanceFilterListInput
}

type DBInstancePrivilegeListInput struct {
	apis.ResourceBaseListInput

	// filter by dbinstanceaccount
	Dbinstanceaccount string `json:"dbinstanceaccount"`
	// filter by dbinstancedatabase
	Dbinstancedatabase string `json:"dbinstancedatabase"`
}

type DBInstanceParameterListInput struct {
	apis.StandaloneResourceListInput

	DbinstanceFilterListInput
}

type DBInstanceDatabaseListInput struct {
	apis.StatusStandaloneResourceListInput

	DbinstanceFilterListInput
}

type DBInstanceAccountListInput struct {
	apis.StatusStandaloneResourceListInput

	DbinstanceFilterListInput
}

type DbinstanceFilterListInput struct {
	// filter by dbinstance
	Dbinstance string `json:"dbinstance"`
	// swagger:ignore
	// Deprecated
	// filter by dbinstance_id
	DbinstanceId string `json:"dbinstance_id" deprecated-by:"dbinstance"`
}

type DBInstanceDetails struct {
	apis.VirtualResourceDetails
	SDBInstance

	CloudproviderInfo
	// 虚拟私有网络名称
	// example: test-vpc
	Vpc string `json:"vpc"`
	// 安全组名称
	// example: Default
	Secgroup string `json:"secgroup"`
	// iops
	// example: 0
	Iops int `json:"iops"`
	// IP子网名称
	// example: test-network
	Network string `json:"network"`
	// 标签信息
	Metadata map[string]string `json:"metadata"`
}
