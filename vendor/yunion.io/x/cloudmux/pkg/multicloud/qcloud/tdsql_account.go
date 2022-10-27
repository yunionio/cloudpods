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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type STDSQLAccount struct {
	multicloud.SDBInstanceAccountBase
	QcloudTags
	rds *STDSQL

	UserName    string
	Host        string
	Description string
	CreateTime  string
	UpdateTime  string
	ReadOnly    int
	DelayThresh int
}

func (self *STDSQLAccount) GetName() string {
	return self.UserName
}

func (self *STDSQLAccount) GetHost() string {
	return self.Host
}

func (self *STDSQLAccount) Delete() error {
	return self.rds.region.DeleteTDSQLAccount(self.rds.InstanceId, self.UserName, self.Host)
}

func (self *STDSQLAccount) GrantPrivilege(database, privilege string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQLAccount) RevokePrivilege(database string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQLAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetTDSQLAccount(id string) ([]STDSQLAccount, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	resp, err := self.dcdbRequest("DescribeAccounts", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAccounts")
	}
	ret := []STDSQLAccount{}
	err = resp.Unmarshal(&ret, "Users")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *STDSQL) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts, err := self.region.GetTDSQLAccount(self.InstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTDSQLAccount")
	}
	ret := []cloudprovider.ICloudDBInstanceAccount{}
	for i := range accounts {
		accounts[i].rds = self
		ret = append(ret, &accounts[i])
	}
	return ret, nil
}

func (self *SRegion) DeleteTDSQLAccount(id, name, host string) error {
	params := map[string]string{
		"InstanceId": id,
		"UserName":   name,
		"Host":       host,
	}
	_, err := self.dcdbRequest("DeleteAccount", params)
	return errors.Wrapf(err, "DeleteAccount")
}
