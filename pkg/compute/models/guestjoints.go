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
type SGuestJointsManager struct {
	db.SVirtualJointResourceBaseManager
	SGuestResourceBaseManager
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

// +onecloud:model-api-gen
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

func (manager *SGuestJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestJointResourceDetails {
	rows := make([]api.GuestJointResourceDetails, len(objs))

	jointRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	guestIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GuestJointResourceDetails{
			VirtualJointResourceBaseDetails: jointRows[i],
		}
		var base *SGuestJointsBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.GuestId) > 0 {
			guestIds[i] = base.GuestId
		}
	}

	guestIdMaps, err := db.FetchIdNameMap2(GuestManager, guestIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := guestIdMaps[guestIds[i]]; ok {
			rows[i].Guest = name
			rows[i].Server = name
		}
	}

	return rows
}

func (manager *SGuestJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SGuestResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ServerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SGuestJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SGuestResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ServerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (self *SGuestJointsBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestJointBaseUpdateInput,
) (api.GuestJointBaseUpdateInput, error) {
	var err error
	input.VirtualJointResourceBaseUpdateInput, err = self.SVirtualJointResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualJointResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualJointResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SGuestJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SGuestResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGuestResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGuestResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
