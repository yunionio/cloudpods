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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiModel stores a model catalog row associated with an SAiProvider.
type SAiModel struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	AiProviderId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// ModelKey is the model id sent to the upstream API (e.g. gpt-4o-mini, qwen-turbo).
	ModelKey string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
	// VisualProviderId is the ai_provider used for tool-delegated image analysis.
	VisualProviderId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	// VisualModelKey is the upstream model id for visual analysis.
	VisualModelKey string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	// Config stores per-model extension settings (e.g. visual delegation).
	Config *api.SAiModelConfig `length:"long" charset:"utf8" list:"user" create:"optional" update:"user"`
}

// VisualActive reports whether visual extension is enabled and columns are set.
func (m *SAiModel) VisualActive() bool {
	if m == nil || !m.Config.VisualEnabled() {
		return false
	}
	return strings.TrimSpace(m.VisualProviderId) != "" && strings.TrimSpace(m.VisualModelKey) != ""
}

type SAiModelManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

var AiModelManager *SAiModelManager

func init() {
	AiModelManager = &SAiModelManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAiModel{},
			"ai_models_tbl",
			"ai_model",
			"ai_models",
		),
	}
	AiModelManager.SetVirtualObject(AiModelManager)
}

func (manager *SAiModelManager) InitializeData() error {
	return backfillEmptyTenantId(manager)
}

func (manager *SAiModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiModelListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	if id := strings.TrimSpace(query.AiProviderId); id != "" {
		q = q.Equals("ai_provider_id", id)
	}
	if key := strings.TrimSpace(query.ModelKey); key != "" {
		q = q.Equals("model_key", key)
	}
	if routingRef := strings.TrimSpace(query.AiRoutingId); routingRef != "" {
		routingId, err := resolveAiRoutingIdForListFilter(ctx, userCred, routingRef)
		if err != nil {
			return nil, err
		}
		modelIds, err := aiModelIdsBoundToRouting(routingId)
		if err != nil {
			return nil, err
		}
		if len(modelIds) == 0 {
			q = q.In("id", []string{"__no_such_ai_model__"})
		} else {
			q = q.In("id", modelIds)
		}
	}
	return q, nil
}

func (manager *SAiModelManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiModelListInput,
) (*sqlchemy.SQuery, error) {
	return manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
}

func (manager *SAiModelManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (m *SAiModel) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(m, ctx, userCred, true); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (m *SAiModel) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(m, ctx, userCred, false); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func resolveAiRoutingIdForListFilter(ctx context.Context, userCred mcclient.TokenCredential, idOrName string) (string, error) {
	obj, err := AiRoutingManager.FetchByIdOrName(ctx, userCred, idOrName)
	if err != nil {
		return "", errors.Wrapf(err, "fetch ai_routing %s", idOrName)
	}
	return obj.GetId(), nil
}

func aiModelIdsBoundToRouting(routingId string) ([]string, error) {
	routingId = strings.TrimSpace(routingId)
	if routingId == "" {
		return nil, nil
	}
	bindings := make([]SAiRoutingModel, 0, 8)
	q := AiRoutingModelManager.Query().Equals("ai_routing_id", routingId).Equals("enabled", true)
	if err := q.All(&bindings); err != nil {
		return nil, errors.Wrap(err, "list ai_routing_models")
	}
	ids := make([]string, 0, len(bindings))
	for i := range bindings {
		ids = append(ids, bindings[i].AiModelId)
	}
	return uniqueNonEmptyStrings(ids), nil
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
	baseRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	providerIds := make([]string, len(objs))
	visualProviderIds := make([]string, 0, len(objs))
	for i := range objs {
		rows[i].VirtualResourceDetails = baseRows[i]
		m := objs[i].(*SAiModel)
		rows[i].ContextWindow = CatalogContextWindow(m.ModelKey)
		providerIds[i] = m.AiProviderId
		if vid := strings.TrimSpace(m.VisualProviderId); vid != "" {
			visualProviderIds = append(visualProviderIds, vid)
		}
	}
	providerNames, err := db.FetchIdNameMap2(AiProviderManager, providerIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 ai_provider: %v", err)
		return rows
	}
	visualProviderNames, err := db.FetchIdNameMap2(AiProviderManager, visualProviderIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 visual ai_provider: %v", err)
		visualProviderNames = nil
	}
	visualProviderKeys, err := db.FetchIdFieldMap2(AiProviderManager, "provider_key", visualProviderIds)
	if err != nil {
		log.Errorf("FetchIdFieldMap2 visual ai_provider provider_key: %v", err)
		visualProviderKeys = nil
	}
	for i := range rows {
		rows[i].AiProviderName, _ = providerNames[providerIds[i]]
		m := objs[i].(*SAiModel)
		rows[i].VisualProviderId = m.VisualProviderId
		rows[i].VisualModelKey = m.VisualModelKey
		rows[i].VisualActive = m.VisualActive()
		rows[i].Config = m.Config
		if visualProviderNames != nil {
			rows[i].VisualProviderName, _ = visualProviderNames[m.VisualProviderId]
		}
		if visualProviderKeys != nil {
			rows[i].VisualProviderKey, _ = visualProviderKeys[m.VisualProviderId]
		}
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
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
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

	vpid, vmk, err := normalizeAiModelVisualFields(ctx, userCred, input.VisualProviderId, input.VisualModelKey)
	if err != nil {
		return input, err
	}
	input.VisualProviderId = vpid
	input.VisualModelKey = vmk
	if err := validateAiModelVisualSettings(input.Config, vpid, vmk); err != nil {
		return input, err
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
	input.VirtualResourceBaseUpdateInput, err = m.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
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

	cfg := m.Config
	if input.Config != nil {
		cfg = input.Config
	}
	vpid := m.VisualProviderId
	vmk := m.VisualModelKey
	if strings.TrimSpace(input.VisualProviderId) != "" || strings.TrimSpace(input.VisualModelKey) != "" {
		nvpid, nvmk, err := normalizeAiModelVisualFields(ctx, userCred, input.VisualProviderId, input.VisualModelKey)
		if err != nil {
			return input, err
		}
		if strings.TrimSpace(input.VisualProviderId) != "" {
			vpid = nvpid
			input.VisualProviderId = nvpid
		}
		if strings.TrimSpace(input.VisualModelKey) != "" {
			vmk = nvmk
			input.VisualModelKey = nvmk
		}
	}
	if err := validateAiModelVisualSettings(cfg, vpid, vmk); err != nil {
		return input, err
	}

	return input, nil
}

func normalizeAiModelVisualFields(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	visualProviderId, visualModelKey string,
) (string, string, error) {
	vpid := strings.TrimSpace(visualProviderId)
	vmk := strings.TrimSpace(visualModelKey)
	if vpid != "" {
		prov, err := fetchEnabledAiProvider(ctx, userCred, vpid)
		if err != nil {
			return "", "", errors.Wrap(err, "visual_provider_id")
		}
		vpid = prov.Id
	}
	return vpid, vmk, nil
}

func validateAiModelVisualSettings(cfg *api.SAiModelConfig, visualProviderId, visualModelKey string) error {
	vpid := strings.TrimSpace(visualProviderId)
	vmk := strings.TrimSpace(visualModelKey)
	if cfg.VisualEnabled() && (vpid == "" || vmk == "") {
		return errors.Wrap(httperrors.ErrInputParameter, "visual_provider_id and visual_model_key are required when visual is enabled")
	}
	return nil
}
