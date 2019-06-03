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

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SGroupJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewGroupJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SGroupJointsManager {
	return SGroupJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			GroupManager,
			slave,
		),
	}
}

type SGroupJointsBase struct {
	db.SVirtualJointResourceBase

	GroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SGroupJointsManager) GetMasterFieldName() string {
	return "group_id"
}

func (self *SGroupJointsBase) GetGroup() *SGroup {
	guest, _ := GroupManager.FetchById(self.GroupId)
	return guest.(*SGroup)
}
