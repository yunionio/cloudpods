package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	apapi "yunion.io/x/onecloud/pkg/apis/aiproxy"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
	"yunion.io/x/pkg/util/printutils"
)

const aiproxyPlaceholderAPIKey = "unused"

func aiproxyAdminSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSession(ctx, options.Options.Region)
}

func mapLLMTypeToProviderKey(llmType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(llmType)) {
	case string(api.LLM_CONTAINER_VLLM):
		return "vllm", true
	case string(api.LLM_CONTAINER_OLLAMA):
		return "ollama", true
	case string(api.LLM_CONTAINER_SGLANG):
		return "sgl", true
	default:
		return "", false
	}
}

func aiProviderNameForLlm(llmId string) string {
	return "llm-" + llmId
}

func aiRoutingNameForDeployment(depId string) string {
	return "llm-dep-" + depId
}

func aiModelNameForLlm(llmId, modelKey string) string {
	return "llm-" + llmId + "-" + slugModelKey(modelKey)
}

func slugModelKey(modelKey string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(modelKey) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastDash = false
		case r == '-', r == '_', r == '.', r == '/', r == ':':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "model"
	}
	return s
}

func clientModelAlias(llm *SLLM, modelKey string) string {
	name := strings.TrimSpace(llm.Name)
	if name == "" {
		name = llm.Id
	}
	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return name
	}
	return name + "-" + modelKey
}

func resolveLlmAccessBaseURL(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) (string, error) {
	if llm.CmpId == "" {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "llm instance has no compute server")
	}
	info, err := llm.GetLLMAccessUrlInfo(ctx, userCred, jsonutils.NewDict())
	if err != nil {
		return "", errors.Wrap(err, "GetLLMAccessUrlInfo")
	}
	if info == nil {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "empty llm access url")
	}
	if u := strings.TrimSpace(info.InternalUrl); u != "" {
		return strings.TrimRight(u, "/"), nil
	}
	if u := strings.TrimSpace(info.LoginUrl); u != "" {
		return strings.TrimRight(u, "/"), nil
	}
	return "", errors.Wrap(httperrors.ErrInvalidStatus, "llm access url is empty")
}

// upstreamModelKeyForBackend maps mounted instant-model metadata to the model id
// the inference backend actually serves (e.g. vLLM --served-model-name basename).
func upstreamModelKeyForBackend(llmType, modelName, modelTag string) string {
	modelName = strings.TrimSpace(modelName)
	modelTag = strings.TrimSpace(modelTag)
	switch strings.ToLower(strings.TrimSpace(llmType)) {
	case string(api.LLM_CONTAINER_VLLM), string(api.LLM_CONTAINER_SGLANG):
		if modelName == "" {
			return ""
		}
		if idx := strings.LastIndex(modelName, "/"); idx >= 0 {
			return modelName[idx+1:]
		}
		return modelName
	case string(api.LLM_CONTAINER_OLLAMA):
		if modelName != "" && modelTag != "" {
			return modelName + ":" + modelTag
		}
		return modelName
	default:
		if modelName != "" && modelTag != "" {
			return modelName + ":" + modelTag
		}
		return modelName
	}
}

func upstreamModelKeyFromInstantModel(llmType string, instMdl *SInstantModel) string {
	if instMdl == nil {
		return ""
	}
	return upstreamModelKeyForBackend(llmType, instMdl.ModelName, instMdl.ModelTag)
}

func upstreamModelKeyFromMountedInfo(llmType string, info *api.MountedModelInfo) string {
	if info == nil {
		return ""
	}
	if info.Id != "" {
		if instMdl, _ := GetInstantModelManager().GetInstantModelById(info.Id); instMdl != nil {
			if key := upstreamModelKeyFromInstantModel(llmType, instMdl); key != "" {
				return key
			}
		}
	}
	if info.FullName != "" {
		parts := strings.SplitN(strings.TrimSpace(info.FullName), ":", 2)
		modelName := parts[0]
		modelTag := ""
		if len(parts) > 1 {
			modelTag = parts[1]
		}
		if key := upstreamModelKeyForBackend(llmType, modelName, modelTag); key != "" {
			return key
		}
	}
	return strings.TrimSpace(info.ModelId)
}

