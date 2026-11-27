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

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestAiModelConfigCategoryPattern(t *testing.T) {
	pattern, err := aiModelConfigCategoryPattern(" Embedding ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `"category":"embedding"`; pattern != want {
		t.Fatalf("pattern = %q, want %q", pattern, want)
	}

	// The pattern must actually match the serialized config written by
	// SAiModelConfig.String() (compact JSON).
	cfg := &api.SAiModelConfig{Category: api.ModelCategoryEmbedding, InputModalities: []string{"text"}}
	serialized := cfg.String()
	if !strings.Contains(serialized, pattern) {
		t.Fatalf("serialized config %q does not contain pattern %q", serialized, pattern)
	}
}

func TestAiModelConfigCategoryPatternRejectsUnknown(t *testing.T) {
	if _, err := aiModelConfigCategoryPattern("chat"); err == nil {
		t.Fatal("expected error for unknown category")
	}
}
