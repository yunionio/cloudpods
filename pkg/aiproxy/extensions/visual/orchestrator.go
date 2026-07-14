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
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type briefInput struct {
	ImageURL  string       `json:"image_url,omitempty"`
	ImageURLs []string     `json:"image_urls,omitempty"`
	ImageRefs []string     `json:"image_refs,omitempty"`
	Images    []ImageInput `json:"images,omitempty"`
	Context   string       `json:"context,omitempty"`
	Focus     string       `json:"focus,omitempty"`
}

type qaInput struct {
	Question           string             `json:"question,omitempty"`
	ImageURL           string             `json:"image_url,omitempty"`
	ImageURLs          []string           `json:"image_urls,omitempty"`
	ImageRefs          []string           `json:"image_refs,omitempty"`
	Images             []ImageInput       `json:"images,omitempty"`
	PriorVisualContext string             `json:"prior_visual_context,omitempty"`
	Context            string             `json:"context,omitempty"`
	Conversation       []ConversationTurn `json:"conversation,omitempty"`
}

// RunChatOrchestrator executes the visual tool loop on a chat/completions request body.
func RunChatOrchestrator(
	ctx context.Context,
	textUp, visUp *models.ChatUpstream,
	body *jsonutils.JSONDict,
	runtime RuntimeConfig,
) ([]byte, error) {
	if textUp == nil || visUp == nil || body == nil {
		return nil, fmt.Errorf("visual orchestrator: missing upstream or body")
	}
	availableImages, err := StripImagesFromChat(body)
	if err != nil {
		return nil, err
	}
	InjectChatTools(body)
	vision := NewHTTPVisionClient(visUp, runtime.MaxTokens)
	maxRounds := runtime.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 4
	}
	for round := 0; round < maxRounds; round++ {
		respBody, err := callChatCompletions(ctx, textUp, body)
		if err != nil {
			return nil, err
		}
		finish := finishReasonFromChatResponse(respBody)
		if finish != "tool_calls" {
			return respBody, nil
		}
		toolCalls := toolCallsFromChatResponse(respBody)
		visualCalls, nonVisual := splitVisualToolCalls(toolCalls)
		if len(visualCalls) == 0 {
			return respBody, nil
		}
		assistantMsg := assistantMessageFromChatResponse(respBody)
		if assistantMsg == nil {
			return respBody, nil
		}
		msgsRaw, err := body.Get("messages")
		if err != nil {
			return nil, err
		}
		msgs, ok := msgsRaw.(*jsonutils.JSONArray)
		if !ok {
			return nil, fmt.Errorf("messages is not an array")
		}
		msgs.Add(assistantMsg)
		for _, tc := range visualCalls {
			result, err := executeVisualToolCall(ctx, vision, tc, availableImages)
			if err != nil {
				result = "Visual error: " + err.Error()
			}
			toolMsg := jsonutils.NewDict()
			toolMsg.Set("role", jsonutils.NewString("tool"))
			toolMsg.Set("tool_call_id", jsonutils.NewString(tc.ID))
			toolMsg.Set("content", jsonutils.NewString(result))
			msgs.Add(toolMsg)
		}
		body.Set("messages", msgs)
		if len(nonVisual) > 0 {
			continue
		}
	}
	return nil, fmt.Errorf("visual loop exceeded max rounds (%d)", maxRounds)
}

func splitVisualToolCalls(calls []openai.ToolCall) (visualCalls, nonVisual []openai.ToolCall) {
	for _, tc := range calls {
		if IsVisualTool(tc.Function.Name) {
			visualCalls = append(visualCalls, tc)
		} else {
			nonVisual = append(nonVisual, tc)
		}
	}
	return visualCalls, nonVisual
}

func executeVisualToolCall(ctx context.Context, client *HTTPVisionClient, tc openai.ToolCall, available []ImageInput) (string, error) {
	request, err := analysisRequestFromToolCall(tc, available)
	if err != nil {
		return "", err
	}
	result, err := client.Analyze(ctx, request)
	if err != nil {
		return "", err
	}
	switch tc.Function.Name {
	case ToolVisualBrief:
		return "Visual Brief result:\n" + strings.TrimSpace(result), nil
	case ToolVisualQA:
		return "Visual QA result:\n" + strings.TrimSpace(result), nil
	default:
		return strings.TrimSpace(result), nil
	}
}

