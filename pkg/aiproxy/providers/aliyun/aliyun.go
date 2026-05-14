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

package aliyun

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

func patchEnableThinkingFalse(body *jsonutils.JSONDict, stream bool) {
	if stream {
		return
	}
	if _, err := body.Get("enable_thinking"); err == nil {
		return
	}
	body.Set("enable_thinking", jsonutils.JSONFalse)
}

// New returns the Aliyun (DashScope compatible-mode) provider adapter.
func New() providerapi.Provider {
	return openai.NewCompat("aliyun", patchEnableThinkingFalse)
}
