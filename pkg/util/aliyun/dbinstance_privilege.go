package aliyun

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDatabasePrivilege struct {
	account *SDBInstanceAccount

	AccountPrivilege       string
	AccountPrivilegeDetail string
	DBName                 string
}

func (privilege *SDatabasePrivilege) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", privilege.account.GetGlobalId(), privilege.DBName)
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
