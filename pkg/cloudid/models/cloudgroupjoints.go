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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCloudgroupJointsManager struct {
	db.SJointResourceBaseManager
}

func NewCloudgroupJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SCloudgroupJointsManager {
	return SCloudgroupJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			CloudgroupManager,
			slave,
		),
	}
}

// +onecloud:swagger-gen-ignore
type SCloudgroupJointsBase struct {
	db.SJointResourceBase

	// 用户组Id
	CloudgroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true" json:"cloudgroup_id"`
}

func (self *SCloudgroupJointsBase) getCloudgroup() (*SCloudgroup, error) {
	group, err := CloudgroupManager.FetchById(self.CloudgroupId)
	if err != nil {
		return nil, errors.Wrap(err, "FetchById")
	}
	return group.(*SCloudgroup), nil
}

func (manager *SCloudgroupJointsManager) GetMasterFieldName() string {
	return "cloudgroup_id"
}

func (manager *SCloudgroupJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	groupCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudgroupJointResourceDetails {
	rows := make([]api.CloudgroupJointResourceDetails, len(objs))

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, groupCred, query, objs, fields, isList)

	groupIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.CloudgroupJointResourceDetails{
			JointResourceBaseDetails: jointRows[i],
		}
		var base *SCloudgroupJointsBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.CloudgroupId) > 0 {
			groupIds[i] = base.CloudgroupId
		}
	}

	groupIdMaps, err := db.FetchIdNameMap2(CloudgroupManager, groupIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := groupIdMaps[groupIds[i]]; ok {
			rows[i].Cloudgroup = name
		}
	}

	return rows
}

func (manager *SCloudgroupJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	groupCred mcclient.TokenCredential,
	query api.CloudgroupJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, groupCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}

	if len(query.Cloudgroup) > 0 {
		group, err := CloudgroupManager.FetchByIdOrName(ctx, nil, query.Cloudgroup)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudgroup", query.Cloudgroup)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudgroup_id", group.GetId())
	}

	return q, nil
}

func (manager *SCloudgroupJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	groupCred mcclient.TokenCredential,
	query api.CloudgroupJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, groupCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (self *SCloudgroupJointsBase) ValidateUpdateData(
	ctx context.Context,
	groupCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.CloudgroupJointBaseUpdateInput,
) (api.CloudgroupJointBaseUpdateInput, error) {
	var err error
	input.JointResourceBaseUpdateInput, err = self.SJointResourceBase.ValidateUpdateData(ctx, groupCred, query, input.JointResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SJointResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SCloudgroupJointsManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	groupCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemExportKeys(ctx, q, groupCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}
