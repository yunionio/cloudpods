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

package jdcloud

import (
	"fmt"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/models"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	JdcloudTags

	rds *SDBInstance
	models.Account
}

func (self *SDBInstanceAccount) GetGlobalId() string {
	return self.AccountName
}

func (self *SDBInstanceAccount) GetId() string {
	return self.AccountName
}

func (self *SDBInstanceAccount) GetName() string {
	return self.AccountName
}

func (self *SDBInstanceAccount) GetStatus() string {
	switch self.AccountStatus {
	case "BUILDING":
		return api.DBINSTANCE_USER_CREATING
	case "RUNNING":
		return api.DBINSTANCE_USER_AVAILABLE
	case "DELETING":
		return api.DBINSTANCE_USER_DELETING
	default:
		return api.DBINSTANCE_USER_AVAILABLE
	}
}

func (self *SDBInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	ret := []cloudprovider.ICloudDBInstanceAccountPrivilege{}
	for i := range self.AccountPrivileges {
		ret = append(ret, &SDBInstanceAccountPrivilege{
			Account:          self.AccountName,
			AccountPrivilege: self.AccountPrivileges[i],
		})
	}
	return ret, nil
}

func (self *SRegion) GetDBInstanceAccounts(id string, pageNumber, pageSize int) ([]SDBInstanceAccount, int, error) {
	req := apis.NewDescribeAccountsRequestWithAllParams(self.ID, id, &pageNumber, &pageSize)
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{debug: self.client.debug}
	resp, err := client.DescribeAccounts(req)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeAccounts")
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	total := resp.Result.TotalCount
	ret := []SDBInstanceAccount{}
	for i := range resp.Result.Accounts {
		ret = append(ret, SDBInstanceAccount{
			Account: resp.Result.Accounts[i],
		})
	}
	return ret, total, nil
}

func (self *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts := []SDBInstanceAccount{}
	n := 1
	for {
		part, total, err := self.region.GetDBInstanceAccounts(self.InstanceId, n, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstanceAccounts")
		}
		accounts = append(accounts, part...)
		if len(accounts) >= total {
			break
		}
		n++
	}
	ret := []cloudprovider.ICloudDBInstanceAccount{}
	for i := range accounts {
		accounts[i].rds = self
		ret = append(ret, &accounts[i])
	}
	return ret, nil
}
