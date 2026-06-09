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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiModel stores a model catalog row associated with an SAiProvider.
type SAiModel struct {
	db.SEnabledStatusStandaloneResourceBase

	AiProviderId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// ModelKey is the model id sent to the upstream API (e.g. gpt-4o-mini, qwen-turbo).
	ModelKey string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
}

type SAiModelManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var AiModelManager *SAiModelManager

func init() {
	AiModelManager = &SAiModelManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SAiModel{},
			"ai_models_tbl",
			"ai_model",
			"ai_models",
		),
	}
	AiModelManager.SetVirtualObject(AiModelManager)
}

func (manager *SAiModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiModelListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	if id := strings.TrimSpace(query.AiProviderId); id != "" {
		q = q.Equals("ai_provider_id", id)
	}
	if key := strings.TrimSpace(query.ModelKey); key != "" {
		q = q.Equals("model_key", key)
	}
	return q, nil
}

func (manager *SAiModelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiModelDetails {
	rows := make([]api.AiModelDetails, len(objs))
	baseRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	providerIds := make([]string, len(objs))
	for i := range objs {
		rows[i].EnabledStatusStandaloneResourceDetails = baseRows[i]
		m := objs[i].(*SAiModel)
		providerIds[i] = m.AiProviderId
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

func (manager *SAiModelManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiModelCreateInput,
) (api.AiModelCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}

	prov, err := fetchEnabledAiProvider(ctx, userCred, input.AiProviderId)
	if err != nil {
		return input, err
	}
	input.AiProviderId = prov.Id

	mk, err := validateAiModelKey(input.ModelKey)
	if err != nil {
		return input, err
	}
	input.ModelKey = mk

	if err := ensureAiModelKeyUniquePerProvider(ctx, prov.Id, mk, ""); err != nil {
		return input, err
	}

	if strings.TrimSpace(input.Name) == "" {
		input.Name = defaultAiModelName(prov.Name, mk)
	}

	return input, nil
}

func (m *SAiModel) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.AiModelUpdateInput,
) (*api.AiModelUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = m.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}

	providerId := m.AiProviderId
	if pid := strings.TrimSpace(input.AiProviderId); pid != "" {
		prov, err := fetchEnabledAiProvider(ctx, userCred, pid)
		if err != nil {
			return input, err
		}
		providerId = prov.Id
		input.AiProviderId = prov.Id
	}

	modelKey := m.ModelKey
	if mk := strings.TrimSpace(input.ModelKey); mk != "" {
		modelKey, err = validateAiModelKey(mk)
		if err != nil {
			return input, err
		}
		input.ModelKey = modelKey
	}

	if modelKey != m.ModelKey || providerId != m.AiProviderId {
		if err := ensureAiModelKeyUniquePerProvider(ctx, providerId, modelKey, m.Id); err != nil {
			return input, err
		}
	}

	return input, nil
}
