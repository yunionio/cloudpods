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
	stderrors "errors"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiRoutingModel binds a catalog model (and provider) to an ai_routing with per-entry priority.
type SAiRoutingModel struct {
	db.SStandaloneResourceBase

	AiRoutingId  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user" index:"true"`
	AiProviderId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	AiModelId    string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// Priority orders models within the same ai_routing (lower value = higher priority).
	Priority int `default:"100" nullable:"false" list:"user" create:"optional" update:"user"`
	// ModelPattern optionally matches the client request "model" (same rules as ai_routing.model_pattern).
	ModelPattern string            `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	Enabled      tristate.TriState `default:"true" nullable:"false" list:"user" create:"optional" update:"user"`
}

type SAiRoutingModelManager struct {
	db.SStandaloneResourceBaseManager
}

var AiRoutingModelManager *SAiRoutingModelManager

func init() {
	AiRoutingModelManager = &SAiRoutingModelManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SAiRoutingModel{},
			"ai_routing_models_tbl",
			"ai_routing_model",
			"ai_routing_models",
		),
	}
	AiRoutingModelManager.SetVirtualObject(AiRoutingModelManager)
}

func (manager *SAiRoutingModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiRoutingModelListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	if id := strings.TrimSpace(query.AiRoutingId); id != "" {
		q = q.Equals("ai_routing_id", id)
	}
	if id := strings.TrimSpace(query.AiProviderId); id != "" {
		q = q.Equals("ai_provider_id", id)
	}
	if id := strings.TrimSpace(query.AiModelId); id != "" {
		q = q.Equals("ai_model_id", id)
	}
	if query.Enabled != nil {
		q = q.Equals("enabled", *query.Enabled)
	}
	return q, nil
}

func (manager *SAiRoutingModelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiRoutingModelDetails {
	rows := make([]api.AiRoutingModelDetails, len(objs))
	baseRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i].StandaloneResourceDetails = baseRows[i]
	}
	return rows
}

func (manager *SAiRoutingModelManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiRoutingModelCreateInput,
) (api.AiRoutingModelCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}

	routingId := strings.TrimSpace(input.AiRoutingId)
	if routingId == "" {
		return input, errors.Wrap(httperrors.ErrInputParameter, "ai_routing_id is required")
	}
	rObj, err := AiRoutingManager.FetchById(routingId)
	if err != nil {
		return input, errors.Wrap(err, "fetch ai_routing")
	}
	routing := rObj.(*SAiRouting)

	providerId, modelId, err := resolveAiRoutingModelRefs(ctx, userCred, strings.TrimSpace(input.AiProviderId), strings.TrimSpace(input.AiModelId))
	if err != nil {
		return input, err
	}
	input.AiProviderId = providerId
	input.AiModelId = modelId
	input.AiRoutingId = routing.Id

	if strings.TrimSpace(input.Name) == "" {
		input.Name = fmt.Sprintf("%s-%s-%d", routing.Name, modelId, input.Priority)
	}

	if input.Enabled == nil {
		enabled := true
		input.Enabled = &enabled
	}
	return input, nil
}

func fetchAiModelRef(ctx context.Context, userCred mcclient.TokenCredential, providerIdOrName, modelIdOrName string) (*SAiModel, error) {
	modelIdOrName = strings.TrimSpace(modelIdOrName)
	if modelIdOrName == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "ai_model_id is required")
	}
	mObj, err := AiModelManager.FetchByIdOrName(ctx, userCred, modelIdOrName)
	if err == nil {
		return mObj.(*SAiModel), nil
	}
	if !stderrors.Is(err, sql.ErrNoRows) {
		return nil, errors.Wrap(err, "fetch ai_model")
	}
	if pk := strings.TrimSpace(providerIdOrName); pk != "" {
		if mObj2, err2 := AiModelManager.FetchByIdOrName(ctx, userCred, catalogModelId(pk, modelIdOrName)); err2 == nil {
			return mObj2.(*SAiModel), nil
		}
	}
	providerIdOrName = strings.TrimSpace(providerIdOrName)
	if providerIdOrName == "" {
		return nil, errors.Wrap(err, "fetch ai_model")
	}
	prov, err := fetchEnabledAiProvider(ctx, userCred, providerIdOrName)
	if err != nil {
		return nil, errors.Wrap(err, "fetch ai_model by model_key")
	}
	mdl := SAiModel{}
	q := AiModelManager.Query().Equals("ai_provider_id", prov.Id).Equals("model_key", modelIdOrName)
	if err := q.First(&mdl); err != nil {
		return nil, errors.Wrap(err, "fetch ai_model")
	}
	return &mdl, nil
}

func resolveAiRoutingModelRefs(ctx context.Context, userCred mcclient.TokenCredential, providerIdOrName, modelIdOrName string) (string, string, error) {
	mdl, err := fetchAiModelRef(ctx, userCred, providerIdOrName, modelIdOrName)
	if err != nil {
		return "", "", err
	}
	if !mdl.GetEnabled() {
		return "", "", errors.Wrap(httperrors.ErrInvalidStatus, "ai_model disabled")
	}

	providerId := strings.TrimSpace(providerIdOrName)
	if providerId == "" {
		providerId = mdl.AiProviderId
	}
	pObj, err := AiProviderManager.FetchByIdOrName(ctx, userCred, providerId)
	if err != nil {
		return "", "", errors.Wrap(err, "fetch ai_provider")
	}
	prov := pObj.(*SAiProvider)
	if !prov.GetEnabled() {
		return "", "", errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider disabled")
	}
	if mdl.AiProviderId != prov.Id {
		return "", "", errors.Wrap(httperrors.ErrInputParameter, "ai_model does not belong to ai_provider")
	}
	return prov.Id, mdl.Id, nil
}

func routingModelItemPriority(priority, weight int) int {
	if priority != 0 {
		return priority
	}
	if weight != 0 {
		return weight
	}
	return 100
}

func validateAiRoutingModelItems(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	items []api.AiRoutingModelItem,
) ([]api.AiRoutingModelItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]api.AiRoutingModelItem, len(items))
	for i := range items {
		item := items[i]
		providerId, modelId, err := resolveAiRoutingModelRefs(ctx, userCred, strings.TrimSpace(item.AiProviderId), strings.TrimSpace(item.AiModelId))
		if err != nil {
			return nil, errors.Wrapf(err, "models[%d]", i)
		}
		item.AiProviderId = providerId
		item.AiModelId = modelId
		item.Priority = routingModelItemPriority(item.Priority, item.Weight)
		item.Weight = 0
		if item.Enabled == nil {
			enabled := true
			item.Enabled = &enabled
		}
		out[i] = item
	}
	return out, nil
}

func deleteAiRoutingModels(ctx context.Context, routingId string) error {
	routingId = strings.TrimSpace(routingId)
	if routingId == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "ai_routing_id is required")
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf("delete from %s where ai_routing_id = ?", AiRoutingModelManager.TableSpec().Name()),
		routingId,
	)
	if err != nil {
		return errors.Wrap(err, "delete ai_routing_models")
	}
	return nil
}

func createAiRoutingModels(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	routing *SAiRouting,
	items []api.AiRoutingModelItem,
) error {
	if routing == nil || len(items) == 0 {
		return nil
	}
	for i := range items {
		item := items[i]
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		dataDict := jsonutils.NewDict()
		dataDict.Set("ai_routing_id", jsonutils.NewString(routing.Id))
		dataDict.Set("ai_provider_id", jsonutils.NewString(item.AiProviderId))
		dataDict.Set("ai_model_id", jsonutils.NewString(item.AiModelId))
		dataDict.Set("priority", jsonutils.NewInt(int64(item.Priority)))
		if mp := strings.TrimSpace(item.ModelPattern); mp != "" {
			dataDict.Set("model_pattern", jsonutils.NewString(mp))
		}
		dataDict.Set("enabled", jsonutils.JSONTrue)
		if !enabled {
			dataDict.Set("enabled", jsonutils.JSONFalse)
		}
		dataDict.Set("name", jsonutils.NewString(fmt.Sprintf("%s-%s-%d", routing.Name, item.AiModelId, item.Priority)))
		if _, err := db.DoCreate(AiRoutingModelManager, ctx, userCred, nil, dataDict, ownerId); err != nil {
			return errors.Wrapf(err, "create ai_routing_model[%d]", i)
		}
	}
	return nil
}

func fetchAiRoutingModels(routingId string, enabledOnly bool) ([]SAiRoutingModel, error) {
	entries := make([]SAiRoutingModel, 0, 8)
	q := AiRoutingModelManager.Query().Equals("ai_routing_id", routingId)
	if enabledOnly {
		q = q.Equals("enabled", true)
	}
	err := q.Asc("priority").Asc("id").All(&entries)
	if err != nil {
		return nil, errors.Wrap(err, "list ai_routing_models")
	}
	return entries, nil
}

func fetchEnabledAiRoutingModels(routingId string) ([]SAiRoutingModel, error) {
	return fetchAiRoutingModels(routingId, true)
}

// pickAiRoutingModel selects provider/model from ai_routing_models by request model name.
func pickAiRoutingModel(ctx context.Context, userCred mcclient.TokenCredential, routing *SAiRouting, reqModel string) (providerId, modelId string, err error) {
	if routing == nil {
		return "", "", errors.Wrap(httperrors.ErrInvalidStatus, "nil ai_routing")
	}
	entries, err := fetchEnabledAiRoutingModels(routing.Id)
	if err != nil {
		return "", "", err
	}
	if len(entries) == 0 {
		return "", "", errors.Wrap(httperrors.ErrInvalidStatus, "ai_routing has no ai_routing_models")
	}
	for i := range entries {
		e := &entries[i]
		if !modelPatternMatches(e.ModelPattern, reqModel) {
			continue
		}
		return e.AiProviderId, e.AiModelId, nil
	}
	return "", "", errors.Wrapf(httperrors.ErrNotFound, "no ai_routing_model matched request model %q", reqModel)
}
