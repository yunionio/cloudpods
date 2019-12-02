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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSchedtagJointsManager struct {
	db.SJointResourceBaseManager
}

func NewSchedtagJointsManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	master db.IStandaloneModelManager,
	slave db.IStandaloneModelManager,
) *SSchedtagJointsManager {
	return &SSchedtagJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			master,
			slave,
		),
	}
}

type SSchedtagJointsBase struct {
	db.SJointResourceBase

	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // =Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SSchedtagJointsManager) GetMasterFieldName() string {
	return "schedtag_id"
}

func (man *SSchedtagJointsManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, man)
}

func (man *SSchedtagJointsManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, man)
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

func (man *SSchedtagJointsManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, man)
}

func (man *SSchedtagJointsManager) GetMasterIdKey(m db.IJointModelManager) string {
	return fmt.Sprintf("%s_id", m.GetMasterManager().Keyword())
}

func (man *SSchedtagJointsManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master db.IStandaloneModel, slave db.IStandaloneModel) bool {
	return db.IsAdminAllowCreate(userCred, man)
}

func (self *SSchedtagJointsBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SSchedtagJointsBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SSchedtagJointsBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (joint *SSchedtagJointsBase) GetSchedtagId() string {
	return joint.SchedtagId
}

func (joint *SSchedtagJointsBase) master(obj db.IJointModel) db.IStandaloneModel {
	return db.JointMaster(obj)
}

func (joint *SSchedtagJointsBase) GetSchedtag() *SSchedtag {
	return joint.Slave().(*SSchedtag)
}

func (joint *SSchedtagJointsBase) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (joint *SSchedtagJointsBase) getCustomizeColumns(obj db.IJointModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := joint.SJointResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return db.JointModelExtra(obj, extra)
}

func (joint *SSchedtagJointsBase) getExtraDetails(obj db.IJointModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := joint.SJointResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return db.JointModelExtra(obj, extra), nil
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
