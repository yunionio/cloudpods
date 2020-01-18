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

package modules

import (
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	DBInstanceAccounts SDBInstanceAccountManager
)

type SDBInstanceAccountManager struct {
	modulebase.ResourceManager
}

func (this *SDBInstanceAccountManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, err := this.Get(s, id, nil)
	if err != nil {
		return nil, err
	}

	account := struct {
		Id     string
		Name   string
		Secret string
	}{}

	err = data.Unmarshal(&account)
	if err != nil {
		return nil, errors.Wrap(err, "data.Unmarshal")
	}

	if len(account.Secret) == 0 {
		return nil, httperrors.NewNotFoundError("No login secret found")
	}

	password, err := utils.DescryptAESBase64(account.Id, account.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "Descrypt")
	}

	return jsonutils.Marshal(map[string]string{
		"account":  account.Name,
		"password": password,
	}), nil
}

func init() {
	DBInstanceAccounts = SDBInstanceAccountManager{NewComputeManager("dbinstanceaccount", "dbinstanceaccounts",
		[]string{"ID", "Name", "DBInstance_id", "Status", "Dbinstanceprivileges"},
		[]string{})}

	registerCompute(&DBInstanceAccounts)
}