func collectUpstreamModelKeys(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) ([]string, error) {
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMSku")
	}
	llmType := sku.LLMType

	infos, err := llm.FetchMountedModelInfo()
	if err != nil {
		return nil, errors.Wrap(err, "FetchMountedModelInfo")
	}
	keys := make([]string, 0, len(infos))
	seen := make(map[string]struct{}, len(infos))
	for i := range infos {
		key := upstreamModelKeyFromMountedInfo(llmType, &infos[i])
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) > 0 {
		return keys, nil
	}
	for _, m := range GetEffectiveMountedModels(llm, sku) {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		key := ""
		if instMdl, _ := GetInstantModelManager().GetInstantModelById(m); instMdl != nil {
			key = upstreamModelKeyFromInstantModel(llmType, instMdl)
		}
		if key == "" {
			key = m
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no mounted models on llm instance")
	}
	return keys, nil
}

func listAiproxyResources(session *mcclient.ClientSession, man interface {
	List(*mcclient.ClientSession, jsonutils.JSONObject) (*printutils.ListResult, error)
}, filter jsonutils.JSONObject) ([]jsonutils.JSONObject, error) {
	result, err := man.List(session, filter)
	if err != nil {
		return nil, err
	}
	if result == nil || len(result.Data) == 0 {
		return nil, nil
	}
	return result.Data, nil
}

func firstResourceID(rows []jsonutils.JSONObject) string {
	if len(rows) == 0 {
		return ""
	}
	id, _ := rows[0].GetString("id")
	return strings.TrimSpace(id)
}

func upsertAiProvider(
	session *mcclient.ClientSession,
	name, providerKey, baseURL, llmDeploymentId, llmId string,
) (string, error) {
	filter := jsonutils.NewDict()
	if llmId != "" {
		filter.Set("llm_id", jsonutils.NewString(llmId))
	}
	rows, err := listAiproxyResources(session, &apmodules.AiProviders, filter)
	if err != nil {
		return "", errors.Wrap(err, "list ai_providers")
	}
	cfg := jsonutils.Marshal(&apapi.SAiProviderConfig{
		BaseURL: baseURL,
		APIKey:  aiproxyPlaceholderAPIKey,
	})
	params := jsonutils.NewDict()
	params.Set("provider_key", jsonutils.NewString(providerKey))
	params.Set("config", cfg)
	params.Set("llm_deployment_id", jsonutils.NewString(llmDeploymentId))
	params.Set("llm_id", jsonutils.NewString(llmId))
	params.Set("enabled", jsonutils.JSONTrue)

	existingId := firstResourceID(rows)
	if existingId == "" {
		params.Set("name", jsonutils.NewString(name))
		resp, err := apmodules.AiProviders.Create(session, params)
		if err != nil {
			return "", errors.Wrap(err, "create ai_provider")
		}
		id, _ := resp.GetString("id")
		return id, nil
	}
	_, err = apmodules.AiProviders.Update(session, existingId, params)
	if err != nil {
		return "", errors.Wrap(err, "update ai_provider")
	}
	return existingId, nil
}

