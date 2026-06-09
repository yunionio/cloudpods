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
	"crypto/rand"
	"math/big"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func effectiveAiKeyRoutingWeight(r *api.SAiKeyRouting) int {
	if r == nil || r.Weight <= 0 {
		return 0
	}
	return r.Weight
}

// baseAiKeyWeight returns configured weight (column, else routing.weight, else 1).
func baseAiKeyWeight(k *SAiKey) int {
	if k == nil {
		return 1
	}
	if k.Weight > 0 {
		return k.Weight
	}
	if w := effectiveAiKeyRoutingWeight(k.Routing); w > 0 {
		return w
	}
	return 1
}

// effectiveAiKeyWeight returns load-balance weight including dynamic penalty (差 key 降权).
func effectiveAiKeyWeight(k *SAiKey) int {
	base := baseAiKeyWeight(k)
	if k == nil || base <= 0 {
		return 0
	}
	mul := dynamicAiKeyWeightMultiplier(k.Id)
	if mul <= 0 {
		return 0
	}
	return base * mul / aiKeyHealthMaxScore
}

func aiKeyRoutingAcceptsModel(r *api.SAiKeyRouting, reqModel string) bool {
	rm := strings.TrimSpace(reqModel)
	if r == nil {
		return true
	}
	for _, block := range r.BlockedModelKeys {
		if modelPatternMatches(block, rm) {
			return false
		}
	}
	if len(r.AllowedModelKeys) > 0 {
		ok := false
		for _, allow := range r.AllowedModelKeys {
			if modelPatternMatches(allow, rm) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func pickWeightedAiKey(candidates []*SAiKey) *SAiKey {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	total := 0
	for _, k := range candidates {
		total += effectiveAiKeyWeight(k)
	}
	if total <= 0 {
		return candidates[0]
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return candidates[0]
	}
	threshold := int(n.Int64()) + 1
	acc := 0
	for _, k := range candidates {
		acc += effectiveAiKeyWeight(k)
		if acc >= threshold {
			return k
		}
	}
	return candidates[len(candidates)-1]
}

type resolvedUpstreamAPIKey struct {
	Secret   string
	AiKeyId  string
	FromRows bool
}

// MaxAiKeyFailoverAttempts is how many alternate ai_key rows to try per chat request.
const MaxAiKeyFailoverAttempts = 8

// resolveUpstreamAPIKey picks an ai_key (weighted + dynamic penalty) or provider.config api_key.
func resolveUpstreamAPIKey(prov *SAiProvider, modelKey string) (*resolvedUpstreamAPIKey, error) {
	return resolveUpstreamAPIKeyExcluding(prov, modelKey, nil)
}

func resolveUpstreamAPIKeyExcluding(prov *SAiProvider, modelKey string, exclude map[string]bool) (*resolvedUpstreamAPIKey, error) {
	if prov == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider is nil")
	}
	pid := strings.TrimSpace(prov.Id)
	if pid == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider id is empty")
	}

	keys := make([]SAiKey, 0, 32)
	q := AiKeyManager.Query().Equals("ai_provider_id", pid).Equals("enabled", true)
	err := q.All(&keys)
	if err != nil {
		return nil, errors.Wrap(err, "list ai_key for provider")
	}

	candidates := make([]*SAiKey, 0, len(keys))
	hasSecretKey := false
	for i := range keys {
		k := &keys[i]
		if strings.TrimSpace(k.Secret) == "" {
			continue
		}
		hasSecretKey = true
		if exclude != nil && exclude[k.Id] {
			continue
		}
		if effectiveAiKeyWeight(k) <= 0 {
			continue
		}
		if aiKeyRoutingAcceptsModel(k.Routing, modelKey) {
			candidates = append(candidates, k)
		}
	}
	if len(candidates) > 0 {
		chosen := pickWeightedAiKey(candidates)
		if chosen == nil {
			return nil, errors.Wrap(httperrors.ErrInvalidStatus, "failed to pick ai_key")
		}
		return &resolvedUpstreamAPIKey{
			Secret:   strings.TrimSpace(chosen.Secret),
			AiKeyId:  chosen.Id,
			FromRows: true,
		}, nil
	}
	if hasSecretKey {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "no available ai_key for catalog model %q (check weight, cooldown, allowed_model_keys)", modelKey)
	}
	if prov.Config == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider.config is empty")
	}
	apiKey := strings.TrimSpace(prov.Config.ResolvedAPIKey())
	if apiKey == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "set api_key on ai_provider or add an enabled ai_key with secret for this provider")
	}
	return &resolvedUpstreamAPIKey{Secret: apiKey}, nil
}

// RepickUpstreamAPIKey selects another ai_key for the same provider/model, excluding already tried ids.
func RepickUpstreamAPIKey(up *ChatUpstream, tried map[string]bool) error {
	if up == nil || strings.TrimSpace(up.AiProviderId) == "" {
		return errors.Wrap(httperrors.ErrInvalidStatus, "missing ai_provider on upstream")
	}
	provObj, err := AiProviderManager.FetchById(up.AiProviderId)
	if err != nil {
		return errors.Wrap(err, "fetch ai_provider for key repick")
	}
	prov := provObj.(*SAiProvider)
	resolved, err := resolveUpstreamAPIKeyExcluding(prov, up.UpstreamModel, tried)
	if err != nil {
		return err
	}
	up.APIKey = resolved.Secret
	up.AiKeyId = resolved.AiKeyId
	return nil
}
