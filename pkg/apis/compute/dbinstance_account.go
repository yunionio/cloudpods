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

type SDBInstanceAccountPrivilege struct {
	// 数据库名称或Id
	// required: true
	Database string `json:"database"`
	// swagger:ignore
	DBInstancedatabaseId string

	// 权限类型
	//
	//
	//
	// | 平台        |Rds引擎                |    支持类型           |
	// | ----        |-------                |    --------           |
	// | Aliyun      |MySQL, MariaBD         |    rw, r, ddl, dml    |
	// | Aliyun      |SQLServer              |    rw, r, owner       |
	// | Huawei      |MySQL, MariaDB         |    rw, r              |
	// 同一平台不同的rds类型支持的权限不尽相同
	// required: true
	Privilege string `json:"privilege"`
}

type DBInstanceAccountCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// rds实例名称或Id,建议使用Id
	//
	//
	//
	// | 平台        |支持Rds引擎                |
	// | ----        |-------                    |
	// | Aliyun      |MySQL, MariaBD, SQLServer  |
	// | 华为云      |MySQL, MariaBD             |
	// | 腾讯云      |MySQL                      |
	// required: true
	// 阿里云SQL Server 2017集群版不支持创建账号
	// 实例状态必须是运行中
	DBInstance string `json:"dbinstance"`
	// swagger:ignore
	DBInstanceId string `json:"dbinstance_id"`

	// 账号密码，若不指定,则会随机生成
	// required: false
	Password string `json:"password"`

	// 账号权限
	// required: false
	Privileges []SDBInstanceAccountPrivilege `json:"privileges"`
}

type SDBInstanceSetPrivilegesInput struct {
	Privileges []SDBInstanceAccountPrivilege `json:"privileges"`
}

type DBInstancePrivilege struct {
	// 数据库名称
	Database string `json:"database"`
	// 账号名称
	Account string `json:"account"`
	// 数据库Id
	DBInstancedatabaseId string `json:"dbinstancedatabase_id"`
	// 权限
	Privileges string `json:"privileges"`
}

type DBInstanceAccountDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ProjectizedResourceInfo
	DBInstanceResourceInfo

	SDBInstanceAccount

	// 账号权限列表
	DBInstanceprivileges []DBInstancePrivilege `json:"dbinstanceprivileges,allowempty"`

	ProjectId string `json:"tenant_id"`
}

type DBInstanceAccountUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput
}
