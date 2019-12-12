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

package db

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SResourceBase struct {
	SModelBase

	CreatedAt     time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user"`
	UpdatedAt     time.Time `nullable:"false" updated_at:"true" list:"user"`
	UpdateVersion int       `default:"0" nullable:"false" auto_version:"true" list:"user"`
	DeletedAt     time.Time ``
	Deleted       bool      `nullable:"false" default:"false"`
}

type SResourceBaseManager struct {
	SModelBaseManager
}

func NewResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SResourceBaseManager {
	return SResourceBaseManager{NewModelBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SResourceBaseManager) GetIResourceModelManager() IResourceModelManager {
	return manager.GetVirtualObject().(IResourceModelManager)
}

func (manager *SResourceBaseManager) Query(fields ...string) *sqlchemy.SQuery {
	return manager.SModelBaseManager.Query(fields...).IsFalse("deleted")
}

func (manager *SResourceBaseManager) RawQuery(fields ...string) *sqlchemy.SQuery {
	return manager.SModelBaseManager.Query(fields...)
}

/*func CanDelete(model IModel, ctx context.Context) bool {
	err := model.ValidateDeleteCondition(ctx)
	if err == nil {
		return true
	} else {
		return false
	}
}*/

func (model *SResourceBase) ResourceModelManager() IResourceModelManager {
	return model.GetModelManager().(IResourceModelManager)
}

func (model *SResourceBase) GetIResourceModel() IResourceModel {
	return model.GetVirtualObject().(IResourceModel)
}

/*func (model *SResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SModelBase.GetCustomizeColumns(ctx, userCred, query)
	canDelete := CanDelete(model, ctx)
	if canDelete {
		extra.Add(jsonutils.JSONTrue, "can_delete")
	} else {
		extra.Add(jsonutils.JSONFalse, "can_delete")
	}
	return extra
}

func (model *SResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SModelBase.GetExtraDetails(ctx, userCred, query)
	canDelete := CanDelete(model, ctx)
	if canDelete {
		extra.Add(jsonutils.JSONTrue, "can_delete")
	} else {
		extra.Add(jsonutils.JSONFalse, "can_delete")
	}
	return extra
}*/

func (model *SResourceBase) MarkDelete() error {
	model.Deleted = true
	model.DeletedAt = timeutils.UtcNow()
	return nil
}

func (model *SResourceBase) MarkUnDelete() error {
	model.Deleted = false
	model.DeletedAt = time.Time{}
	return nil
}

func (model *SResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return DeleteModel(ctx, userCred, model.GetIResourceModel())
}

func (manager *SResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.ResourceBaseCreateInput) (apis.ResourceBaseCreateInput, error) {
	var err error
	input.ModelBaseCreateInput, err = manager.SModelBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ModelBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SModelBaseManager.ValidateCreateData")
	}
	return input, nil
}
