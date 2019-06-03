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

type SGuestJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewGuestJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SGuestJointsManager {
	return SGuestJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			GuestManager,
			slave,
		),
	}
}

type SGuestJointsBase struct {
	db.SVirtualJointResourceBase

	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (self *SGuestJointsBase) getGuest() *SGuest {
	guest, _ := GuestManager.FetchById(self.GuestId)
	return guest.(*SGuest)
}

func (manager *SGuestJointsManager) GetMasterFieldName() string {
	return "guest_id"
}
