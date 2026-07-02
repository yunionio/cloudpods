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
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/aiproxy/ft"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	providerCreateConnectivityTimeout = 15 * time.Second
	providerTestConnectivityTimeout   = 60 * time.Second
)

func normalizeProviderModelKeys(modelKeys []string) ([]string, error) {
	seen := make(map[string]struct{}, len(modelKeys))
	out := make([]string, 0, len(modelKeys))
	for _, key := range modelKeys {
		mk, err := validateAiModelKey(key)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[mk]; ok {
			continue
		}
		seen[mk] = struct{}{}
		out = append(out, mk)
	}
	return out, nil
}

func catalogModelKeysForConnectivity(providerKey string) []string {
	entries := catalogSeedModelsForProvider(providerKey)
	if len(entries) == 0 {
		return []string{placeholderCatalogModelKey}
	}
	keys := make([]string, len(entries))
	for i := range entries {
		keys[i] = entries[i].ModelKey
	}
	return keys
}

func probeModelForConnectivity(providerKey string) string {
	entries := catalogSeedModelsForProvider(providerKey)
	if len(entries) > 0 && strings.TrimSpace(entries[0].ModelKey) != "" {
		return entries[0].ModelKey
	}
	if model := strings.TrimSpace(ft.DefaultModelForProvider(providerKey)); model != "" {
		return model
	}
	return placeholderCatalogModelKey
}

func shouldFallbackToChatFromListModels(uerr *upstream.Error) bool {
	if uerr == nil {
		return true
	}
	switch uerr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return false
	default:
		return true
	}
}

func probeChatConnectivity(ctx context.Context, providerKey, secret string, cfg *api.SAiProviderConfig) error {
	if cfg == nil {
		cfg = &api.SAiProviderConfig{}
	}
	effectiveURL := cfg.EffectiveBaseURL(providerKey)
	if effectiveURL == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "config.base_url is required (no default for this provider_key)")
	}
	apiMode := cfg.ResolvedAPIMode()
	probeModel := probeModelForConnectivity(providerKey)

	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("ping"))
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString(probeModel))
	body.Set("max_tokens", jsonutils.NewInt(1))
	body.Set("messages", jsonutils.NewArray(userMsg))

	prov := providers.ChatProviderForUpstream(providerKey, apiMode)
	httpReq, err := prov.BuildUpstreamRequest(providers.ChatContextFromUpstream(
		providerKey, effectiveURL, secret, probeModel, apiMode,
	), body, false)
	if err != nil {
		return httperrors.NewInputParameterError("failed to build chat probe request: %s", err.Error())
	}
	_, uerr := upstream.ChatCompletion(ctx, providers.ToUpstreamRequest(httpReq, secret))
	if uerr != nil {
		return connectivityErrorFromUpstream(uerr)
	}
	return nil
}

func listProviderModels(ctx context.Context, providerKey, secret string, cfg *api.SAiProviderConfig, timeout time.Duration) ([]string, bool, error) {
	pk, err := validateAiCatalogIdentifier("provider_key", providerKey, maxAiProviderKeyLen)
	if err != nil {
		return nil, false, err
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, false, errors.Wrap(httperrors.ErrInputParameter, "secret is required for connectivity test")
	}
	cfg = normalizeAiProviderConfig(cfg)
	if err := validateAiProviderConfig(cfg, pk); err != nil {
		return nil, false, err
	}
	if cfg == nil {
		cfg = &api.SAiProviderConfig{}
	}
	effectiveURL := cfg.EffectiveBaseURL(pk)
	if effectiveURL == "" {
		return nil, false, errors.Wrap(httperrors.ErrInputParameter, "config.base_url is required (no default for this provider_key)")
	}
	if timeout <= 0 {
		timeout = providerCreateConnectivityTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, uerr := upstream.ListModels(ctx, effectiveURL, secret)
	if uerr == nil {
		modelKeys, err := upstream.ParseModelsListBody(resp.Body)
		if err == nil && len(modelKeys) > 0 {
			return modelKeys, false, nil
		}
	} else if !shouldFallbackToChatFromListModels(uerr) {
		return nil, false, connectivityErrorFromUpstream(uerr)
	}

	if err := probeChatConnectivity(ctx, pk, secret, cfg); err != nil {
		return nil, false, err
	}
	return catalogModelKeysForConnectivity(pk), true, nil
}

func probeProviderConnectivity(ctx context.Context, providerKey, secret string, cfg *api.SAiProviderConfig) error {
	_, _, err := listProviderModels(ctx, providerKey, secret, cfg, providerCreateConnectivityTimeout)
	return err
}

func connectivityErrorFromUpstream(uerr *upstream.Error) error {
	if uerr == nil {
		return nil
	}
	msg := strings.TrimSpace(uerr.Message)
	if msg == "" {
		msg = uerr.Error()
	}
	switch uerr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return httperrors.NewInputParameterError("invalid API key: %s", msg)
	case http.StatusNotFound:
		return httperrors.NewInputParameterError("API URL not found (check config.base_url): %s", msg)
	default:
		return httperrors.NewInputParameterError("upstream connectivity test failed: %s", msg)
	}
}

func providerUpstreamModels(modelKeys []string) []api.AiProviderUpstreamModel {
	out := make([]api.AiProviderUpstreamModel, len(modelKeys))
	for i, mk := range modelKeys {
		out[i] = api.AiProviderUpstreamModel{ModelKey: mk}
	}
	return out
}

// PerformTestConnectivity probes upstream list-models without persisting an ai_provider row.
func (manager *SAiProviderManager) PerformTestConnectivity(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.AiProviderTestConnectivityInput,
) (api.AiProviderTestConnectivityOutput, error) {
	out := api.AiProviderTestConnectivityOutput{}
	modelKeys, fromCatalog, err := listProviderModels(ctx, input.ProviderKey, input.Secret, input.Config, providerTestConnectivityTimeout)
	if err != nil {
		return out, err
	}
	out.Ok = true
	if fromCatalog {
		out.Message = "connectivity test passed (catalog models)"
		out.ModelsSource = api.AiProviderModelsSourceCatalog
	} else {
		out.Message = "connectivity test passed"
		out.ModelsSource = api.AiProviderModelsSourceUpstream
	}
	out.Models = providerUpstreamModels(modelKeys)
	return out, nil
}
