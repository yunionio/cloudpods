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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStatusDomainLevelUserResourceBase struct {
	SStatusDomainLevelResourceBase

	// 本地用户Id
	OwnerId string `width:"128" charset:"ascii" index:"true" list:"user" nullable:"false" create:"optional"`
}

type SStatusDomainLevelUserResourceBaseManager struct {
	SStatusDomainLevelResourceBaseManager
}

func NewStatusDomainLevelUserResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusDomainLevelUserResourceBaseManager {
	return SStatusDomainLevelUserResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: NewStatusDomainLevelResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (model *SStatusDomainLevelUserResourceBase) SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(ctx, model, userCred, status, reason)
}

func (manager *SStatusDomainLevelUserResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StatusDomainLevelUserResourceCreateInput) (apis.StatusDomainLevelUserResourceCreateInput, error) {
	var err error
	input.StatusDomainLevelResourceCreateInput, err = manager.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (manager *SStatusDomainLevelUserResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeUser:
			return q.Equals("owner_id", owner.GetUserId())
		case rbacscope.ScopeDomain:
			sq := UserCacheManager.Query("id").Equals("domain_id", owner.GetProjectDomainId())
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					sqlchemy.In(q.Field("owner_id"), sq.SubQuery()),
				),
			)
		}
	}
	return q
}

func (model *SStatusDomainLevelUserResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: model.DomainId, UserId: model.OwnerId}
	return &owner
}

func (manager *SStatusDomainLevelUserResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusDomainLevelUserResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	if len(query.UserId) > 0 {
		uc, _ := UserCacheManager.FetchUserByIdOrName(ctx, query.UserId)
		if uc == nil {
			return nil, httperrors.NewUserNotFoundError("user %s not found", query.UserId)
		}
		q = q.Equals("owner_id", uc.Id)
	}

	return q, nil
}

func (manager *SStatusDomainLevelUserResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.StatusDomainLevelUserResourceListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusDomainLevelResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusDomainLevelUserResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStatusDomainLevelUserResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusDomainLevelResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

func (manager *SStatusDomainLevelUserResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StatusDomainLevelUserResourceDetails {
	rows := make([]apis.StatusDomainLevelUserResourceDetails, len(objs))
	domainRows := manager.SStatusDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = apis.StatusDomainLevelUserResourceDetails{
			StatusDomainLevelResourceDetails: domainRows[i],
		}
		var base *SStatusDomainLevelUserResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.OwnerId) > 0 {
			userIds[i] = base.OwnerId
		}
	}

	userMaps, err := FetchIdNameMap2(UserCacheManager, userIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail: %v", err)
		return rows
	}

	for i := range rows {
		rows[i].OwnerName, _ = userMaps[userIds[i]]
	}

	return rows
}
