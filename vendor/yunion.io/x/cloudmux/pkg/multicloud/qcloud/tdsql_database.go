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

package qcloud

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type STDSQLDatabase struct {
	rds *STDSQL
	multicloud.SResourceBase
	QcloudTags

	DbName string
}

func (self *STDSQLDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}

func (self *STDSQLDatabase) GetId() string {
	return self.DbName
}

func (self *STDSQLDatabase) GetName() string {
	return self.DbName
}

func (self *STDSQLDatabase) GetGlobalId() string {
	return self.DbName
}

func (self *STDSQLDatabase) GetCharacterSet() string {
	return ""
}

func (self *STDSQLDatabase) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetTDSQLDatabases(id string) ([]STDSQLDatabase, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	resp, err := self.dcdbRequest("DescribeDatabases", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDatabases")
	}
	ret := []STDSQLDatabase{}
	err = resp.Unmarshal(&ret, "Databases")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *STDSQL) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	dbs, err := self.region.GetTDSQLDatabases(self.InstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTDSQLDatabases")
	}
	ret := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := range dbs {
		dbs[i].rds = self
		ret = append(ret, &dbs[i])
	}
	return ret, nil
}
