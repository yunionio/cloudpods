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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/reflectutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

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

func (self *SGroupJointsBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.GroupJointResourceDetails, error) {
	return api.GroupJointResourceDetails{}, nil
}

func (manager *SGroupJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GroupJointResourceDetails {
	rows := make([]api.GroupJointResourceDetails, len(objs))

	jointRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	groupIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GroupJointResourceDetails{
			VirtualJointResourceBaseDetails: jointRows[i],
		}
		var base *SGroupJointsBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.GroupId) > 0 {
			groupIds[i] = base.GroupId
		}
	}

	groupIdMaps, err := db.FetchIdNameMap2(GroupManager, groupIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := groupIdMaps[groupIds[i]]; ok {
			rows[i].Instancegroup = name
		}
	}

	return rows
}
