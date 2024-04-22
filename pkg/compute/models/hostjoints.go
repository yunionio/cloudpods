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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SHostJointsManager struct {
	db.SJointResourceBaseManager
	SHostResourceBaseManager
}

func NewHostJointsManager(hostIdFieldName string, dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IStandaloneModelManager) SHostJointsManager {
	return SHostJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			HostManager,
			slave,
		),
		SHostResourceBaseManager: SHostResourceBaseManager{
			hostIdFieldName: hostIdFieldName,
		},
	}
}

// +onecloud:model-api-gen
type SHostJointsBase struct {
	db.SJointResourceBase
}

func (manager *SHostJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostJointResourceDetails {
	rows := make([]api.HostJointResourceDetails, len(objs))

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostRows := manager.SHostResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.HostJointResourceDetails{
			JointResourceBaseDetails: jointRows[i],
			HostResourceInfo:         hostRows[i],
		}
	}

	return rows
}

func (manager *SHostJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SHostResourceBaseManager.ListItemFilter(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SHostJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SHostResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SHostJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SHostResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SHostResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