func upsertAiModel(session *mcclient.ClientSession, name, providerId, modelKey string) (string, error) {
	filter := jsonutils.NewDict()
	filter.Set("ai_provider_id", jsonutils.NewString(providerId))
	filter.Set("model_key", jsonutils.NewString(modelKey))
	rows, err := listAiproxyResources(session, &apmodules.AiModels, filter)
	if err != nil {
		return "", errors.Wrap(err, "list ai_models")
	}
	params := jsonutils.NewDict()
	params.Set("ai_provider_id", jsonutils.NewString(providerId))
	params.Set("model_key", jsonutils.NewString(modelKey))
	params.Set("enabled", jsonutils.JSONTrue)

	existingId := firstResourceID(rows)
	if existingId == "" {
		params.Set("name", jsonutils.NewString(name))
		resp, err := apmodules.AiModels.Create(session, params)
		if err != nil {
			return "", errors.Wrap(err, "create ai_model")
		}
		id, _ := resp.GetString("id")
		return id, nil
	}
	_, err = apmodules.AiModels.Update(session, existingId, params)
	if err != nil {
		return "", errors.Wrap(err, "update ai_model")
	}
	return existingId, nil
}

func ensureAiRouting(
	session *mcclient.ClientSession,
	name, llmDeploymentId string,
	dep *SLLMDeployment,
) (string, error) {
	filter := jsonutils.NewDict()
	filter.Set("llm_deployment_id", jsonutils.NewString(llmDeploymentId))
	rows, err := listAiproxyResources(session, &apmodules.AiRoutings, filter)
	if err != nil {
		return "", errors.Wrap(err, "list ai_routings")
	}
	if id := firstResourceID(rows); id != "" {
		return id, nil
	}
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(name))
	params.Set("llm_deployment_id", jsonutils.NewString(llmDeploymentId))
	params.Set("priority", jsonutils.NewInt(100))
	params.Set("enabled", jsonutils.JSONTrue)
	params.Set("project_id", jsonutils.NewString(dep.ProjectId))
	params.Set("domain_id", jsonutils.NewString(dep.DomainId))
	resp, err := apmodules.AiRoutings.Create(session, params)
	if err != nil {
		return "", errors.Wrap(err, "create ai_routing")
	}
	id, _ := resp.GetString("id")
	return id, nil
}

// SyncLlmInstance registers or updates one running llm replica in aiproxy.
func SyncLlmInstance(ctx context.Context, userCred mcclient.TokenCredential, dep *SLLMDeployment, llm *SLLM) (*api.AiproxyInstanceBinding, error) {
	if dep == nil || llm == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "nil deployment or llm")
	}
	if llm.Status != api.LLM_STATUS_RUNNING {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "llm %s is not running", llm.Id)
	}
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMSku")
	}
	providerKey, ok := mapLLMTypeToProviderKey(sku.LLMType)
	if !ok {
		return nil, errors.Wrapf(httperrors.ErrNotSupported, "llm_type %q is not supported for aiproxy sync", sku.LLMType)
	}
	baseURL, err := resolveLlmAccessBaseURL(ctx, userCred, llm)
	if err != nil {
		return nil, err
	}
	modelKeys, err := collectUpstreamModelKeys(ctx, userCred, llm)
	if err != nil {
		return nil, err
	}

	session := aiproxyAdminSession(ctx)
	providerName := aiProviderNameForLlm(llm.Id)
	providerId, err := upsertAiProvider(session, providerName, providerKey, baseURL, dep.Id, llm.Id)
	if err != nil {
		return nil, err
	}

	primaryModelId := ""
	for _, mk := range modelKeys {
		modelName := aiModelNameForLlm(llm.Id, mk)
		modelId, err := upsertAiModel(session, modelName, providerId, mk)
		if err != nil {
			return nil, err
		}
		if primaryModelId == "" {
			primaryModelId = modelId
		}
	}
	if primaryModelId == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no ai_model created")
	}

	primaryModelKey := ""
	if len(modelKeys) > 0 {
		primaryModelKey = modelKeys[0]
	}
	alias := clientModelAlias(llm, primaryModelKey)
	binding := api.AiproxyInstanceBinding{
		LlmId:            llm.Id,
		ClientModelAlias: alias,
		AiProviderId:     providerId,
		AiProviderName:   providerName,
		BaseURL:          baseURL,
		SyncStatus:       api.AIPROXY_SYNC_STATUS_SYNCED,
	}
	return &binding, nil
}

