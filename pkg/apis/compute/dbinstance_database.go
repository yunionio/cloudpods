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

type SDBInstanceDatabasePrivilege struct {
	// 数据库账号名称或Id
	// required: true
	Account string `json:"account"`
	// swagger:ignore
	DBInstanceaccountId string
	// 权限
	//
	//
	//
	// | 平台        |Rds引擎                |    支持类型    |
	// | ----        |-------                |    --------    |
	// | Aliyun      |MySQL, MariaBD         |    rw, r, ddl, dml    |
	// | Aliyun      |SQLServer              |    rw, r, owner    |
	// | Huawei      |MySQL, MariaDB         |    rw, r    |
	// 同一平台不同的rds类型支持的权限不尽相同
	// required: true
	Privilege string `json:"privilege"`
}

type DBInstanceDatabaseCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// rds实例名称或Id,建议使用Id
	//
	//
	//
	// | 平台        |支持Rds引擎                |
	// | ----        |-------                    |
	// | Aliyun      |MySQL, MariaBD, SQLServer  |
	// | 华为云      |MySQL, MariaBD             |
	// | 腾讯云      |                           |
	// required: true
	// 阿里云SQL Server 2017集群版不支持创建数据库
	// 阿里云只读实例不支持创建数据库
	// 实例状态必须是运行中
	DBInstance string `json:"dbinstance"`
	// swagger:ignore
	DBInstanceId string `json:"dbinstance_id"`

	// 数据库字符集
	// required: true
	CharacterSet string `json:"character_set"`

	// 赋予账号权限
	// required: false
	Accounts []SDBInstanceDatabasePrivilege `json:"accounts"`
}

type DBInstancedatabaseDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ProjectizedResourceInfo
	DBInstanceResourceInfo

	SDBInstanceDatabase

	// 数据库权限
	DBInstanceprivileges []DBInstancePrivilege `json:"dbinstanceprivileges"`
	ProjectId            string                `json:"tenant_id"`
}

type DBInstanceparameterDetails struct {
	apis.StandaloneResourceDetails
	DBInstanceResourceInfo

	SDBInstanceParameter
}

type DBInstanceDatabaseUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput
}
