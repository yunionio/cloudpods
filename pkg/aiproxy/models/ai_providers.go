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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiProvider stores an LLM provider catalog entry (routing key and OpenAI-compatible config).
type SAiProvider struct {
	db.SEnabledStatusStandaloneResourceBase

	// ProviderKey selects the upstream adapter implementation (e.g. openai, vllm, aliyun).
	// Multiple ai_provider rows may share the same provider_key with different config.
	ProviderKey string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// Config is a JSON snapshot of provider connectivity (base_url, optional api_key).
	Config *api.SAiProviderConfig `length:"long" charset:"utf8" list:"user" create:"optional" update:"user"`
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
	if err := validateAiProviderConfig(input.Config); err != nil {
		return input, err
	}

	if strings.TrimSpace(input.Name) == "" {
		input.Name = pk
	}

	return input, nil
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
		if err := validateAiProviderConfig(input.Config); err != nil {
			return input, err
		}
	}

	return input, nil
}
