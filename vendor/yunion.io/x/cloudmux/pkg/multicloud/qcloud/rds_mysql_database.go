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
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMySQLInstanceDatabase struct {
	rds *SMySQLInstance
	multicloud.SResourceBase
	QcloudTags

	CharacterSet string
	DatabaseName string
}

func (self *SMySQLInstanceDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}

func (self *SMySQLInstanceDatabase) GetId() string {
	return self.DatabaseName
}

func (self *SMySQLInstanceDatabase) GetName() string {
	return self.DatabaseName
}

func (self *SMySQLInstanceDatabase) GetGlobalId() string {
	return self.DatabaseName
}

func (self *SMySQLInstanceDatabase) GetCharacterSet() string {
	return self.CharacterSet
}

func (self *SMySQLInstanceDatabase) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) DescribeMySQLDatabases(instanceId string, offset, limit int) ([]SMySQLInstanceDatabase, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Offset":     fmt.Sprintf("%d", offset),
		"Limit":      fmt.Sprintf("%d", limit),
		"InstanceId": instanceId,
	}
	resp, err := self.cdbRequest("DescribeDatabases", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDatabases")
	}
	databases := []SMySQLInstanceDatabase{}
	err = resp.Unmarshal(&databases, "DatabaseList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return databases, int(totalCount), nil
}

func (rds *SMySQLInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	ret := []cloudprovider.ICloudDBInstanceDatabase{}
	for {
		part, total, err := rds.region.DescribeMySQLDatabases(rds.InstanceId, len(ret), 100)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeMySQLDatabases")
		}
		for i := range part {
			part[i].rds = rds
			ret = append(ret, &part[i])
		}
		if len(ret) >= total || len(part) == 0 {
			break
		}
	}
	return ret, nil
}
