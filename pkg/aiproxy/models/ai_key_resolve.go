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
	"fmt"
	"math/big"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const maxAiKeySkipReasonsInError = 8

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
// When mul > 0 the result is at least 1 so weight=1 keys are not permanently excluded by
// integer truncation (e.g. 1*50/100=0 after cooldown recovery).
func effectiveAiKeyWeight(k *SAiKey) int {
	base := baseAiKeyWeight(k)
	if k == nil || base <= 0 {
		return 0
	}
	mul := dynamicAiKeyWeightMultiplier(k.Id)
	if mul <= 0 {
		return 0
	}
	w := base * mul / aiKeyHealthMaxScore
	if w < 1 {
		return 1
	}
	return w
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

func aiKeyLabel(k *SAiKey) string {
	if k == nil {
		return "?"
	}
	if n := strings.TrimSpace(k.Name); n != "" {
		return n
	}
	if id := strings.TrimSpace(k.Id); id != "" {
		return id
	}
	return "?"
}

// aiKeySkipReason returns why an ai_key cannot be used for modelKey, or "" if usable.
func aiKeySkipReason(k *SAiKey, modelKey string, exclude map[string]bool) string {
	if k == nil {
		return "?: nil ai_key"
	}
	label := aiKeyLabel(k)
	if strings.TrimSpace(k.Secret) == "" {
		return label + ": empty secret"
	}
	if exclude != nil && exclude[k.Id] {
		return label + ": already tried"
	}
	if baseAiKeyWeight(k) <= 0 {
		return label + ": weight=0"
	}
	if effectiveAiKeyWeight(k) <= 0 {
		info := aiKeyHealthInfo(k.Id)
		if info.inCooldown {
			return fmt.Sprintf("%s: cooldown %ds remaining (health_score=%d)", label, info.remainingSec, info.score)
		}
		return fmt.Sprintf("%s: health_score=%d", label, info.score)
	}
	if !aiKeyRoutingAcceptsModel(k.Routing, modelKey) {
		return label + ": model not allowed by routing (allowed_model_keys/blocked_model_keys)"
	}
	return ""
}

func formatAiKeySkipReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	if len(reasons) <= maxAiKeySkipReasonsInError {
		return strings.Join(reasons, "; ")
	}
	shown := strings.Join(reasons[:maxAiKeySkipReasonsInError], "; ")
	return fmt.Sprintf("%s; and %d more", shown, len(reasons)-maxAiKeySkipReasonsInError)
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

// resolveUpstreamAPIKey picks an enabled ai_key for the provider (weighted + dynamic penalty).
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
	skipReasons := make([]string, 0, len(keys))
	for i := range keys {
		k := &keys[i]
		if strings.TrimSpace(k.Secret) == "" {
			continue
		}
		hasSecretKey = true
		if reason := aiKeySkipReason(k, modelKey, exclude); reason != "" {
			skipReasons = append(skipReasons, reason)
			continue
		}
		candidates = append(candidates, k)
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
		detail := formatAiKeySkipReasons(skipReasons)
		if detail != "" {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "no available ai_key for catalog model %q: %s", modelKey, detail)
		}
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "no available ai_key for catalog model %q", modelKey)
	}
	return nil, errors.Wrap(httperrors.ErrInvalidStatus, "add an enabled ai_key with secret for this provider")
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
