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

package hcs

import (
	"fmt"
	"net/url"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	multicloud.HuaweiTags
	instance *SDBInstance
	Name     string
}

func (account *SDBInstanceAccount) GetName() string {
	return account.Name
}

func (account *SDBInstanceAccount) Delete() error {
	return account.instance.region.rdsDelete(fmt.Sprintf("%s/db_user/%s", account.instance.Id, account.Name))
}

func (account *SDBInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	privileges, err := account.instance.region.GetDBInstancePrivileges(account.instance.Id, account.Name)
	if err != nil {
		return nil, err
	}
	iprivileves := []cloudprovider.ICloudDBInstanceAccountPrivilege{}
	for i := 0; i < len(privileges); i++ {
		privileges[i].account = account
		iprivileves = append(iprivileves, &privileges[i])
	}
	return iprivileves, nil
}

func (region *SRegion) GetDBInstanceAccounts(instanceId string) ([]SDBInstanceAccount, error) {
	accounts := []SDBInstanceAccount{}
	err := region.rdsList(fmt.Sprintf("instances/%s/db_user/detail", instanceId), nil, accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (region *SRegion) GetDBInstancePrivileges(instanceId string, username string) ([]SDatabasePrivilege, error) {
	query := url.Values{}
	query.Add("iser-name", username)
	privileges := []SDatabasePrivilege{}
	err := region.rdsList(fmt.Sprintf("instances/%s/db_user/database", instanceId), query, privileges)
	if err != nil {
		return nil, err
	}
	return privileges, nil
}

func (account *SDBInstanceAccount) RevokePrivilege(database string) error {
	return account.instance.region.RevokeDBInstancePrivilege(account.instance.Id, account.Name, database)
}

func (region *SRegion) RevokeDBInstancePrivilege(instanceId string, account, database string) error {
	params := map[string]interface{}{
		"db_name": database,
		"users": []map[string]interface{}{
			map[string]interface{}{
				"name": account,
			},
		},
	}
	return region.rdsDBPrivilegesDelete(fmt.Sprintf("instances/%s/db_privilege", instanceId), params)
}

func (account *SDBInstanceAccount) GrantPrivilege(database, privilege string) error {
	return account.instance.region.GrantDBInstancePrivilege(account.instance.Id, account.Name, database, privilege)
}

func (region *SRegion) GrantDBInstancePrivilege(instanceId string, account, database string, privilege string) error {
	readonly := false
	switch privilege {
	case api.DATABASE_PRIVILEGE_R:
		readonly = true
	case api.DATABASE_PRIVILEGE_RW:
	default:
		return fmt.Errorf("Unknown privilege %s", privilege)
	}
	params := map[string]interface{}{
		"db_name": database,
		"users": []map[string]interface{}{
			map[string]interface{}{
				"name":     account,
				"readonly": readonly,
			},
		},
	}
	resp := &struct {
		Resp string
	}{}
	err := region.rdsDBPrivilegesGrant(fmt.Sprintf("instances/%s/db_privilege", instanceId), params, resp)
	return err
}

func (account *SDBInstanceAccount) ResetPassword(password string) error {
	return account.instance.region.ResetDBInstanceAccountPassword(account.instance.Id, account.Name, password)
}

func (region *SRegion) ResetDBInstanceAccountPassword(instanceId, account, password string) error {
	return fmt.Errorf("The API does not exist or has not been published in the environment")
}
