package huawei

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDatabasePrivilege struct {
	account *SDBInstanceAccount

	Name     string
	Readonly bool
}

func (privilege *SDatabasePrivilege) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", privilege.account.GetGlobalId(), privilege.Name)
}

func (privilege *SDatabasePrivilege) GetPrivilege() string {
	if privilege.Readonly {
		return api.DATABASE_PRIVILEGE_R
	}
	return api.DATABASE_PRIVILEGE_RW
}

func (privilege *SDatabasePrivilege) GetDBName() string {
	return privilege.Name
}
