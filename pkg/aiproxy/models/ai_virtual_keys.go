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
	"strings"

	"github.com/google/uuid"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const virtualKeyPrefix = "sk-"

// SAiVirtualKey stores a client-facing virtual API key; upstream routing is resolved from project-scoped ai_routing rules.
type SAiVirtualKey struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	// OwnerId is the user that owns this virtual key within the project.
	OwnerId string `width:"128" charset:"ascii" index:"true" list:"user" nullable:"false" create:"optional" update:"user"`
	// VirtualKey is the opaque key id or prefix presented to clients (not the upstream provider secret).
	VirtualKey string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	// Limits constrains allowed providers, per-request max_tokens, and request rate.
	Limits *api.SAiVirtualKeyLimits `length:"medium" charset:"utf8" list:"user" create:"optional" update:"user"`
}

type SAiVirtualKeyManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

var AiVirtualKeyManager *SAiVirtualKeyManager

func init() {
	AiVirtualKeyManager = &SAiVirtualKeyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAiVirtualKey{},
			"ai_virtual_keys_tbl",
			"ai_virtual_key",
			"ai_virtual_keys",
		),
	}
	AiVirtualKeyManager.SetVirtualObject(AiVirtualKeyManager)
}

func (m *SAiVirtualKey) GetOwnerId() mcclient.IIdentityProvider {
	owner := db.SOwnerId{
		UserId:    m.OwnerId,
		DomainId:  m.DomainId,
		ProjectId: m.ProjectId,
	}
	return &owner
}

func (manager *SAiVirtualKeyManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (m *SAiVirtualKey) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.GetUserId() == m.OwnerId
}

func (manager *SAiVirtualKeyManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SAiVirtualKeyManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchUserInfo(ctx, data)
}

func (manager *SAiVirtualKeyManager) FilterByOwner(
	ctx context.Context,
	q *sqlchemy.SQuery,
	man db.FilterByOwnerProvider,
	userCred mcclient.TokenCredential,
	owner mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
) *sqlchemy.SQuery {
	if owner != nil && scope == rbacscope.ScopeUser {
		if uid := strings.TrimSpace(owner.GetUserId()); uid != "" {
			q = q.Equals("owner_id", uid)
		}
	}
	return q
}

func (manager *SAiVirtualKeyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiVirtualKeyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	if v := strings.TrimSpace(query.VirtualKey); v != "" {
		q = q.Equals("virtual_key", v)
	}
	userId := strings.TrimSpace(query.UserId)
	if userId != "" {
		if !db.IsAdminAllowList(userCred, manager).Result.IsAllow() {
			return nil, httperrors.NewForbiddenError("only admin may filter by user_id")
		}
		uc, err := db.UserCacheManager.FetchUserByIdOrName(ctx, userId)
		if err != nil {
			return nil, errors.Wrap(err, "fetch user")
		}
		q = q.Equals("owner_id", uc.Id)
	} else if !db.IsAdminAllowList(userCred, manager).Result.IsAllow() {
		q = q.Equals("owner_id", userCred.GetUserId())
	}
	return q, nil
}

func (manager *SAiVirtualKeyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiVirtualKeyListInput,
) (*sqlchemy.SQuery, error) {
	return manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
}

func (manager *SAiVirtualKeyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SAiVirtualKeyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiVirtualKeyDetails {
	rows := make([]api.AiVirtualKeyDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range objs {
		rows[i].VirtualResourceDetails = virtRows[i]
		vk := objs[i].(*SAiVirtualKey)
		if strings.TrimSpace(vk.OwnerId) != "" {
			userIds[i] = vk.OwnerId
		}
	}
	userMaps, err := db.FetchIdNameMap2(db.UserCacheManager, userIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].OwnerName, _ = userMaps[userIds[i]]
	}
	return rows
}

func (m *SAiVirtualKey) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(m, ctx, userCred, true); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (m *SAiVirtualKey) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(m, ctx, userCred, false); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (manager *SAiVirtualKeyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiVirtualKeyCreateInput,
) (api.AiVirtualKeyCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}

	if strings.TrimSpace(input.OwnerId) == "" {
		input.OwnerId = userCred.GetUserId()
	} else if !db.IsAdminAllowList(userCred, manager).Result.IsAllow() && input.OwnerId != userCred.GetUserId() {
		return input, httperrors.NewForbiddenError("cannot create virtual key for another user")
	}

	if err := validateAiVirtualKeyLimits(ctx, userCred, input.Limits); err != nil {
		return input, err
	}

	vk := strings.TrimSpace(input.VirtualKey)
	if vk != "" {
		if !strings.HasPrefix(vk, virtualKeyPrefix) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "virtual_key must start with %q", virtualKeyPrefix)
		}
		if len(vk) > 128 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "virtual_key too long")
		}
		exists, err := aiVirtualKeyExists(vk)
		if err != nil {
			return input, err
		}
		if exists {
			return input, errors.Wrap(httperrors.ErrConflict, "virtual_key already exists")
		}
		input.VirtualKey = vk
	} else {
		input.VirtualKey, err = generateUniqueVirtualKey()
		if err != nil {
			return input, err
		}
	}

	if input.Enabled.IsNone() {
		input.Enabled = tristate.True
	}

	return input, nil
}

func validateAiVirtualKeyLimits(ctx context.Context, userCred mcclient.TokenCredential, lim *api.SAiVirtualKeyLimits) error {
	if lim == nil {
		return nil
	}
	if lim.MaxTokensPerRequest < 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "limits.max_tokens_per_request must be >= 0")
	}
	if lim.RequestsPerMinute < 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "limits.requests_per_minute must be >= 0")
	}
	if len(lim.AllowedAiProviderIds) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(lim.AllowedAiProviderIds))
	for _, idOrName := range lim.AllowedAiProviderIds {
		idOrName = strings.TrimSpace(idOrName)
		if idOrName == "" {
			continue
		}
		pObj, err := AiProviderManager.FetchByIdOrName(ctx, userCred, idOrName)
		if err != nil {
			return errors.Wrapf(err, "limits.allowed_ai_provider_ids: fetch %q", idOrName)
		}
		prov := pObj.(*SAiProvider)
		if !prov.GetEnabled() {
			return errors.Wrapf(httperrors.ErrInvalidStatus, "limits.allowed_ai_provider_ids: ai_provider %q disabled", idOrName)
		}
		resolved = append(resolved, prov.Id)
	}
	lim.AllowedAiProviderIds = resolved
	return nil
}

func aiVirtualKeyExists(virtualKey string) (bool, error) {
	cnt, err := AiVirtualKeyManager.Query().Equals("virtual_key", virtualKey).CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count ai_virtual_key")
	}
	return cnt > 0, nil
}

func generateUniqueVirtualKey() (string, error) {
	const maxAttempts = 8
	for i := 0; i < maxAttempts; i++ {
		vk := virtualKeyPrefix + strings.ReplaceAll(uuid.New().String(), "-", "")
		exists, err := aiVirtualKeyExists(vk)
		if err != nil {
			return "", err
		}
		if !exists {
			return vk, nil
		}
	}
	return "", errors.Wrap(httperrors.ErrConflict, "failed to generate unique virtual_key")
}
