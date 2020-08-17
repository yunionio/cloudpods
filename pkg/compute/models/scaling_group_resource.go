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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SScalingGroupResourceBase struct {
	// ScalingGroupId
	ScalingGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SScalingGroupResourceBaseManager struct{}

func (self *SScalingGroupResourceBase) GetScalingGroup() *SScalingGroup {
	obj, _ := ScalingGroupManager.FetchById(self.ScalingGroupId)
	if obj != nil {
		return obj.(*SScalingGroup)
	}
	return nil
}

func (manager *SScalingGroupResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScalingGroupResourceInfo {
	rows := make([]api.ScalingGroupResourceInfo, len(objs))
	scalingGroupIds := make([]string, len(objs))
	for i := range objs {
		var base *SScalingGroupResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SScalingGroupResourceBase in object %s", objs[i])
		}
		scalingGroupIds[i] = base.ScalingGroupId
	}

	for i := range scalingGroupIds {
		rows[i].ScalingGroupId = scalingGroupIds[i]
	}
	scalingGroupNames, err := db.FetchIdNameMap2(ScalingGroupManager, scalingGroupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}
	for i := range rows {
		if name, ok := scalingGroupNames[scalingGroupIds[i]]; ok {
			rows[i].ScalingGroup = name
		}
	}
	return rows
}

func (manager *SScalingGroupResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ScalingGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ScalingGroup) > 0 {
		scalingGroupObj, err := ScalingGroupManager.FetchByIdOrName(userCred, query.ScalingGroup)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ScalingGroupManager.Keyword(), query.ScalingGroup)
			} else {
				return nil, errors.Wrap(err, "ScalingGroupManager.FetchByIdOrName")
			}
		}
		q = q.Equals("scaling_group_id", scalingGroupObj.GetId())
	}
	return q, nil
}

func (manager *SScalingGroupResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery,
	field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "scaling_group":
		scalingGroupQuery := ScalingGroupManager.Query("name", "id").SubQuery()
		q = q.AppendField(scalingGroupQuery.Field("name", field)).Distinct()
		q = q.Join(scalingGroupQuery, sqlchemy.Equals(q.Field("scaling_group_id"), scalingGroupQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SScalingGroupResourceBaseManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	parentId, _ := data.GetString("scaling_group_id")
	return jsonutils.Marshal(map[string]string{"scaling_group_id": parentId})
}

func (manager *SScalingGroupResourceBaseManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	scalingGroupId, _ := values.GetString("scaling_group_id")
	if len(scalingGroupId) > 0 {
		q = q.Equals("scaling_group_id", scalingGroupId)
	}
	return q
}
