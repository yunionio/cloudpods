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

	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/models"
)

type SDBInstanceAccountPrivilege struct {
	models.AccountPrivilege
	Account string
}

func (self *SDBInstanceAccountPrivilege) GetDBName() string {
	if self.DbName != nil {
		return *self.DbName
	}
	return ""
}

func (self *SDBInstanceAccountPrivilege) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.GetDBName(), self.Account, self.GetPrivilege())
}

func (self *SDBInstanceAccountPrivilege) GetPrivilege() string {
	if self.Privilege != nil {
		return *self.Privilege
	}
	return ""
}
