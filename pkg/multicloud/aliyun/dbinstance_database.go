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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	multicloud.SDBInstanceDatabaseBase
	instance *SDBInstance

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

func (database *SDBInstanceDatabase) Delete() error {
	return database.instance.region.DeleteDBInstanceDatabase(database.DBInstanceId, database.DBName)
}

func (region *SRegion) DeleteDBInstanceDatabase(instanceId string, dbName string) error {
	params := map[string]string{
		"DBInstanceId": instanceId,
		"DBName":       dbName,
	}

	_, err := region.rdsRequest("DeleteDatabase", params)
	return err
}

func (region *SRegion) CreateDBInstanceDatabae(instanceId, characterSet, dbName, desc string) error {
	params := map[string]string{
		"DBInstanceId":     instanceId,
		"DBName":           dbName,
		"CharacterSetName": characterSet,
		"DBDescription":    desc,
	}

	_, err := region.rdsRequest("CreateDatabase", params)
	return err

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
