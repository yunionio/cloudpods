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
	"strings"
	"testing"
	"time"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func resetAiKeyHealthForTest() {
	aiKeyHealthMu.Lock()
	aiKeyHealth = map[string]*aiKeyHealthState{}
	aiKeyHealthMu.Unlock()
}

func TestRecordAiKeyFailure_ConsecutiveTriggersCooldown(t *testing.T) {
	resetAiKeyHealthForTest()
	const id = "key-consec"
	for i := 0; i < aiKeyHealthCooldownAfter; i++ {
		RecordAiKeyFailure(id, 429)
	}
	if mul := dynamicAiKeyWeightMultiplier(id); mul != 0 {
		t.Fatalf("expected multiplier 0 during cooldown, got %d", mul)
	}
	info := aiKeyHealthInfo(id)
	if !info.inCooldown {
		t.Fatal("expected inCooldown after consecutive failures")
	}
	if info.score != 0 {
		t.Fatalf("expected score 0, got %d", info.score)
	}
}

func TestRecordAiKeyFailure_IntermittentScoreExhaustionEntersCooldown(t *testing.T) {
	resetAiKeyHealthForTest()
	const id = "key-intermittent"
	// fail -25 / success +10 with consecutiveFails reset: score can hit 0 without 3 consecutive fails.
	for {
		info := aiKeyHealthInfo(id)
		if info.score <= 0 || info.inCooldown {
			break
		}
		RecordAiKeyFailure(id, 500)
		info = aiKeyHealthInfo(id)
		if info.score <= 0 || info.inCooldown {
			break
		}
		RecordAiKeySuccess(id)
		if dynamicAiKeyWeightMultiplier(id) <= 0 {
			t.Fatal("unexpected zero multiplier after success")
		}
	}
	info := aiKeyHealthInfo(id)
	if !info.inCooldown {
		t.Fatalf("expected cooldown after score exhaustion, got score=%d inCooldown=%v", info.score, info.inCooldown)
	}
	if mul := dynamicAiKeyWeightMultiplier(id); mul != 0 {
		t.Fatalf("expected multiplier 0 during cooldown, got %d", mul)
	}
}

func TestDynamicAiKeyWeightMultiplier_RecoversAfterCooldown(t *testing.T) {
	resetAiKeyHealthForTest()
	const id = "key-recover"
	RecordAiKeyFailure(id, 429)
	RecordAiKeyFailure(id, 429)
	RecordAiKeyFailure(id, 429)
	st := getAiKeyHealth(id)
	aiKeyHealthMu.Lock()
	st.cooldownUntil = time.Now().Add(-time.Second)
	aiKeyHealthMu.Unlock()

	mul := dynamicAiKeyWeightMultiplier(id)
	if mul < aiKeyHealthMaxScore/2 {
		t.Fatalf("expected recovered multiplier >= %d, got %d", aiKeyHealthMaxScore/2, mul)
	}
	info := aiKeyHealthInfo(id)
	if info.inCooldown {
		t.Fatal("expected cooldown cleared after expiry")
	}
	if info.score < aiKeyHealthMaxScore/2 {
		t.Fatalf("expected score >= %d, got %d", aiKeyHealthMaxScore/2, info.score)
	}
}

func TestDynamicAiKeyWeightMultiplier_StuckScoreStartsCooldown(t *testing.T) {
	resetAiKeyHealthForTest()
	const id = "key-stuck"
	st := getAiKeyHealth(id)
	aiKeyHealthMu.Lock()
	st.score = 0
	st.cooldownUntil = time.Time{}
	aiKeyHealthMu.Unlock()

	if mul := dynamicAiKeyWeightMultiplier(id); mul != 0 {
		t.Fatalf("expected 0, got %d", mul)
	}
	info := aiKeyHealthInfo(id)
	if !info.inCooldown {
		t.Fatal("expected fallback cooldown for stuck score=0")
	}

	aiKeyHealthMu.Lock()
	st.cooldownUntil = time.Now().Add(-time.Second)
	aiKeyHealthMu.Unlock()
	mul := dynamicAiKeyWeightMultiplier(id)
	if mul < aiKeyHealthMaxScore/2 {
		t.Fatalf("expected recovery after fallback cooldown, got %d", mul)
	}
}

func TestRecordAiKeySuccess_ClearsCooldown(t *testing.T) {
	resetAiKeyHealthForTest()
	const id = "key-success"
	RecordAiKeyFailure(id, 401)
	RecordAiKeyFailure(id, 401)
	RecordAiKeyFailure(id, 401)
	RecordAiKeySuccess(id)
	info := aiKeyHealthInfo(id)
	if info.inCooldown {
		t.Fatal("success should clear cooldown")
	}
	if mul := dynamicAiKeyWeightMultiplier(id); mul <= 0 {
		t.Fatalf("expected positive multiplier after success, got %d", mul)
	}
}

func TestAiKeySkipReason(t *testing.T) {
	resetAiKeyHealthForTest()

	t.Run("already tried", func(t *testing.T) {
		k := &SAiKey{Secret: "sk-test"}
		k.Id = "id-tried"
		k.Name = "tried-key"
		reason := aiKeySkipReason(k, "deepseek-v4-pro", map[string]bool{"id-tried": true})
		if !strings.Contains(reason, "already tried") {
			t.Fatalf("got %q", reason)
		}
	})

	t.Run("cooldown", func(t *testing.T) {
		k := &SAiKey{Secret: "sk-test", Weight: 1}
		k.Id = "id-cd"
		k.Name = "cd-key"
		RecordAiKeyFailure(k.Id, 429)
		RecordAiKeyFailure(k.Id, 429)
		RecordAiKeyFailure(k.Id, 429)
		reason := aiKeySkipReason(k, "deepseek-v4-pro", nil)
		if !strings.Contains(reason, "cooldown") || !strings.Contains(reason, "remaining") {
			t.Fatalf("got %q", reason)
		}
	})

	t.Run("routing", func(t *testing.T) {
		resetAiKeyHealthForTest()
		k := &SAiKey{
			Secret: "sk-test",
			Weight: 1,
			Routing: &api.SAiKeyRouting{
				AllowedModelKeys: []string{"other-model"},
			},
		}
		k.Id = "id-route"
		k.Name = "route-key"
		reason := aiKeySkipReason(k, "deepseek-v4-pro", nil)
		if !strings.Contains(reason, "model not allowed by routing") {
			t.Fatalf("got %q", reason)
		}
	})

	t.Run("empty secret", func(t *testing.T) {
		k := &SAiKey{Secret: "  "}
		k.Name = "empty-key"
		reason := aiKeySkipReason(k, "m", nil)
		if !strings.Contains(reason, "empty secret") {
			t.Fatalf("got %q", reason)
		}
	})

	t.Run("usable", func(t *testing.T) {
		resetAiKeyHealthForTest()
		k := &SAiKey{Secret: "sk-ok", Weight: 1}
		k.Id = "id-ok"
		k.Name = "ok-key"
		if reason := aiKeySkipReason(k, "deepseek-v4-pro", nil); reason != "" {
			t.Fatalf("expected empty reason, got %q", reason)
		}
	})
}

func TestFormatAiKeySkipReasons_Truncates(t *testing.T) {
	reasons := make([]string, maxAiKeySkipReasonsInError+3)
	for i := range reasons {
		reasons[i] = "r"
	}
	out := formatAiKeySkipReasons(reasons)
	if !strings.Contains(out, "and 3 more") {
		t.Fatalf("got %q", out)
	}
}
