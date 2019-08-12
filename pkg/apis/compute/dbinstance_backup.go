package compute

import "yunion.io/x/onecloud/pkg/apis"

type SDBInstanceBackupCreateInput struct {
	apis.Meta

	DBInstanceId  string `json:"dbinstance_id"`
	Name          string
	Description   string
	DBNames       string
	Databases     []string
	Engine        string
	EngineVersion string
	CloudregionId string
	BackupMode    string
	ManagerId     string
}
