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

package huawei

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	instance *SDBInstance
	Name     string
}

func (account *SDBInstanceAccount) GetId() string {
	return account.Name

}

func (account *SDBInstanceAccount) GetGlobalId() string {
	return account.Name
}

func (account *SDBInstanceAccount) GetName() string {
	return account.Name
}

func (account *SDBInstanceAccount) GetStatus() string {
	return api.DBINSTANCE_USER_AVAILABLE
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
