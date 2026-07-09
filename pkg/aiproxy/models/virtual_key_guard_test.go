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
	"testing"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestEnsureResponsesMaxOutputTokens(t *testing.T) {
	body := jsonutils.NewDict()
	if err := EnsureResponsesMaxOutputTokens(body, nil); err != nil {
		t.Fatal(err)
	}
	if mt, _ := body.Int("max_output_tokens"); mt != defaultResponsesMaxOutputTokens {
		t.Fatalf("max_output_tokens = %d", mt)
	}
}

func TestEnforceVirtualKeyMaxOutputTokens(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("max_output_tokens", jsonutils.NewInt(9000))
	lim := &api.SAiVirtualKeyLimits{MaxTokensPerRequest: 8000}
	if err := EnforceVirtualKeyMaxTokens(body, lim); err == nil {
		t.Fatal("expected limit error")
	}
	body = jsonutils.NewDict()
	if err := EnforceVirtualKeyMaxTokens(body, lim); err != nil {
		t.Fatal(err)
	}
	if mt, _ := body.Int("max_output_tokens"); mt != 8000 {
		t.Fatalf("max_output_tokens = %d", mt)
	}
}