func listRunningDeploymentLlms(depId string) ([]SLLM, error) {
	rows := make([]SLLM, 0, 8)
	q := GetLLMManager().Query().Equals("llm_deployment_id", depId).Equals("status", api.LLM_STATUS_RUNNING)
	if err := q.All(&rows); err != nil {
		return nil, errors.Wrap(err, "list running llms")
	}
	return rows, nil
}

func buildRoutingModelItems(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	dep *SLLMDeployment,
	llms []SLLM,
) ([]apapi.AiRoutingModelItem, []api.AiproxyInstanceBinding, error) {
	items := make([]apapi.AiRoutingModelItem, 0, len(llms))
	bindings := make([]api.AiproxyInstanceBinding, 0, len(llms))
	priority := 10
	for i := range llms {
		llm := &llms[i]
		binding, err := SyncLlmInstance(ctx, userCred, dep, llm)
		if err != nil {
			log.Warningf("SyncLlmInstance deployment=%s llm=%s: %v", dep.Name, llm.Id, err)
			bindings = append(bindings, api.AiproxyInstanceBinding{
				LlmId:      llm.Id,
				SyncStatus: api.AIPROXY_SYNC_STATUS_FAILED,
				LastError:  err.Error(),
			})
			continue
		}
		modelKeys, err := collectUpstreamModelKeys(ctx, userCred, llm)
		if err != nil {
			bindings = append(bindings, api.AiproxyInstanceBinding{
				LlmId:      llm.Id,
				SyncStatus: api.AIPROXY_SYNC_STATUS_FAILED,
				LastError:  err.Error(),
			})
			continue
		}
		filter := jsonutils.NewDict()
		filter.Set("ai_provider_id", jsonutils.NewString(binding.AiProviderId))
		filter.Set("model_key", jsonutils.NewString(modelKeys[0]))
		session := aiproxyAdminSession(ctx)
		rows, err := listAiproxyResources(session, &apmodules.AiModels, filter)
		if err != nil {
			return nil, nil, err
		}
		modelId := firstResourceID(rows)
		if modelId == "" {
			return nil, nil, errors.Wrapf(httperrors.ErrNotFound, "ai_model for llm %s", llm.Id)
		}
		enabled := true
		items = append(items, apapi.AiRoutingModelItem{
			AiProviderId: binding.AiProviderId,
			AiModelId:    modelId,
			Priority:     priority,
			ModelPattern: binding.ClientModelAlias,
			LlmId:        llm.Id,
			Enabled:      &enabled,
		})
		priority += 10
		bindings = append(bindings, *binding)
	}
	return items, bindings, nil
}

func applyRoutingModels(session *mcclient.ClientSession, routingId string, items []apapi.AiRoutingModelItem) error {
	params := jsonutils.NewDict()
	params.Set("models", jsonutils.Marshal(items))
	_, err := apmodules.AiRoutings.PerformAction(session, routingId, "set-models", params)
	if err != nil {
		return errors.Wrap(err, "ai_routing set-models")
	}
	return nil
}

func persistDeploymentAiproxyState(dep *SLLMDeployment, status, routingId string, bindings []api.AiproxyInstanceBinding) error {
	raw := jsonutils.Marshal(bindings)
	_, err := db.Update(dep, func() error {
		dep.AiproxySyncStatus = status
		dep.AiproxyRoutingId = routingId
		dep.AiproxyBindings = raw.String()
		return nil
	})
	return err
}

func parseDeploymentAiproxyBindings(dep *SLLMDeployment) []api.AiproxyInstanceBinding {
	raw := strings.TrimSpace(dep.AiproxyBindings)
	if raw == "" {
		return nil
	}
	bindings := make([]api.AiproxyInstanceBinding, 0, 4)
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		log.Warningf("parse aiproxy bindings for deployment %s: %v", dep.Id, err)
		return nil
	}
	if err := obj.Unmarshal(&bindings); err != nil {
		log.Warningf("unmarshal aiproxy bindings for deployment %s: %v", dep.Id, err)
		return nil
	}
	return bindings
}

