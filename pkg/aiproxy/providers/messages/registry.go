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

package messages

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

var (
	passthrough = passthroughAdapter{}
)

// GetAdapter returns the MessagesAdapter for a resolved catalog provider_key and api_mode.
func GetAdapter(providerKey, apiMode string) (providerapi.MessagesAdapter, error) {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	mode := strings.ToLower(strings.TrimSpace(apiMode))
	if mode == "" {
		mode = api.ProviderAPIModeOpenAI
	}
	if key == api.ProviderKeyAnthropic {
		return passthrough, nil
	}
	if api.IsNativeMessagesAdapterProvider(key) {
		return nil, fmt.Errorf("provider %q does not support anthropic messages API", providerKey)
	}
	if mode == api.ProviderAPIModeAnthropic && api.SupportsDualAPIMode(key) {
		return passthrough, nil
	}
	return translationAdapter{providerKey: providerKey}, nil
}
