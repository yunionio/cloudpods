package aws

import (
	"yunion.io/x/onecloud/pkg/multicloud"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDBInstanceDatabase struct {
	multicloud.SDBInstanceDatabaseBase

	DBName string
}

func (database *SDBInstanceDatabase) GetId() string {
	return database.DBName
}

func (database *SDBInstanceDatabase) GetGlobalId() string {
	return database.DBName
}

func (database *SDBInstanceDatabase) GetName() string {
	return database.DBName
}

func (database *SDBInstanceDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}

func (database *SDBInstanceDatabase) GetCharacterSet() string {
	return ""
}