func computeAiproxySyncStatus(bindings []api.AiproxyInstanceBinding, running int) string {
	if len(bindings) == 0 {
		if running > 0 {
			return api.AIPROXY_SYNC_STATUS_PENDING
		}
		return api.AIPROXY_SYNC_STATUS_PENDING
	}
	synced := 0
	failed := 0
	for i := range bindings {
		switch bindings[i].SyncStatus {
		case api.AIPROXY_SYNC_STATUS_SYNCED:
			synced++
		case api.AIPROXY_SYNC_STATUS_FAILED:
			failed++
		}
	}
	if failed > 0 && synced == 0 {
		return api.AIPROXY_SYNC_STATUS_FAILED
	}
	if synced < running || failed > 0 {
		return api.AIPROXY_SYNC_STATUS_PARTIAL
	}
	return api.AIPROXY_SYNC_STATUS_SYNCED
}

// ReconcileDeploymentAiproxy syncs all running replicas and refreshes routing bindings.
func ReconcileDeploymentAiproxy(ctx context.Context, userCred mcclient.TokenCredential, dep *SLLMDeployment) error {
	if dep == nil {
		return errors.Wrap(httperrors.ErrInvalidStatus, "nil deployment")
	}
	if !dep.AutoRegisterAiproxy {
		return nil
	}

	if _, err := db.Update(dep, func() error {
		dep.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_SYNCING
		return nil
	}); err != nil {
		return errors.Wrap(err, "set aiproxy syncing status")
	}

	llms, err := listRunningDeploymentLlms(dep.Id)
	if err != nil {
		return err
	}

	session := aiproxyAdminSession(ctx)
	routingName := aiRoutingNameForDeployment(dep.Id)
	routingId, err := ensureAiRouting(session, routingName, dep.Id, dep)
	if err != nil {
		_ = persistDeploymentAiproxyState(dep, api.AIPROXY_SYNC_STATUS_FAILED, dep.AiproxyRoutingId, parseDeploymentAiproxyBindings(dep))
		return err
	}

	items, bindings, err := buildRoutingModelItems(ctx, userCred, dep, llms)
	if err != nil {
		_ = persistDeploymentAiproxyState(dep, api.AIPROXY_SYNC_STATUS_FAILED, routingId, bindings)
		return err
	}
	if err := applyRoutingModels(session, routingId, items); err != nil {
		_ = persistDeploymentAiproxyState(dep, api.AIPROXY_SYNC_STATUS_FAILED, routingId, bindings)
		return err
	}

	status := computeAiproxySyncStatus(bindings, len(llms))
	return persistDeploymentAiproxyState(dep, status, routingId, bindings)
}

func deleteAiProviderById(session *mcclient.ClientSession, providerId string) error {
	providerId = strings.TrimSpace(providerId)
	if providerId == "" {
		return nil
	}
	filterModels := jsonutils.NewDict()
	filterModels.Set("ai_provider_id", jsonutils.NewString(providerId))
	modelRows, err := listAiproxyResources(session, &apmodules.AiModels, filterModels)
	if err != nil {
		return errors.Wrap(err, "list ai_models for delete")
	}
	for _, mrow := range modelRows {
		mid, _ := mrow.GetString("id")
		if mid != "" {
			if _, err := apmodules.AiModels.Delete(session, mid, nil); err != nil {
				log.Warningf("delete ai_model %s: %v", mid, err)
			}
		}
	}
	if _, err := apmodules.AiProviders.Delete(session, providerId, nil); err != nil {
		return errors.Wrapf(err, "delete ai_provider %s", providerId)
	}
	return nil
}

