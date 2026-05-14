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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiKey stores a named upstream API key (or other secret material) for reuse by routing or providers.
type SAiKey struct {
	db.SEnabledStatusStandaloneResourceBase

	// AiProviderId optionally associates this key with a catalog provider row.
	AiProviderId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	// Secret holds raw key material; only privileged scopes should list it.
	Secret string `width:"4096" charset:"ascii" nullable:"false" create:"required"`
	// Weight is used for weighted random load balancing among matching keys (default 1).
	Weight int `default:"1" nullable:"false" list:"user" create:"optional" update:"user"`
	// Routing limits which request "model" values may use this key.
	Routing *api.SAiKeyRouting `length:"medium" charset:"utf8" list:"user" create:"optional" update:"user"`
}

type SAiKeyManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var AiKeyManager *SAiKeyManager

func init() {
	AiKeyManager = &SAiKeyManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SAiKey{},
			"ai_keys_tbl",
			"ai_key",
			"ai_keys",
		),
	}
	AiKeyManager.SetVirtualObject(AiKeyManager)
}

func (manager *SAiKeyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiKeyListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	if id := strings.TrimSpace(query.AiProviderId); id != "" {
		q = q.Equals("ai_provider_id", id)
	}
	return q, nil
}

func (manager *SAiKeyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiKeyDetails {
	rows := make([]api.AiKeyDetails, len(objs))
	baseRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	providerIds := make([]string, len(objs))
	for i := range objs {
		rows[i].EnabledStatusStandaloneResourceDetails = baseRows[i]
		k := objs[i].(*SAiKey)
		providerIds[i] = k.AiProviderId
	}
	providerNames, err := db.FetchIdNameMap2(AiProviderManager, providerIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 ai_provider: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].AiProviderName, _ = providerNames[providerIds[i]]
	}
	return rows
}

func (manager *SAiKeyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiKeyCreateInput,
) (api.AiKeyCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	if input.Weight < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "weight must be >= 0")
	}
	if input.Weight == 0 {
		input.Weight = 1
	}
	if strings.TrimSpace(input.Secret) == "" {
		return input, errors.Wrap(httperrors.ErrInputParameter, "secret is required")
	}
	if strings.TrimSpace(input.AiProviderId) == "" {
		return input, errors.Wrap(httperrors.ErrInputParameter, "ai_provider_id is required")
	}
	prov, err := fetchEnabledAiProvider(ctx, userCred, input.AiProviderId)
	if err != nil {
		return input, err
	}
	input.AiProviderId = prov.Id
	if input.Enabled == nil && input.Disabled == nil {
		input.SetEnabled()
	}
	return input, nil
}

func (k *SAiKey) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.AiKeyUpdateInput,
) (*api.AiKeyUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = k.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}
	if input.Weight < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "weight must be >= 0")
	}
	if pid := strings.TrimSpace(input.AiProviderId); pid != "" {
		prov, err := fetchEnabledAiProvider(ctx, userCred, pid)
		if err != nil {
			return input, err
		}
		input.AiProviderId = prov.Id
	}
	return input, nil
}
