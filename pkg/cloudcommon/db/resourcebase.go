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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SResourceBase struct {
	SModelBase

	// 资源创建时间
	CreatedAt time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user" json:"created_at"`
	// 资源更新时间
	UpdatedAt time.Time `nullable:"false" updated_at:"true" list:"user" json:"updated_at"`
	// 资源被更新次数
	UpdateVersion int `default:"0" nullable:"false" auto_version:"true" list:"user" json:"update_version"`
	// 资源删除时间
	DeletedAt time.Time `json:"deleted_at" list:"user"`
	// 资源是否被删除
	Deleted bool `nullable:"false" default:"false" json:"deleted" list:"user"`
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

func (manager *SResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SModelBaseManager.OrderByExtraFields(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SModelBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (model *SResourceBase) GetUpdateVersion() int {
	return model.UpdateVersion
}

func (model *SResourceBase) GetUpdatedAt() time.Time {
	return model.UpdatedAt
}

func (model *SResourceBase) GetDeleted() bool {
	return model.Deleted
}

func (manager *SResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.ResourceBaseDetails {
	ret := make([]apis.ResourceBaseDetails, len(objs))
	upperRet := manager.SModelBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		ret[i] = apis.ResourceBaseDetails{
			ModelBaseDetails: upperRet[i],
		}
	}
	return ret
}

func (model *SResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.ResourceBaseUpdateInput) (apis.ResourceBaseUpdateInput, error) {
	var err error
	input.ModelBaseUpdateInput, err = model.SModelBase.ValidateUpdateData(ctx, userCred, query, input.ModelBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SModelBase.ValidateUpdateData")
	}
	return input, nil
}