func deleteAiProviderByLlmId(session *mcclient.ClientSession, llmId string) error {
	filter := jsonutils.NewDict()
	filter.Set("llm_id", jsonutils.NewString(llmId))
	rows, err := listAiproxyResources(session, &apmodules.AiProviders, filter)
	if err != nil {
		return errors.Wrap(err, "list ai_providers for delete")
	}
	for _, row := range rows {
		id, _ := row.GetString("id")
		if err := deleteAiProviderById(session, id); err != nil {
			return err
		}
	}
	return nil
}

func deleteAiRoutingsByDeploymentId(session *mcclient.ClientSession, deploymentId string) error {
	filter := jsonutils.NewDict()
	filter.Set("llm_deployment_id", jsonutils.NewString(deploymentId))
	rows, err := listAiproxyResources(session, &apmodules.AiRoutings, filter)
	if err != nil {
		return errors.Wrap(err, "list ai_routings for delete")
	}
	for _, row := range rows {
		id, _ := row.GetString("id")
		if id == "" {
			continue
		}
		if _, err := apmodules.AiRoutings.Delete(session, id, nil); err != nil {
			log.Warningf("delete ai_routing %s: %v", id, err)
		}
	}
	return nil
}

// DeleteDeploymentAiproxyResources removes all aiproxy catalog/routing rows linked to a deployment.
func DeleteDeploymentAiproxyResources(ctx context.Context, deploymentId string) error {
	deploymentId = strings.TrimSpace(deploymentId)
	if deploymentId == "" {
		return nil
	}
	session := aiproxyAdminSession(ctx)

	filter := jsonutils.NewDict()
	filter.Set("llm_deployment_id", jsonutils.NewString(deploymentId))
	provRows, err := listAiproxyResources(session, &apmodules.AiProviders, filter)
	if err != nil {
		return errors.Wrap(err, "list ai_providers by llm_deployment_id")
	}
	for _, row := range provRows {
		id, _ := row.GetString("id")
		if err := deleteAiProviderById(session, id); err != nil {
			log.Warningf("delete ai_provider %s for deployment %s: %v", id, deploymentId, err)
		}
	}

	// Fallback: child llm instances may still have providers keyed only by llm_id.
	llmRows := make([]SLLM, 0, 8)
	q := GetLLMManager().Query("id").Equals("llm_deployment_id", deploymentId)
	if err := q.All(&llmRows); err != nil {
		return errors.Wrap(err, "list llms for aiproxy cleanup")
	}
	for i := range llmRows {
		if err := deleteAiProviderByLlmId(session, llmRows[i].Id); err != nil {
			log.Warningf("delete ai_provider for llm %s: %v", llmRows[i].Id, err)
		}
	}

	if err := deleteAiRoutingsByDeploymentId(session, deploymentId); err != nil {
		return err
	}
	return nil
}

