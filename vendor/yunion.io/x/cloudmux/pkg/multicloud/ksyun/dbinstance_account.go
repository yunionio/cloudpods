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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDatabasePrivilege struct {
	InstanceDatabaseName string
	Privilege            string
}

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	SKsyunTags

	instance *SDBInstance

	InstanceAccountName        string
	InstanceAccountDescription string
	Created                    string
	InstanceAccountType        string
	InstanceAccountPrivileges  []SDatabasePrivilege
}

func (account *SDBInstanceAccount) GetName() string {
	return account.InstanceAccountName
}

func (account *SDBInstanceAccount) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (account *SDBInstanceAccount) RevokePrivilege(database string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RevokePrivilege")
}

func (account *SDBInstanceAccount) GrantPrivilege(database, privilege string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "GrantPrivilege")
}

func (account *SDBInstanceAccount) ResetPassword(password string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ResetPassword")
}

func (account *SDBInstanceAccount) GetStatus() string {
	return api.DBINSTANCE_USER_AVAILABLE
}

func (account *SDBInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceAccountPrivileges")
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts, err := rds.region.GetDBInstanceAccounts(rds.DBInstanceIdentifier)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstanceAccount{}
	for i := 0; i < len(accounts); i++ {
		accounts[i].instance = rds
		ret = append(ret, &accounts[i])
	}
	return ret, nil
}

func (region *SRegion) GetDBInstanceAccounts(id string) ([]SDBInstanceAccount, error) {
	params := map[string]interface{}{
		"DBInstanceIdentifier": id,
	}
	body, err := region.rdsRequest("DescribeInstanceAccounts", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeInstanceAccounts")
	}
	accounts := []SDBInstanceAccount{}
	err = body.Unmarshal(&accounts, "Data", "Accounts")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return accounts, nil
}
