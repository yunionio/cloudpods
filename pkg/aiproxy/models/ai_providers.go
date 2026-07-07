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
	"fmt"
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

// SAiProvider stores an LLM provider catalog entry (routing key and OpenAI-compatible config).
type SAiProvider struct {
	db.SEnabledStatusStandaloneResourceBase

	// ProviderKey selects the upstream adapter implementation (e.g. openai, vllm, aliyun).
	// Multiple ai_provider rows may share the same provider_key with different config.
	ProviderKey string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// Config is a JSON snapshot of provider connectivity (base_url, api_mode).
	Config *api.SAiProviderConfig `length:"long" charset:"utf8" list:"user" create:"optional" update:"user"`
	// LlmDeploymentId and LlmId link this provider to an llm_deployment replica (set by llm sync).
	LlmDeploymentId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" index:"true"`
	LlmId           string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" index:"true"`
}

type SAiProviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var AiProviderManager *SAiProviderManager

func init() {
	AiProviderManager = &SAiProviderManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SAiProvider{},
			"ai_providers_tbl",
			"ai_provider",
			"ai_providers",
		),
	}
	AiProviderManager.SetVirtualObject(AiProviderManager)
}

func (manager *SAiProviderManager) InitializeData() error {
	return SeedStandardCatalog(context.Background())
}

func (manager *SAiProviderManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiProviderListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	if key := strings.TrimSpace(query.ProviderKey); key != "" {
		q = q.Equals("provider_key", key)
	}
	if v := strings.TrimSpace(query.LlmDeploymentId); v != "" {
		q = q.Equals("llm_deployment_id", v)
	}
	if v := strings.TrimSpace(query.LlmId); v != "" {
		q = q.Equals("llm_id", v)
	}
	return q, nil
}

func (manager *SAiProviderManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiProviderDetails {
	rows := make([]api.AiProviderDetails, len(objs))
	baseRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i].EnabledStatusStandaloneResourceDetails = baseRows[i]
		prov := objs[i].(*SAiProvider)
		rows[i].LlmDeploymentId = prov.LlmDeploymentId
		rows[i].LlmId = prov.LlmId
	}
	return rows
}

func (manager *SAiProviderManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiProviderCreateInput,
) (api.AiProviderCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}

	pk, err := validateAiCatalogIdentifier("provider_key", input.ProviderKey, maxAiProviderKeyLen)
	if err != nil {
		return input, err
	}
	input.ProviderKey = pk

	input.Config = normalizeAiProviderConfig(input.Config)
	if err := validateAiProviderConfig(input.Config, input.ProviderKey); err != nil {
		return input, err
	}

	input.Secret = strings.TrimSpace(input.Secret)

	if api.IsCustomProviderKey(input.ProviderKey) {
		if input.Secret == "" {
			return input, errors.Wrap(httperrors.ErrInputParameter, "secret is required for provider_key custom")
		}
		if input.Config == nil || strings.TrimSpace(input.Config.ResolvedBaseURL()) == "" {
			return input, errors.Wrap(httperrors.ErrInputParameter, "config.base_url is required for provider_key custom")
		}
	}

	if input.Enabled == nil && input.Disabled == nil {
		input.SetEnabled()
	}

	if strings.TrimSpace(input.Name) == "" {
		input.Name = pk
	}

	input.LlmDeploymentId = strings.TrimSpace(input.LlmDeploymentId)
	input.LlmId = strings.TrimSpace(input.LlmId)
	if err := ensureUniqueAiProviderLlmId(ctx, input.LlmId, ""); err != nil {
		return input, err
	}

	if strings.TrimSpace(input.Secret) != "" {
		var err error
		input.ModelKeys, err = normalizeProviderModelKeys(input.ModelKeys)
		if err != nil {
			return input, err
		}
		if len(input.ModelKeys) == 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "model_keys is required when secret is provided")
		}
		if err := probeProviderConnectivity(ctx, input.ProviderKey, input.Secret, input.Config); err != nil {
			return input, err
		}
	}

	return input, nil
}

func (p *SAiProvider) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	if err := rejectProviderConfigAPIKeyInJSON(data); err != nil {
		return err
	}
	return p.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (p *SAiProvider) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	p.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	input := api.AiProviderCreateInput{}
	if err := data.Unmarshal(&input); err != nil {
		log.Errorf("ai_provider PostCreate unmarshal: %v", err)
		return
	}
	if len(input.ModelKeys) > 0 {
		if err := createSelectedProviderModels(ctx, userCred, ownerId, p, input.ModelKeys); err != nil {
			log.Errorf("ai_provider %s create selected models: %v", p.Id, err)
		}
	} else if err := createCatalogModelsForUserProvider(ctx, userCred, ownerId, p); err != nil {
		log.Errorf("ai_provider %s create catalog models: %v", p.Id, err)
	}

	secret := strings.TrimSpace(input.Secret)
	if secret == "" {
		secret, _ = data.GetString("secret")
		secret = strings.TrimSpace(secret)
	}
	if secret == "" {
		return
	}
	if err := createInitialAiKey(ctx, userCred, ownerId, p, secret); err != nil {
		log.Errorf("ai_provider %s create initial ai_key: %v", p.Id, err)
	}
}

func createInitialAiKey(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	prov *SAiProvider,
	secret string,
) error {
	if prov == nil || strings.TrimSpace(prov.Id) == "" {
		return errors.Error("ai_provider is nil or has no id")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	dataDict := jsonutils.NewDict()
	dataDict.Set("ai_provider_id", jsonutils.NewString(prov.Id))
	dataDict.Set("secret", jsonutils.NewString(secret))
	dataDict.Set("weight", jsonutils.NewInt(1))
	dataDict.Set("enabled", jsonutils.JSONTrue)
	dataDict.Set("generate_name", jsonutils.NewString(fmt.Sprintf("%s-key", prov.Name)))
	if _, err := db.DoCreate(AiKeyManager, ctx, userCred, nil, dataDict, ownerId); err != nil {
		return errors.Wrap(err, "create ai_key for provider")
	}
	return nil
}

func (p *SAiProvider) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.AiProviderUpdateInput,
) (*api.AiProviderUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = p.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}

	if pk := strings.TrimSpace(input.ProviderKey); pk != "" {
		pk, err = validateAiCatalogIdentifier("provider_key", pk, maxAiProviderKeyLen)
		if err != nil {
			return input, err
		}
		input.ProviderKey = pk
	}

	if input.Config != nil {
		input.Config = normalizeAiProviderConfig(input.Config)
		pk := strings.TrimSpace(input.ProviderKey)
		if pk == "" {
			pk = p.ProviderKey
		}
		if err := validateAiProviderConfig(input.Config, pk); err != nil {
			return input, err
		}
	}

	if query.Contains("llm_deployment_id") {
		input.LlmDeploymentId = strings.TrimSpace(input.LlmDeploymentId)
	}
	if query.Contains("llm_id") {
		input.LlmId = strings.TrimSpace(input.LlmId)
		if err := ensureUniqueAiProviderLlmId(ctx, input.LlmId, p.Id); err != nil {
			return input, err
		}
	}

	return input, nil
}
