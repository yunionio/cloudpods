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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SGroupJointsManager struct {
	db.SVirtualJointResourceBaseManager
	SGroupResourceBaseManager
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

// +onecloud:model-api-gen
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

func (manager *SGroupJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.GroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGroupResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SGroupJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.GroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGroupJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGroupResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (gjb *SGroupJointsBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := gjb.SVirtualJointResourceBase.GetShortDesc(ctx)
	desc.Set("group_id", jsonutils.NewString(gjb.GroupId))
	group := gjb.GetGroup()
	desc.Set("group", jsonutils.NewString(group.Name))
	return desc
}
