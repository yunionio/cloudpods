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

package ksyun

import (
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	multicloud.SDBInstanceDatabaseBase
	SKsyunTags
	instance *SDBInstance

	InstanceDatabaseName         string
	InstanceDatabaseCollation    string
	InstanceDatabaseCollationSet string
	InstanceDatabaseDescription  string
	InstanceDatabaseStatus       string
}

func (database *SDBInstanceDatabase) GetId() string {
	return database.InstanceDatabaseName
}

func (database *SDBInstanceDatabase) GetGlobalId() string {
	return database.InstanceDatabaseName
}

func (database *SDBInstanceDatabase) GetName() string {
	return database.InstanceDatabaseName
}

func (database *SDBInstanceDatabase) GetStatus() string {
	switch database.InstanceDatabaseStatus {
	case "CREATING", "TASKS":
		return api.DBINSTANCE_DATABASE_CREATING
	case "ACTIVE":
		return api.DBINSTANCE_DATABASE_RUNNING
	}
	return strings.ToLower(database.InstanceDatabaseStatus)
}

func (database *SDBInstanceDatabase) GetCharacterSet() string {
	return database.InstanceDatabaseCollationSet
}

func (region *SRegion) GetDBInstanceDatabases(id string) ([]SDBInstanceDatabase, error) {
	params := map[string]interface{}{
		"DBInstanceIdentifier": id,
	}
	resp, err := region.rdsRequest("DescribeInstanceDatabases", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeDatabases")
	}
	ret := []SDBInstanceDatabase{}
	err = resp.Unmarshal(&ret, "Data", "InstanceDatabases")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return ret, nil
}
