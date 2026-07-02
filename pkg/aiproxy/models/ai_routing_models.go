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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		rm := objs[i].(*SAiRoutingModel)
		rows[i].AiRoutingId = rm.AiRoutingId
		rows[i].AiProviderId = rm.AiProviderId
		rows[i].AiModelId = rm.AiModelId
		rows[i].Priority = rm.Priority
		rows[i].ModelPattern = rm.ModelPattern
		rows[i].Enabled = rm.Enabled.IsTrue()
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
func pickAiRoutingModel(ctx context.Context, userCred mcclient.TokenCredential, routing *SAiRouting, reqModel string, body *jsonutils.JSONDict) (providerId, modelId string, routingLog *AiRoutingLog, err error) {
	if routing == nil {
		return "", "", nil, errors.Wrap(httperrors.ErrInvalidStatus, "nil ai_routing")
	}
	entries, err := fetchEnabledAiRoutingModels(routing.Id)
	if err != nil {
		return "", "", nil, err
	}
	if len(entries) == 0 {
		return "", "", nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_routing has no ai_routing_models")
	}
	var modelsById map[string]*SAiModel
	if routing.RouterEnabled {
		modelIds := make([]string, 0, len(entries))
		for i := range entries {
			modelIds = append(modelIds, entries[i].AiModelId)
		}
		modelsById, err = fetchEnabledAiModelsByIds(modelIds)
		if err != nil {
			return "", "", nil, err
		}
	}
	selected, routingLog, err := pickAiRoutingModelFromEntriesWithLog(ctx, routing, reqModel, body, entries, modelsById)
	if err != nil {
		return "", "", routingLog, err
	}
	return selected.AiProviderId, selected.AiModelId, routingLog, nil
}

type aiRoutingModelCandidate struct {
	entry     *SAiRoutingModel
	modelName string
}

func pickAiRoutingModelFromEntries(
	ctx context.Context,
	routing *SAiRouting,
	reqModel string,
	body *jsonutils.JSONDict,
	entries []SAiRoutingModel,
	modelsById map[string]*SAiModel,
) (*SAiRoutingModel, error) {
	selected, _, err := pickAiRoutingModelFromEntriesWithLog(ctx, routing, reqModel, body, entries, modelsById)
	return selected, err
}

func pickAiRoutingModelFromEntriesWithLog(
	ctx context.Context,
	routing *SAiRouting,
	reqModel string,
	body *jsonutils.JSONDict,
	entries []SAiRoutingModel,
	modelsById map[string]*SAiModel,
) (*SAiRoutingModel, *AiRoutingLog, error) {
	allEntries := make([]*SAiRoutingModel, 0, len(entries))
	for i := range entries {
		allEntries = append(allEntries, &entries[i])
	}
	if len(allEntries) == 0 {
		return nil, nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_routing has no ai_routing_models")
	}

	rlog := &AiRoutingLog{Method: "priority"}
	if routing != nil && routing.RouterEnabled {
		rlog.Enabled = true
		rlog.Method = "router"
		candidates := buildAiRoutingModelCandidates(routing, allEntries, modelsById)
		if len(candidates) == 0 {
			selected, err := routerFallbackPick(routing, allEntries[0], errors.Wrap(httperrors.ErrInvalidStatus, "router has no candidate models"))
			rlog.Error = "router has no candidate models"
			if err == nil {
				rlog.Method = "priority_fallback"
				rlog.SelectedModel = clientFacingModelID(routing, selected, modelsById[selected.AiModelId])
			}
			return selected, rlog, err
		}

		candidateNames := make([]string, 0, len(candidates))
		for i := range candidates {
			candidateNames = append(candidateNames, candidates[i].modelName)
		}
		rlog.Candidates = candidateNames
		start := time.Now()
		out, err := callAiRoutingRouter(ctx, routing, body, candidateNames)
		rlog.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			selected, ferr := routerFallbackPick(routing, allEntries[0], err)
			rlog.Error = err.Error()
			if ferr == nil {
				rlog.Method = "priority_fallback"
				rlog.SelectedModel = clientFacingModelID(routing, selected, modelsById[selected.AiModelId])
			}
			return selected, rlog, ferr
		}
		selected := strings.TrimSpace(out.Model)
		rlog.SelectedModel = selected
		rlog.Scores = out.Scores
		rlog.Confidence = out.Confidence
		rlog.Reason = out.Reason
		for i := range candidates {
			if strings.EqualFold(candidates[i].modelName, selected) {
				return candidates[i].entry, rlog, nil
			}
		}
		err = errors.Wrapf(httperrors.ErrInvalidStatus, "router selected model %q outside candidates", selected)
		selectedEntry, ferr := routerFallbackPick(
			routing,
			allEntries[0],
			err,
		)
		rlog.Error = err.Error()
		if ferr == nil {
			rlog.Method = "priority_fallback"
			rlog.SelectedModel = clientFacingModelID(routing, selectedEntry, modelsById[selectedEntry.AiModelId])
		}
		return selectedEntry, rlog, ferr
	}

	matches := make([]*SAiRoutingModel, 0, len(entries))
	for i := range entries {
		e := &entries[i]
		if !modelPatternMatches(e.ModelPattern, reqModel) {
			continue
		}
		matches = append(matches, e)
	}
	if len(matches) == 0 {
		return nil, rlog, errors.Wrapf(httperrors.ErrNotFound, "no ai_routing_model matched request model %q", reqModel)
	}
	rlog.SelectedModel = strings.TrimSpace(matches[0].ModelPattern)
	if rlog.SelectedModel == "" {
		rlog.SelectedModel = strings.TrimSpace(reqModel)
	}
	return matches[0], rlog, nil
}

