package compute

import "yunion.io/x/onecloud/pkg/apis"

type SDBInstanceAccountPrivilege struct {
	Database             string
	DBInstancedatabaseId string
	Privilege            string
}

type SDBInstanceAccountCreateInput struct {
	apis.Meta

	DBInstanceId string `json:"dbinstance_id"`
	Name         string
	Description  string
	Password     string
	Privileges   []SDBInstanceAccountPrivilege
}

type SDBInstanceSetPrivilegesInput struct {
	Privileges []SDBInstanceAccountPrivilege
}
