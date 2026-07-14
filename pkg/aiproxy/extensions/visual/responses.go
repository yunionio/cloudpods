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

package visual

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

// ShouldHandle reports whether the Responses visual orchestration path should run.
func ShouldHandle(dict *jsonutils.JSONDict, up *models.ChatUpstream, isStream bool) bool {
	if isStream || dict == nil || up == nil || !Enabled(up.ModelConfig) {
		return false
	}
	return openai.ResponsesInputHasImage(dict)
}

// HandleResponsesCreate runs the visual orchestration loop for a non-streaming Responses request.
func HandleResponsesCreate(
	ctx context.Context,
	dict *jsonutils.JSONDict,
	textUp *models.ChatUpstream,
) ([]byte, *openai.ResponsesConvertState, error) {
	runtime, visCfg := RuntimeConfigFromModel(textUp.ModelConfig)
	if visCfg == nil {
		return nil, nil, fmt.Errorf("visual config is missing")
	}
	userCred := auth.AdminCredential()
	vk, err := models.LoadEnabledVirtualKeyById(textUp.VirtualKeyId)
	if err != nil {
		return nil, nil, err
	}
	visUp, err := models.ResolveVisualUpstream(ctx, userCred, vk, visCfg)
	if err != nil {
		return nil, nil, err
	}
	chatBody, state, err := openai.ResponsesToChatCompletions(dict, textUp.UpstreamModel)
	if err != nil {
		return nil, nil, err
	}
	chatClone := cloneDict(chatBody)
	textUpChat := *textUp
	textUpChat.BaseURL = ChatBaseURL(textUp.BaseURL)
	visUpChat := *visUp
	visUpChat.BaseURL = ChatBaseURL(visUp.BaseURL)
	respBody, err := RunChatOrchestrator(ctx, &textUpChat, &visUpChat, chatClone, runtime)
	if err != nil {
		return nil, nil, err
	}
	out, err := openai.ChatCompletionToResponses(respBody, state)
	if err != nil {
		return nil, nil, err
	}
	return out, state, nil
}

func cloneDict(src *jsonutils.JSONDict) *jsonutils.JSONDict {
	if src == nil {
		return nil
	}
	parsed, err := jsonutils.Parse([]byte(src.String()))
	if err != nil {
		return src
	}
	if d, ok := parsed.(*jsonutils.JSONDict); ok {
		return d
	}
	return src
}
