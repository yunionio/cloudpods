package aliyun

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	multicloud.SDBInstanceDatabaseBase

	CharacterSetName string
	DBDescription    string
	DBInstanceId     string
	DBName           string
	DBStatus         string
	Engine           string
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
	switch database.DBStatus {
	case "Creating":
		return api.DBINSTANCE_DATABASE_CREATING
	case "Running":
		return api.DBINSTANCE_DATABASE_RUNNING
	case "Deleting":
		return api.DBINSTANCE_DATABASE_DELETING
	}
	return database.DBStatus
}

func (database *SDBInstanceDatabase) GetCharacterSet() string {
	return database.CharacterSetName
}

func (region *SRegion) GetDBInstanceDatabases(instanceId, dbName string, offset int, limit int) ([]SDBInstanceDatabase, int, error) {
	if limit > 500 || limit <= 0 {
		limit = 500
	}
	params := map[string]string{
		"RegionId":     region.RegionId,
		"PageSize":     fmt.Sprintf("%d", limit),
		"PageNumber":   fmt.Sprintf("%d", (offset/limit)+1),
		"DBInstanceId": instanceId,
	}
	if len(dbName) > 0 {
		params["DBName"] = dbName
	}
	body, err := region.rdsRequest("DescribeDatabases", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeDatabases")
	}
	databases := []SDBInstanceDatabase{}
	err = body.Unmarshal(&databases, "Databases", "Database")
	if err != nil {
		return nil, 0, errors.Wrap(err, "Unmarshal")
	}
	total, _ := body.Int("TotalRecordCount")
	return databases, int(total), nil
}
