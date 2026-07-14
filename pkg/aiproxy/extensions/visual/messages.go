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
	"encoding/json"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

var ErrVisualStreamingUnsupported = fmt.Errorf("visual extension does not support streaming")

// AnthropicMessagesHasImage reports whether an Anthropic Messages body carries image blocks.
func AnthropicMessagesHasImage(body *jsonutils.JSONDict) bool {
	if body == nil {
		return false
	}
	raw, err := body.Get("messages")
	if err != nil {
		return false
	}
	rawBytes := []byte(raw.String())
	var messages []struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(rawBytes, &messages) != nil {
		return false
	}
	for _, msg := range messages {
		if anthropicContentHasImage(msg.Content) {
			return true
		}
	}
	return false
}

func anthropicContentHasImage(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return false
	}
	for _, blk := range blocks {
		if strings.EqualFold(blk.Type, "image") {
			return true
		}
	}
	return false
}

// ShouldHandleMessages reports whether the Messages visual path should run.
// Streaming requests with images use non-streaming orchestration and synthetic SSE.
func ShouldHandleMessages(dict *jsonutils.JSONDict, up *models.ChatUpstream, isStream bool) bool {
	_ = isStream
	if dict == nil || up == nil || !Enabled(up) {
		return false
	}
	return AnthropicMessagesHasImage(dict)
}

// ShouldRejectMessagesStreaming is deprecated: stream+visual+image now uses synthetic SSE.
// Kept for compatibility; always returns false.
func ShouldRejectMessagesStreaming(dict *jsonutils.JSONDict, up *models.ChatUpstream, isStream bool) bool {
	_, _, _ = dict, up, isStream
	return false
}

// HandleMessagesCreateChat runs visual orchestration and returns the upstream chat completion body.
func HandleMessagesCreateChat(
	ctx context.Context,
	dict *jsonutils.JSONDict,
	textUp *models.ChatUpstream,
) ([]byte, error) {
	runtime, visCfg := RuntimeConfigFromModel(textUp.ModelConfig)
	if visCfg == nil || !visCfg.Enabled {
		return nil, fmt.Errorf("visual config is missing")
	}
	userCred := auth.AdminCredential()
	vk, err := models.LoadEnabledVirtualKeyById(textUp.VirtualKeyId)
	if err != nil {
		return nil, err
	}
	visUp, err := models.ResolveVisualUpstream(ctx, userCred, vk, textUp.VisualProviderId, textUp.VisualModelKey)
	if err != nil {
		return nil, err
	}
	chatBody, err := openai.AnthropicToChatCompletions(dict, textUp.UpstreamModel)
	if err != nil {
		return nil, err
	}
	chatClone := cloneDict(chatBody)
	forceNonStreamChatBody(chatClone)
	textUpChat := *textUp
	textUpChat.BaseURL = ChatBaseURL(textUp.BaseURL)
	visUpChat := *visUp
	visUpChat.BaseURL = ChatBaseURL(visUp.BaseURL)
	return RunChatOrchestrator(ctx, &textUpChat, &visUpChat, chatClone, runtime)
}

// HandleMessagesCreate runs visual orchestration for a non-streaming Anthropic Messages request.
func HandleMessagesCreate(
	ctx context.Context,
	dict *jsonutils.JSONDict,
	textUp *models.ChatUpstream,
) ([]byte, error) {
	respBody, err := HandleMessagesCreateChat(ctx, dict, textUp)
	if err != nil {
		return nil, err
	}
	return openai.ChatCompletionToAnthropic(respBody)
}
