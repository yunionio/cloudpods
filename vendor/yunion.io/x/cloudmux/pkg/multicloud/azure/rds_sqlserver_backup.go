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
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSQLServerBackup struct {
	rds *SSQLServer
	multicloud.SDBInstanceBackupBase
	AzureTags

	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		Servername              string    `json:"serverName"`
		Servercreatetime        time.Time `json:"serverCreateTime"`
		Databasename            string    `json:"databaseName"`
		Databasedeletiontime    string    `json:"databaseDeletionTime"`
		Backuptime              time.Time `json:"backupTime"`
		Backupstorageredundancy string    `json:"backupStorageRedundancy"`
	} `json:"properties"`
}

func (self *SSQLServerBackup) GetBackupMode() string {
	return api.BACKUP_MODE_AUTOMATED
}

func (self *SSQLServerBackup) GetBackupMethod() cloudprovider.TBackupMethod {
	return cloudprovider.BackupMethodUnknown
}

func (self *SSQLServerBackup) GetBackupSizeMb() int {
	return 0
}

func (self *SSQLServerBackup) GetDBInstanceId() string {
	return self.rds.GetGlobalId()
}

func (self *SSQLServerBackup) GetName() string {
	return self.Name
}

func (self *SSQLServerBackup) GetDBNames() string {
	return self.Properties.Databasename
}

func (self *SSQLServerBackup) GetStatus() string {
	return api.DBINSTANCE_BACKUP_READY
}

func (self *SSQLServerBackup) GetId() string {
	return self.ID
}

func (self *SSQLServerBackup) GetGlobalId() string {
	return strings.ToLower(self.Name)
}

func (self *SSQLServerBackup) GetStartTime() time.Time {
	return self.Properties.Backuptime
}

func (self *SSQLServerBackup) GetEndTime() time.Time {
	return time.Time{}
}

func (self *SSQLServerBackup) GetEngine() string {
	return self.rds.GetEngine()
}

func (self *SSQLServerBackup) GetEngineVersion() string {
	return self.rds.GetEngineVersion()
}

func (self *SRegion) ListSQLServerBackups(id, location string) ([]SSQLServerBackup, error) {
	ret := struct {
		Value []SSQLServerBackup
	}{}
	r := fmt.Sprintf("Microsoft.Sql/locations/%s/longTermRetentionServers", location)
	prefix := strings.Replace(strings.ToLower(id), "microsoft.sql/servers", r, -1)
	return ret.Value, self.get(prefix+"/longTermRetentionBackups", url.Values{}, &ret)
}

func (self *SSQLServer) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := self.region.ListSQLServerBackups(self.ID, self.Location)
	if err != nil {
		return nil, errors.Wrapf(err, "ListSQLServerBackups")
	}
	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := range backups {
		backups[i].rds = self
		ret = append(ret, &backups[i])
	}
	return ret, nil
}
