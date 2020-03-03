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

type SAlertJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewAlertJointsManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	slave db.IVirtualModelManager) SAlertJointsManager {
	return SAlertJointsManager{
		db.NewVirtualJointResourceBaseManager(
			dt, tableName, keyword, keywordPlural, AlertManager, slave),
	}
}

type SAlertJointsBase struct {
	db.SVirtualJointResourceBase

	AlertId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (b *SAlertJointsBase) getAlert() *SAlert {
	alert, _ := AlertManager.GetAlert(b.AlertId)
	return alert
}

func (man *SAlertJointsManager) GetMasterFieldName() string {
	return "alert_id"
}
