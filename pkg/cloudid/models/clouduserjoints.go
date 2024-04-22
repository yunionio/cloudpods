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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SClouduserJointsManager struct {
	db.SJointResourceBaseManager
	SClouduserResourceBaseManager
}

func NewClouduserJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SClouduserJointsManager {
	return SClouduserJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			ClouduserManager,
			slave,
		),
	}
}

// +onecloud:swagger-gen-ignore
type SClouduserJointsBase struct {
	db.SJointResourceBase

	SClouduserResourceBase
}

func (manager *SClouduserJointsManager) GetMasterFieldName() string {
	return "clouduser_id"
}

func (manager *SClouduserJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ClouduserJointResourceDetails {
	rows := make([]api.ClouduserJointResourceDetails, len(objs))

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userRows := manager.SClouduserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.ClouduserJointResourceDetails{
			JointResourceBaseDetails: jointRows[i],
			ClouduserResourceDetails: userRows[i],
		}
	}
	return rows
}

func (manager *SClouduserJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ClouduserJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SClouduserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ClouduserResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SClouduserJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ClouduserJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (self *SClouduserJointsBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ClouduserJointBaseUpdateInput,
) (api.ClouduserJointBaseUpdateInput, error) {
	var err error
	input.JointResourceBaseUpdateInput, err = self.SJointResourceBase.ValidateUpdateData(ctx, userCred, query, input.JointResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SJointResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SClouduserJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}
