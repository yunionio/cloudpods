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

package aiproxy

import "testing"

func TestSAiModelConfigVisualEnabled(t *testing.T) {
	cfg := &SAiModelConfig{
		Extensions: &SAiModelExtensions{
			Visual: &SAiModelVisualConfig{Enabled: true},
		},
	}
	if !cfg.VisualEnabled() {
		t.Fatal("expected visual enabled")
	}
	mods := cfg.InputModalitiesForCatalog()
	if len(mods) != 2 || mods[1] != "image" {
		t.Fatalf("modalities = %#v", mods)
	}
	if cfg.IsZero() {
		t.Fatal("expected non-zero config")
	}
	if cfg.String() == "{}" {
		t.Fatal("expected non-empty String()")
	}
	var empty *SAiModelConfig
	if !empty.IsZero() {
		t.Fatal("nil config should be zero")
	}
}
