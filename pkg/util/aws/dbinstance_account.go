package aws

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	instance *SDBInstance

	AccountName string
}

func (account *SDBInstanceAccount) GetId() string {
	return account.AccountName
}

func (account *SDBInstanceAccount) GetGlobalId() string {
	return account.AccountName
}

func (account *SDBInstanceAccount) GetName() string {
	return account.AccountName
}

func (account *SDBInstanceAccount) GetStatus() string {
	return api.DBINSTANCE_USER_AVAILABLE
}
