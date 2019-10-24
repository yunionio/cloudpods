package compute

import "yunion.io/x/onecloud/pkg/apis"

type SDBInstanceDatabasePrivilege struct {
	Account             string
	DBInstancedccountId string
	Privilege           string
}

type SDBInstanceDatabaseCreateInput struct {
	apis.Meta

	DBInstanceId        string `json:"dbinstance_id"`
	CharacterSet        string
	Name                string
	Description         string
	Accounts            []SDBInstanceDatabasePrivilege
	DBInstanceaccountId string
}
