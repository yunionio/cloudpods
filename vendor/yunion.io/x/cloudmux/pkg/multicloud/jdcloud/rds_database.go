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

package jdcloud

import (
	"fmt"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/models"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	multicloud.SDBInstanceDatabaseBase
	JdcloudTags

	rds *SDBInstance
	models.Database
}

func (self *SDBInstanceDatabase) GetCharacterSet() string {
	return self.CharacterSetName
}

func (self *SDBInstanceDatabase) GetGlobalId() string {
	return self.DbName
}

func (self *SDBInstanceDatabase) GetId() string {
	return self.DbName
}

func (self *SDBInstanceDatabase) GetName() string {
	return self.DbName
}

func (self *SDBInstanceDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}

func (self *SRegion) GetDBInstanceDatabases(id string, pageNumber, pageSize int) ([]SDBInstanceDatabase, int, error) {

	req := apis.NewDescribeDatabasesRequestWithAllParams(self.ID, id, nil, &pageNumber, &pageSize)
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{}
	resp, err := client.DescribeDatabases(req)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDatabases")
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	total := resp.Result.TotalCount
	ret := []SDBInstanceDatabase{}
	for i := range resp.Result.Databases {
		ret = append(ret, SDBInstanceDatabase{
			Database: resp.Result.Databases[i],
		})
	}
	return ret, total, nil
}

func (self *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	dbs := []SDBInstanceDatabase{}
	n := 1
	for {
		part, total, err := self.region.GetDBInstanceDatabases(self.InstanceId, n, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstanceDatabases")
		}
		dbs = append(dbs, part...)
		if len(dbs) >= total {
			break
		}
		n++
	}
	ret := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := range dbs {
		dbs[i].rds = self
		ret = append(ret, &dbs[i])
	}
	return ret, nil
}
