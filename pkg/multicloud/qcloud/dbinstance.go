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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	ret := []cloudprovider.ICloudDBInstance{}
	mysql := []SMySQLInstance{}
	for {
		part, total, err := self.ListMySQLInstances([]string{}, len(mysql), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListMySQLInstances")
		}
		mysql = append(mysql, part...)
		if len(mysql) >= total {
			break
		}
	}
	for i := range mysql {
		mysql[i].region = self
		ret = append(ret, &mysql[i])
	}
	return ret, nil
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	if strings.HasPrefix(id, "cdb-") {
		return self.GetMySQLInstanceById(id)
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateIDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	switch opts.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		rds, err := self.CreateMySQLDBInstance(opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateMySQLDBInstance")
		}
		return rds, nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "For %s", opts.Engine)
}