func analysisRequestFromToolCall(tc openai.ToolCall, available []ImageInput) (AnalysisRequest, error) {
	switch tc.Function.Name {
	case ToolVisualBrief:
		var input briefInput
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			return AnalysisRequest{}, fmt.Errorf("parse visual_brief input: %w", err)
		}
		images := normalizeImages(input.ImageURL, input.ImageURLs, input.Images, input.ImageRefs, available)
		if len(images) == 0 {
			return AnalysisRequest{}, fmt.Errorf("visual_brief requires valid image URLs/data/images or attached images")
		}
		return AnalysisRequest{Tool: ToolVisualBrief, Prompt: buildBriefPrompt(input), Images: images}, nil
	case ToolVisualQA:
		var input qaInput
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			return AnalysisRequest{}, fmt.Errorf("parse visual_qa input: %w", err)
		}
		if strings.TrimSpace(input.Question) == "" {
			return AnalysisRequest{}, fmt.Errorf("visual_qa requires question")
		}
		return AnalysisRequest{
			Tool:   ToolVisualQA,
			Prompt: buildQAPrompt(input),
			Images: normalizeImages(input.ImageURL, input.ImageURLs, input.Images, input.ImageRefs, available),
		}, nil
	default:
		return AnalysisRequest{}, fmt.Errorf("unknown visual tool %q", tc.Function.Name)
	}
}

func buildBriefPrompt(input briefInput) string {
	var b strings.Builder
	b.WriteString("Provide a first-round visual brief for the main agent.\n")
	if strings.TrimSpace(input.Context) != "" {
		b.WriteString("\nTask context:\n")
		b.WriteString(strings.TrimSpace(input.Context))
		b.WriteByte('\n')
	}
	if strings.TrimSpace(input.Focus) != "" {
		b.WriteString("\nFocus:\n")
		b.WriteString(strings.TrimSpace(input.Focus))
		b.WriteByte('\n')
	}
	b.WriteString("\nReturn concise sections: overview, important visual details, any readable text/OCR, uncertainties, and useful Visual QA follow-ups.")
	return b.String()
}

func buildQAPrompt(input qaInput) string {
	var b strings.Builder
	b.WriteString("Answer this targeted visual clarification question for the main agent.\n\nQuestion:\n")
	b.WriteString(strings.TrimSpace(input.Question))
	b.WriteByte('\n')
	if strings.TrimSpace(input.Context) != "" {
		b.WriteString("\nTask context:\n")
		b.WriteString(strings.TrimSpace(input.Context))
		b.WriteByte('\n')
	}
	if strings.TrimSpace(input.PriorVisualContext) != "" {
		b.WriteString("\nPrior visual context:\n")
		b.WriteString(strings.TrimSpace(input.PriorVisualContext))
		b.WriteByte('\n')
	}
	b.WriteString("\nAnswer directly, call out uncertainty, and say what extra image/detail would resolve ambiguity.")
	return b.String()
}

func normalizeImages(single string, urls []string, images []ImageInput, refs []string, availableImages []ImageInput) []ImageInput {
	normalized := make([]ImageInput, 0, len(urls)+len(images)+len(refs)+1)
	if image, ok := resolveImageValue(single, availableImages); ok {
		normalized = append(normalized, image)
	}
	for _, url := range urls {
		if image, ok := resolveImageValue(url, availableImages); ok {
			normalized = append(normalized, image)
		}
	}
	for _, ref := range refs {
		if image, ok := resolveAttachedImage(ref, availableImages); ok {
			normalized = append(normalized, image)
		}
	}
	for _, image := range images {
		if strings.TrimSpace(image.URL) != "" {
			if resolved, ok := resolveImageValue(image.URL, availableImages); ok {
				if resolved.Detail == "" {
					resolved.Detail = image.Detail
				}
				normalized = append(normalized, resolved)
			}
			continue
		}
		if strings.TrimSpace(image.Data) != "" || isSupportedImageURL(image.URL) {
			normalized = append(normalized, image)
		}
	}
	if len(normalized) == 0 && len(availableImages) > 0 {
		normalized = append(normalized, availableImages...)
	}
	return normalized
}

func resolveImageValue(value string, availableImages []ImageInput) (ImageInput, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ImageInput{}, false
	}
	if isSupportedImageURL(value) {
		return ImageInput{URL: value}, true
	}
	return resolveAttachedImage(value, availableImages)
}

func resolveAttachedImage(value string, availableImages []ImageInput) (ImageInput, bool) {
	index, ok := imageReferenceIndex(value)
	if !ok || index <= 0 || index > len(availableImages) {
		return ImageInput{}, false
	}
	return availableImages[index-1], true
}

var imageReferencePattern = regexp.MustCompile(`(?i)\bimage\s*#\s*(\d+)\b`)

func imageReferenceIndex(value string) (int, bool) {
	match := imageReferencePattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) != 2 {
		return 0, false
	}
	index, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return index, true
}