func buildAiRoutingModelCandidates(routing *SAiRouting, entries []*SAiRoutingModel, modelsById map[string]*SAiModel) []aiRoutingModelCandidate {
	candidates := make([]aiRoutingModelCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		name := clientFacingModelID(routing, entry, modelsById[entry.AiModelId])
		if name == "" {
			continue
		}
		candidates = append(candidates, aiRoutingModelCandidate{
			entry:     entry,
			modelName: name,
		})
	}
	return candidates
}

func routerFallbackPick(routing *SAiRouting, first *SAiRoutingModel, err error) (*SAiRoutingModel, error) {
	if routing == nil || strings.TrimSpace(routing.RouterFallbackPolicy) == "" || strings.EqualFold(routing.RouterFallbackPolicy, api.AiRoutingRouterFallbackPriority) {
		return first, nil
	}
	return nil, err
}

type aiRoutingRouterResult struct {
	Model      string
	Scores     map[string]interface{}
	Confidence *float64
	Reason     string
}

func callAiRoutingRouter(ctx context.Context, routing *SAiRouting, body *jsonutils.JSONDict, candidates []string) (*aiRoutingRouterResult, error) {
	if routing == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "nil ai_routing")
	}
	endpoint, err := aiRoutingRouterEndpoint(routing.RouterUrl, routing.RouterRoutePath)
	if err != nil {
		return nil, err
	}
	payload := jsonutils.NewDict()
	if body != nil {
		if model, _ := body.GetString("model"); model != "" {
			payload.Set("model", jsonutils.NewString(model))
		}
		if messages, err := body.Get("messages"); err == nil {
			payload.Set("messages", messages)
		}
	}
	payload.Set("candidates", jsonutils.Marshal(candidates))

	timeout := routing.RouterTimeoutSeconds
	if timeout <= 0 {
		timeout = api.AiRoutingRouterDefaultTimeoutSeconds
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader([]byte(payload.String())))
	if err != nil {
		return nil, errors.Wrap(err, "new router request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "call ai routing router")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, errors.Errorf("router status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out struct {
		Model         string                 `json:"model"`
		SelectedModel string                 `json:"selected_model"`
		Scores        map[string]interface{} `json:"scores"`
		Confidence    *float64               `json:"confidence"`
		Reason        string                 `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errors.Wrap(err, "decode router response")
	}
	model := strings.TrimSpace(out.Model)
	if model == "" {
		model = strings.TrimSpace(out.SelectedModel)
	}
	if model == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "router response missing model")
	}
	return &aiRoutingRouterResult{
		Model:      model,
		Scores:     out.Scores,
		Confidence: out.Confidence,
		Reason:     out.Reason,
	}, nil
}

func aiRoutingRouterEndpoint(routerUrl, routePath string) (string, error) {
	routerUrl = strings.TrimSpace(routerUrl)
	if routerUrl == "" {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "router_url is required when router is enabled")
	}
	u, err := url.Parse(routerUrl)
	if err != nil {
		return "", errors.Wrap(err, "parse router_url")
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.Wrap(httperrors.ErrInputParameter, "router_url must be absolute")
	}
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		routePath = api.AiRoutingRouterDefaultRoutePath
	}
	if !strings.HasPrefix(routePath, "/") {
		routePath = "/" + routePath
	}
	basePath := strings.TrimRight(u.Path, "/")
	cleanRoutePath := strings.TrimRight(routePath, "/")
	if basePath == cleanRoutePath {
		return u.String(), nil
	}
	u.Path = basePath + routePath
	return u.String(), nil
}
