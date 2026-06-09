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

package providers

import api "yunion.io/x/onecloud/pkg/aiproxy/providerapi"

// OpenAIStreamPassthrough reports whether upstream SSE chunks are already OpenAI-compatible.
func OpenAIStreamPassthrough(prov Provider, ctx *ChatContext) bool {
	if p, ok := prov.(api.ContextualStreamPassthrough); ok {
		return p.OpenAIStreamPassthroughForContext(ctx)
	}
	return prov.OpenAIStreamPassthrough()
}
