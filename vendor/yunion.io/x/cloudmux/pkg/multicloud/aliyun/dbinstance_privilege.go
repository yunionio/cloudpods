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

package aliyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

type SDatabasePrivilege struct {
	account *SDBInstanceAccount

	AccountPrivilege       string
	AccountPrivilegeDetail string
	DBName                 string
}

func (privilege *SDatabasePrivilege) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", privilege.account.GetName(), privilege.DBName)
}

func (privilege *SDatabasePrivilege) GetPrivilege() string {
	switch privilege.AccountPrivilege {
	case "ReadWrite":
		return api.DATABASE_PRIVILEGE_RW
	case "ReadOnly":
		return api.DATABASE_PRIVILEGE_R
	case "DDLOnly":
		return api.DATABASE_PRIVILEGE_DDL
	case "DMLOnly":
		return api.DATABASE_PRIVILEGE_DML
	case "DBOwner":
		return api.DATABASE_PRIVILEGE_OWNER
	case "Custom":
		return api.DBINSTANCE_DATABASE_CREATING
	}
	return privilege.AccountPrivilege
}

func (privilege *SDatabasePrivilege) GetDBName() string {
	return privilege.DBName
}
