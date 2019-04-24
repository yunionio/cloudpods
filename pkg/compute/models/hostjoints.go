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
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostJointsManager struct {
	db.SJointResourceBaseManager
}

func NewHostJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IStandaloneModelManager) SHostJointsManager {
	return SHostJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			HostManager,
			slave,
		),
	}
}

type SHostJointsBase struct {
	db.SJointResourceBase
}

func (manager *SHostJointsManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SHostJointsManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SHostJointsManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SHostJointsManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master db.IStandaloneModel, slave db.IStandaloneModel) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SHostJointsBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SHostJointsBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SHostJointsBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}
