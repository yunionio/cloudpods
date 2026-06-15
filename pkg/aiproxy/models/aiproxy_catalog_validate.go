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
	"net/url"
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	maxAiProviderKeyLen = 64
	maxAiModelKeyLen    = 256
)

var aiCatalogIdentifierRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validateAiCatalogIdentifier(field, value string, maxLen int) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "%s is required", field)
	}
	if len(v) > maxLen {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "%s too long (max %d)", field, maxLen)
	}
	if !aiCatalogIdentifierRe.MatchString(v) {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "%s must match [a-z0-9][a-z0-9_-]*", field)
	}
	return v, nil
}

func validateAiModelKey(modelKey string) (string, error) {
	key := strings.TrimSpace(modelKey)
	if key == "" {
		return "", errors.Wrap(httperrors.ErrInputParameter, "model_key is required")
	}
	if len(key) > maxAiModelKeyLen {
		return "", errors.Wrap(httperrors.ErrInputParameter, "model_key too long")
	}
	return key, nil
}

// catalogModelId returns a stable ai_model row id for catalog seed (readable when model_key is simple).
// Format: {provider_key}-{slug(model_key)}; falls back to GenId for path-like or overlong keys.
func catalogModelId(providerKey, modelKey string) string {
	pk := strings.ToLower(strings.TrimSpace(providerKey))
	mk := strings.TrimSpace(modelKey)
	if pk == "" || mk == "" {
		return stringutils2.GenId("aiproxy.ai_model", providerKey, modelKey)
	}
	slug := catalogModelKeySlug(mk)
	id := pk + "-" + slug
	const maxIdLen = 128
	if len(id) > maxIdLen {
		trim := maxIdLen - len(pk) - 1
		if trim > 0 {
			id = pk + "-" + slug[:trim]
		} else {
			id = pk[:maxIdLen]
		}
	}
	if aiCatalogIdentifierRe.MatchString(id) {
		return id
	}
	return stringutils2.GenId("aiproxy.ai_model", providerKey, modelKey)
}

func catalogModelKeySlug(modelKey string) string {
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
		case r == '-', r == '_', r == '.', r == '/':
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

func validateAiProviderConfig(cfg *api.SAiProviderConfig) error {
	if cfg == nil || cfg.IsZero() {
		return nil
	}
	baseURL := cfg.ResolvedBaseURL()
	if baseURL == "" {
		return nil
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return errors.Wrapf(httperrors.ErrInputParameter, "config.base_url: invalid URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Wrap(httperrors.ErrInputParameter, "config.base_url must use http or https scheme")
	}
	if strings.TrimSpace(u.Host) == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "config.base_url must include a host")
	}
	return nil
}

func normalizeAiProviderConfig(cfg *api.SAiProviderConfig) *api.SAiProviderConfig {
	if cfg == nil || cfg.IsZero() {
		return cfg
	}
	out := &api.SAiProviderConfig{}
	if base := cfg.ResolvedBaseURL(); base != "" {
		out.BaseURL = base
	}
	if key := cfg.ResolvedAPIKey(); key != "" {
		out.APIKey = key
	}
	return out
}

func ensureAiModelKeyUniquePerProvider(ctx context.Context, providerId, modelKey, excludeId string) error {
	q := AiModelManager.Query().Equals("ai_provider_id", providerId).Equals("model_key", modelKey)
	if excludeId != "" {
		q = q.NotEquals("id", excludeId)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "count ai_model by provider and model_key")
	}
	if cnt > 0 {
		return errors.Wrapf(httperrors.ErrConflict, "model_key %q already exists for ai_provider", modelKey)
	}
	return nil
}

func fetchEnabledAiProvider(ctx context.Context, userCred mcclient.TokenCredential, idOrName string) (*SAiProvider, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "ai_provider_id is required")
	}
	pObj, err := AiProviderManager.FetchByIdOrName(ctx, userCred, idOrName)
	if err != nil {
		return nil, errors.Wrap(err, "fetch ai_provider")
	}
	prov := pObj.(*SAiProvider)
	if !prov.GetEnabled() {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "ai_provider %q is disabled", idOrName)
	}
	return prov, nil
}

func defaultAiModelName(providerName, modelKey string) string {
	return catalogModelId(providerName, modelKey)
}

func ensureUniqueAiProviderLlmId(ctx context.Context, llmId, excludeId string) error {
	llmId = strings.TrimSpace(llmId)
	if llmId == "" {
		return nil
	}
	q := AiProviderManager.Query().Equals("llm_id", llmId)
	if excludeId != "" {
		q = q.NotEquals("id", excludeId)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "count ai_provider by llm_id")
	}
	if cnt > 0 {
		return errors.Wrapf(httperrors.ErrConflict, "ai_provider for llm_id %q already exists", llmId)
	}
	return nil
}

func ensureUniqueAiRoutingLlmDeploymentId(ctx context.Context, llmDeploymentId, excludeId string) error {
	llmDeploymentId = strings.TrimSpace(llmDeploymentId)
	if llmDeploymentId == "" {
		return nil
	}
	q := AiRoutingManager.Query().Equals("llm_deployment_id", llmDeploymentId)
	if excludeId != "" {
		q = q.NotEquals("id", excludeId)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "count ai_routing by llm_deployment_id")
	}
	if cnt > 0 {
		return errors.Wrapf(httperrors.ErrConflict, "ai_routing for llm_deployment_id %q already exists", llmDeploymentId)
	}
	return nil
}
