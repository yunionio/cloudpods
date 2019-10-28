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

package models

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SDBInstanceJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewDBInstanceJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SDBInstanceJointsManager {
	return SDBInstanceJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			DBInstanceManager,
			slave,
		),
	}
}

type SDBInstanceJointsBase struct {
	db.SVirtualJointResourceBase

	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (self *SDBInstanceJointsBase) getDBInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.DBInstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "getDBInstance.FetchById")
	}
	return instance.(*SDBInstance), nil
}

func (manager *SDBInstanceJointsManager) GetMasterFieldName() string {
	return "dbinstance_id"
}
