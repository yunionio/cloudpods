package huawei

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	instance *SDBInstance
	multicloud.SDBInstanceDatabaseBase

	Name         string
	CharacterSet string
}

func (database *SDBInstanceDatabase) GetId() string {
	return database.Name
}

func (database *SDBInstanceDatabase) GetGlobalId() string {
	return database.Name
}

func (database *SDBInstanceDatabase) GetName() string {
	return database.Name
}

func (database *SDBInstanceDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}

func (database *SDBInstanceDatabase) GetCharacterSet() string {
	return database.CharacterSet
}

func (region *SRegion) GetDBInstanceDatabases(instanceId string) ([]SDBInstanceDatabase, error) {
	params := map[string]string{
		"instance_id": instanceId,
	}
	databases := []SDBInstanceDatabase{}
	err := doListAllWithPage(region.ecsClient.DBInstance.ListDatabases, params, &databases)
	if err != nil {
		return nil, err
	}
	return databases, nil
}