// UnsyncLlmInstance removes aiproxy resources for one llm replica.
func UnsyncLlmInstance(ctx context.Context, userCred mcclient.TokenCredential, dep *SLLMDeployment, llmId string) error {
	if dep == nil || strings.TrimSpace(llmId) == "" {
		return nil
	}
	session := aiproxyAdminSession(ctx)
	if err := deleteAiProviderByLlmId(session, llmId); err != nil {
		return err
	}

	llms, err := listRunningDeploymentLlms(dep.Id)
	if err != nil {
		return err
	}
	remaining := make([]SLLM, 0, len(llms))
	for i := range llms {
		if llms[i].Id != llmId {
			remaining = append(remaining, llms[i])
		}
	}

	routingId := strings.TrimSpace(dep.AiproxyRoutingId)
	if routingId == "" {
		filter := jsonutils.NewDict()
		filter.Set("llm_deployment_id", jsonutils.NewString(dep.Id))
		rows, err := listAiproxyResources(session, &apmodules.AiRoutings, filter)
		if err != nil {
			return err
		}
		routingId = firstResourceID(rows)
	}

	if routingId == "" {
		return persistDeploymentAiproxyState(dep, api.AIPROXY_SYNC_STATUS_PENDING, "", nil)
	}

	if len(remaining) == 0 {
		if _, err := apmodules.AiRoutings.Delete(session, routingId, nil); err != nil {
			log.Warningf("delete ai_routing %s: %v", routingId, err)
		}
		return persistDeploymentAiproxyState(dep, api.AIPROXY_SYNC_STATUS_PENDING, "", nil)
	}

	items := make([]apapi.AiRoutingModelItem, 0, len(remaining))
	bindings := make([]api.AiproxyInstanceBinding, 0, len(remaining))
	priority := 10
	for i := range remaining {
		llm := &remaining[i]
		b, err := parseBindingForLlm(dep, llm.Id)
		if err != nil {
			continue
		}
		modelKeys, err := collectUpstreamModelKeys(ctx, userCred, llm)
		if err != nil {
			continue
		}
		filter := jsonutils.NewDict()
		filter.Set("ai_provider_id", jsonutils.NewString(b.AiProviderId))
		filter.Set("model_key", jsonutils.NewString(modelKeys[0]))
		rows, err := listAiproxyResources(session, &apmodules.AiModels, filter)
		if err != nil {
			return err
		}
		modelId := firstResourceID(rows)
		if modelId == "" {
			continue
		}
		enabled := true
		items = append(items, apapi.AiRoutingModelItem{
			AiProviderId: b.AiProviderId,
			AiModelId:    modelId,
			Priority:     priority,
			ModelPattern: b.ClientModelAlias,
			LlmId:        llm.Id,
			Enabled:      &enabled,
		})
		priority += 10
		bindings = append(bindings, b)
	}
	if err := applyRoutingModels(session, routingId, items); err != nil {
		return err
	}
	status := computeAiproxySyncStatus(bindings, len(remaining))
	return persistDeploymentAiproxyState(dep, status, routingId, bindings)
}

func parseBindingForLlm(dep *SLLMDeployment, llmId string) (api.AiproxyInstanceBinding, error) {
	for _, b := range parseDeploymentAiproxyBindings(dep) {
		if b.LlmId == llmId && b.SyncStatus == api.AIPROXY_SYNC_STATUS_SYNCED {
			return b, nil
		}
	}
	return api.AiproxyInstanceBinding{}, errors.Wrapf(httperrors.ErrNotFound, "binding for llm %s", llmId)
}

// UnregisterDeploymentAiproxy removes all aiproxy resources linked to a deployment.
func UnregisterDeploymentAiproxy(ctx context.Context, userCred mcclient.TokenCredential, dep *SLLMDeployment) error {
	if dep == nil {
		return nil
	}
	if err := DeleteDeploymentAiproxyResources(ctx, dep.Id); err != nil {
		return err
	}

	_, err := db.Update(dep, func() error {
		dep.AutoRegisterAiproxy = false
		dep.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_PENDING
		dep.AiproxyRoutingId = ""
		dep.AiproxyBindings = ""
		return nil
	})
	return err
}

func (dep *SLLMDeployment) StartAiproxySyncTask(ctx context.Context, userCred mcclient.TokenCredential, llmId string) error {
	params := jsonutils.NewDict()
	if llmId != "" {
		params.Set("llm_id", jsonutils.NewString(llmId))
	}
	return dep.startAiproxySyncTaskWithParams(ctx, userCred, params)
}

func (dep *SLLMDeployment) startAiproxySyncTaskWithParams(ctx context.Context, userCred mcclient.TokenCredential, params jsonutils.JSONObject) error {
	pdict, _ := params.(*jsonutils.JSONDict)
	if pdict == nil {
		pdict = jsonutils.NewDict()
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMAiproxySyncTask", dep, userCred, pdict, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMAiproxySyncTask")
	}
	return task.ScheduleRun(nil)
}
