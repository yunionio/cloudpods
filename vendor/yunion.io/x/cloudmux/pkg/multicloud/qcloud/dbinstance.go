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
	"context"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	ret := []cloudprovider.ICloudDBInstance{}
	mysqls, err := self.GetIMySQLs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIMySQLs")
	}
	ret = append(ret, mysqls...)
	tdsqls, err := self.GetITDSQLs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetITDSQLs")
	}
	ret = append(ret, tdsqls...)
	return ret, nil
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	if strings.HasPrefix(id, "cdb") {
		return self.GetMySQLInstanceById(id)
	} else if strings.HasPrefix(id, "tdsqlshard") {
		return self.GetTDSQL(id)
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateIDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	if opts.Category == api.QCLOUD_DBINSTANCE_CATEGORY_TDSQL {
		return nil, cloudprovider.ErrNotImplemented
	}
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

func (rds *SMySQLInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	if strings.HasPrefix(rds.InstanceId, "cdb") {
		return rds.region.Update(rds.InstanceId, input.NAME)
	}
	return errors.ErrNotImplemented
}
