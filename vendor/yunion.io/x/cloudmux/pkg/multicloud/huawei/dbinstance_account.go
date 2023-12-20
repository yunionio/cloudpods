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
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/util/httputils"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	HuaweiTags
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
	_, err := region.delete(SERVICE_RDS, fmt.Sprintf("instances/%s/db_user/%s", instanceId, account))
	return err
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
	params := url.Values{}
	params.Set("limit", "100")
	page := 1
	params.Set("page", fmt.Sprintf("%d", page))
	ret := []SDBInstanceAccount{}
	for {
		resp, err := region.list(SERVICE_RDS, fmt.Sprintf("instances/%s/db_user/detail", instanceId), params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Users      []SDBInstanceAccount
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Users...)
		if len(ret) >= part.TotalCount || len(part.Users) == 0 {
			break
		}
		page++
		params.Set("page", fmt.Sprintf("%d", page))
	}
	return ret, nil
}

func (region *SRegion) GetDBInstancePrivileges(instanceId string, username string) ([]SDatabasePrivilege, error) {
	params := url.Values{}
	params.Set("instance_id", instanceId)
	params.Set("user-name", username)
	params.Set("limit", "100")
	page := 1
	params.Set("page", fmt.Sprintf("%d", page))
	ret := []SDatabasePrivilege{}
	for {
		resp, err := region.list(SERVICE_RDS, fmt.Sprintf("instances/%s/db_user/database", instanceId), params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Databases  []SDatabasePrivilege
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Databases...)
		if len(ret) >= part.TotalCount || len(part.Databases) == 0 {
			break
		}
		page++
		params.Set("page", fmt.Sprintf("%d", page))
	}
	return ret, nil
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
	resource := fmt.Sprintf("instances/%s/db_privilege", instanceId)
	url, err := region.client.getUrl(SERVICE_RDS, region.ID, resource, httputils.DELETE, nil)
	if err != nil {
		return err
	}
	_, err = region.client.request(httputils.DELETE, url, nil, params)
	return err
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
	_, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/db_privilege", instanceId), params)
	return err
}

func (account *SDBInstanceAccount) ResetPassword(password string) error {
	return account.instance.region.ResetDBInstanceAccountPassword(account.instance.Id, account.Name, password)
}

func (region *SRegion) ResetDBInstanceAccountPassword(instanceId, account, password string) error {
	return fmt.Errorf("The API does not exist or has not been published in the environment")
}
