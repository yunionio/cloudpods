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
	"sync"
	"time"
)

const (
	aiKeyHealthMaxScore       = 100
	aiKeyHealthFailPenalty    = 25
	aiKeyHealthSuccessBoost   = 10
	aiKeyHealthCooldownAfter  = 3
	aiKeyHealthCooldownPeriod = 60 * time.Second
)

type aiKeyHealthState struct {
	score            int
	consecutiveFails int
	cooldownUntil    time.Time
}

var (
	aiKeyHealthMu sync.RWMutex
	aiKeyHealth   = map[string]*aiKeyHealthState{}
)

func getAiKeyHealth(keyId string) *aiKeyHealthState {
	if keyId == "" {
		return nil
	}
	aiKeyHealthMu.RLock()
	st := aiKeyHealth[keyId]
	aiKeyHealthMu.RUnlock()
	if st != nil {
		return st
	}
	aiKeyHealthMu.Lock()
	defer aiKeyHealthMu.Unlock()
	if st = aiKeyHealth[keyId]; st == nil {
		st = &aiKeyHealthState{score: aiKeyHealthMaxScore}
		aiKeyHealth[keyId] = st
	}
	return st
}

// dynamicAiKeyWeightMultiplier returns 0-100 applied to configured ai_key.weight (100 = full weight).
func dynamicAiKeyWeightMultiplier(keyId string) int {
	if keyId == "" {
		return aiKeyHealthMaxScore
	}
	st := getAiKeyHealth(keyId)
	now := time.Now()
	aiKeyHealthMu.Lock()
	defer aiKeyHealthMu.Unlock()
	if !st.cooldownUntil.IsZero() && now.Before(st.cooldownUntil) {
		return 0
	}
	if !st.cooldownUntil.IsZero() && !now.Before(st.cooldownUntil) {
		st.cooldownUntil = time.Time{}
		if st.score < aiKeyHealthMaxScore/2 {
			st.score = aiKeyHealthMaxScore / 2
		}
	}
	if st.score <= 0 {
		return 0
	}
	if st.score > aiKeyHealthMaxScore {
		return aiKeyHealthMaxScore
	}
	return st.score
}

// RecordAiKeySuccess boosts dynamic weight after a successful upstream call.
func RecordAiKeySuccess(keyId string) {
	if keyId == "" {
		return
	}
	st := getAiKeyHealth(keyId)
	aiKeyHealthMu.Lock()
	defer aiKeyHealthMu.Unlock()
	st.consecutiveFails = 0
	st.cooldownUntil = time.Time{}
	st.score += aiKeyHealthSuccessBoost
	if st.score > aiKeyHealthMaxScore {
		st.score = aiKeyHealthMaxScore
	}
}

// RecordAiKeyFailure reduces dynamic weight when upstream rejects a key (429/401/5xx etc.).
func RecordAiKeyFailure(keyId string, statusCode int) {
	if keyId == "" || !IsRetryableAiKeyUpstreamStatus(statusCode) {
		return
	}
	st := getAiKeyHealth(keyId)
	aiKeyHealthMu.Lock()
	defer aiKeyHealthMu.Unlock()
	st.consecutiveFails++
	st.score -= aiKeyHealthFailPenalty
	if st.score < 0 {
		st.score = 0
	}
	if st.consecutiveFails >= aiKeyHealthCooldownAfter {
		st.cooldownUntil = time.Now().Add(aiKeyHealthCooldownPeriod)
		st.score = 0
	}
}

// IsRetryableAiKeyUpstreamStatus reports HTTP statuses that imply the api_key may be bad or overloaded.
func IsRetryableAiKeyUpstreamStatus(statusCode int) bool {
	if statusCode <= 0 {
		return true
	}
	switch {
	case statusCode == 401, statusCode == 403, statusCode == 429:
		return true
	case statusCode >= 500 && statusCode <= 599:
		return true
	default:
		return false
	}
}
