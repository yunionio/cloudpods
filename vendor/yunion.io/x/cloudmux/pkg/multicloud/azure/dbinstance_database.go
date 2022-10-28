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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceDatabase struct {
	region *SRegion
	multicloud.SDBInstanceDatabaseBase
	AzureTags
	ID         string                        `json:"id"`
	Name       string                        `json:"name"`
	Type       string                        `json:"type"`
	Properties SDBInstanceDatabaseProperties `json:"properties"`
}

type SDBInstanceDatabaseProperties struct {
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
}

func (self *SRegion) ListSDBInstanceDatabase(Id string) ([]SDBInstanceDatabase, error) {
	type databases struct {
		Value []SDBInstanceDatabase
	}

	result := databases{}
	err := self.get(fmt.Sprintf("%s/databases", Id), url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s/databases)", Id)
	}
	return result.Value, nil
}

func (database *SDBInstanceDatabase) GetCharacterSet() string {
	return database.Properties.Charset
}

func (database *SDBInstanceDatabase) GetName() string {
	return database.Name
}

func (database *SDBInstanceDatabase) GetGlobalId() string {
	return strings.ToLower(database.ID)
}

func (database *SDBInstanceDatabase) GetId() string {
	return strings.ToLower(database.ID)
}

func (database *SDBInstanceDatabase) IsEmulated() bool {
	return false
}

func (database *SDBInstanceDatabase) Refresh() error {
	newdb := SDBInstanceDatabase{}
	err := database.region.get(database.ID, url.Values{}, &newdb)
	if err != nil {
		return errors.Wrapf(err, "database.region.get(%s, url.Values{}, &newdb)", database.ID)
	}

	return jsonutils.Update(database, newdb)
}

func (database *SDBInstanceDatabase) GetStatus() string {
	return api.DBINSTANCE_DATABASE_RUNNING
}
