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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SSchedtagJointsManager struct {
	db.SJointResourceBaseManager
	SSchedtagResourceBaseManager
}

func NewSchedtagJointsManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	master db.IStandaloneModelManager,
) *SSchedtagJointsManager {
	return &SSchedtagJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			master,
			SchedtagManager,
		),
	}
}

// +onecloud:model-api-gen
type SSchedtagJointsBase struct {
	db.SJointResourceBase

	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // =Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SSchedtagJointsManager) GetSlaveFieldName() string {
	return "schedtag_id"
}

func (man *SSchedtagJointsManager) FetchSchedtagById(id string) *SSchedtag {
	schedtagObj, _ := SchedtagManager.FetchById(id)
	if schedtagObj == nil {
		return nil
	}
	return schedtagObj.(*SSchedtag)
}

func (man *SSchedtagJointsManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	schedtagId, err := data.GetString("schedtag_id")
	if err != nil || schedtagId == "" {
		return nil, httperrors.NewInputParameterError("schedtag_id not provide")
	}
	resourceType := man.GetMasterManager().KeywordPlural()
	if !utils.IsInStringArray(resourceType, SchedtagManager.GetResourceTypes()) {
		return nil, httperrors.NewInputParameterError("Not support resource_type %s", resourceType)
	}
	schedtag := man.FetchSchedtagById(schedtagId)
	if schedtag == nil {
		return nil, httperrors.NewNotFoundError("Schedtag %s", schedtagId)
	}
	if resourceType != schedtag.ResourceType {
		return nil, httperrors.NewInputParameterError("Schedtag %s resource_type mismatch: %s != %s", schedtag.GetName(), schedtag.ResourceType, resourceType)
	}

	input := apis.JoinResourceBaseCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal JointResourceCreateInput fail %s", err)
	}
	input, err = man.SJointResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (man *SSchedtagJointsManager) GetResourceIdKey(m db.IJointModelManager) string {
	return fmt.Sprintf("%s_id", m.GetMasterManager().Keyword())
}

func (joint *SSchedtagJointsBase) GetSchedtagId() string {
	return joint.SchedtagId
}

func (manager *SSchedtagJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	rows := make([]interface{}, len(objs))

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	tagIds := make([]string, len(rows))
	resIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.SchedtagJointResourceDetails{
			JointResourceBaseDetails: jointRows[i],
		}
		obj := objs[i].(ISchedtagJointModel)
		tagIds[i] = obj.GetSchedtagId()
		resIds[i] = obj.GetResourceId()
	}

	tags := make(map[string]SSchedtag)
	err := db.FetchStandaloneObjectsByIds(SchedtagManager, tagIds, &tags)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if schedtag, ok := tags[tagIds[i]]; ok {
			out := rows[i].(api.SchedtagJointResourceDetails)
			out.Schedtag = schedtag.Name
			out.ResourceType = schedtag.ResourceType
			rows[i] = out
		}
	}

	resIdMaps, err := db.FetchIdNameMap2(manager.GetMasterManager(), resIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 %sIds error: %v", manager.GetMasterManager().Keyword(), err)
		return rows
	}

	for idx := range objs {
		obj := objs[idx].(ISchedtagJointModel)
		baseDetail := rows[idx].(api.SchedtagJointResourceDetails)
		out := obj.GetDetails(baseDetail, resIdMaps[resIds[idx]], isList)
		rows[idx] = out
	}

	return rows
}

func (joint *SSchedtagJointsBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return fmt.Errorf("Delete must be override")
}

func (joint *SSchedtagJointsBase) delete(obj db.IJointModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, obj)
}

func (joint *SSchedtagJointsBase) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return fmt.Errorf("Detach must be override")
}

func (joint *SSchedtagJointsBase) detach(obj db.IJointModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (manager *SSchedtagJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSchedtagResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SchedtagFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SSchedtagJointsManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagJointsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SSchedtagResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SchedtagFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
