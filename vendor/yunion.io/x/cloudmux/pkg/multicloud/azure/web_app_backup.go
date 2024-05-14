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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SAppBackup struct {
	Id   string
	Name string
	Type string
}

func (self *SAppBackup) GetGlobalId() string {
	return strings.ToLower(self.Id)
}

func (self *SAppBackup) GetName() string {
	return self.Name
}

func (self *SAppBackup) GetType() string {
	return self.Type
}

func (self *SRegion) GetAppSnapshots(appId string) ([]SAppBackup, error) {
	res := fmt.Sprintf("%s/snapshots", appId)
	resp, err := self.list_v2(res, "2023-12-01", nil)
	if err != nil {
		return nil, err
	}
	ret := []SAppBackup{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetAppBackups(appId string) ([]SAppBackup, error) {
	res := fmt.Sprintf("%s/backups", appId)
	resp, err := self.list_v2(res, "2023-12-01", nil)
	if err != nil {
		return nil, err
	}
	ret := []SAppBackup{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SAppBackupConfig struct {
	Id         string
	Name       string
	Properties struct {
		Enabled        bool
		BackupSchedule struct {
			FrequencyInterval     int
			FrequencyUnit         string
			RetentionPeriodInDays int
		}
	}
}

func (self *SRegion) GetAppBackupConfig(appId string) (*SAppBackupConfig, error) {
	res := fmt.Sprintf("%s/config/backup/list", appId)
	resp, err := self.post_v2(res, "2023-12-01", nil)
	if err != nil {
		return nil, err
	}
	ret := &SAppBackupConfig{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SAppSite) GetBackupConfig() cloudprovider.AppBackupConfig {
	ret := cloudprovider.AppBackupConfig{}
	opts, err := self.region.GetAppBackupConfig(self.Id)
	if err != nil {
		return ret
	}
	ret.Enabled = opts.Properties.Enabled
	ret.FrequencyInterval = opts.Properties.BackupSchedule.FrequencyInterval
	ret.FrequencyUnit = opts.Properties.BackupSchedule.FrequencyUnit
	ret.RetentionPeriodInDays = opts.Properties.BackupSchedule.RetentionPeriodInDays
	return ret
}
