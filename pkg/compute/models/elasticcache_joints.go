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
type SElasticcacheJointsManager struct {
	db.SVirtualJointResourceBaseManager
	SElasticcacheResourceBaseManager
}

func NewElasticcacheJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SElasticcacheJointsManager {
	return SElasticcacheJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			ElasticcacheManager,
			slave,
		),
	}
}

// +onecloud:model-api-gen
type SElasticcacheJointsBase struct {
	db.SVirtualJointResourceBase
	SElasticcacheResourceBase
}

func (self *SElasticcacheJointsBase) getElasticcache() *SElasticcache {
	ec, _ := ElasticcacheManager.FetchById(self.ElasticcacheId)
	return ec.(*SElasticcache)
}

func (manager *SElasticcacheJointsManager) GetMasterFieldName() string {
	return "elasticcache_id"
}

func (manager *SElasticcacheJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheJointResourceDetails {
	rows := make([]api.ElasticcacheJointResourceDetails, len(objs))

	jointRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	ecIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.ElasticcacheJointResourceDetails{
			VirtualJointResourceBaseDetails: jointRows[i],
		}
		var base *SElasticcacheJointsBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.ElasticcacheId) > 0 {
			ecIds[i] = base.ElasticcacheId
		}
	}

	ecIdMaps, err := db.FetchIdNameMap2(ElasticcacheManager, ecIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := ecIdMaps[ecIds[i]]; ok {
			rows[i].Elasticcache = name
			rows[i].ElasticcacheId = ecIds[i]
		}
	}

	return rows
}

func (manager *SElasticcacheJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SElasticcacheResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SElasticcacheJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SElasticcacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (self *SElasticcacheJointsBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ElasticcacheJointBaseUpdateInput,
) (api.ElasticcacheJointBaseUpdateInput, error) {
	var err error
	input.VirtualJointResourceBaseUpdateInput, err = self.SVirtualJointResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualJointResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualJointResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SElasticcacheJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SElasticcacheResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SElasticcacheResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
