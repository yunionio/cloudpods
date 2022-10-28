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

package hcso

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	huawei.HuaweiTags
	instance *SDBInstance
	Name     string
}

func (account *SDBInstanceAccount) GetName() string {
	return account.Name
}

func (account *SDBInstanceAccount) Delete() error {
	return account.instance.region.DeleteDBInstanceAccount(account.instance.Id, account.Name)
}

func (region *SRegion) DeleteDBInstanceAccount(instanceId string, account string) error {
	return DoDeleteWithSpec(region.ecsClient.DBInstance.DeleteInContextWithSpec, nil, instanceId, fmt.Sprintf("db_user/%s", account), nil, nil)
}

func (account *SDBInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	privileges, err := account.instance.region.GetDBInstancePrivvileges(account.instance.Id, account.Name)
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
	params := map[string]string{
		"instance_id": instanceId,
	}
	accounts := []SDBInstanceAccount{}
	err := doListAllWithPage(region.ecsClient.DBInstance.ListAccounts, params, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (region *SRegion) GetDBInstancePrivvileges(instanceId string, username string) ([]SDatabasePrivilege, error) {
	params := map[string]string{
		"instance_id": instanceId,
		"user-name":   username,
	}
	privileges := []SDatabasePrivilege{}
	err := doListAllWithPage(region.ecsClient.DBInstance.ListPrivileges, params, &privileges)
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
	return DoDeleteWithSpec(region.ecsClient.DBInstance.DeleteInContextWithSpec, nil, instanceId, "db_privilege", nil, jsonutils.Marshal(params))
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
	_, err := region.ecsClient.DBInstance.PerformAction("db_privilege", instanceId, jsonutils.Marshal(params))
	return err
}

func (account *SDBInstanceAccount) ResetPassword(password string) error {
	return account.instance.region.ResetDBInstanceAccountPassword(account.instance.Id, account.Name, password)
}

func (region *SRegion) ResetDBInstanceAccountPassword(instanceId, account, password string) error {
	return fmt.Errorf("The API does not exist or has not been published in the environment")
}
