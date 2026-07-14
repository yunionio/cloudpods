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
	"encoding/json"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

func visualAttachmentText(index int) string {
	return fmt.Sprintf("[Image #%d is available to Visual Brief and Visual QA. Use image_refs [\"Image #%d\"] or omit image fields to analyze attached images.]", index, index)
}

// StripImagesFromChat removes image_url parts from chat messages and replaces them with placeholders.
func StripImagesFromChat(body *jsonutils.JSONDict) ([]ImageInput, error) {
	if body == nil {
		return nil, nil
	}
	msgsRaw, err := body.Get("messages")
	if err != nil {
		return nil, nil
	}
	arr, ok := msgsRaw.(*jsonutils.JSONArray)
	if !ok {
		return nil, nil
	}
	available := make([]ImageInput, 0, 2)
	for i := 0; i < arr.Size(); i++ {
		msgRaw, _ := arr.GetAt(i)
		msg, ok := msgRaw.(*jsonutils.JSONDict)
		if !ok {
			continue
		}
		contentRaw, err := msg.Get("content")
		if err != nil {
			continue
		}
		if contentArr, ok := contentRaw.(*jsonutils.JSONArray); ok {
			rewritten := jsonutils.NewArray()
			for j := 0; j < contentArr.Size(); j++ {
				partRaw, _ := contentArr.GetAt(j)
				part, ok := partRaw.(*jsonutils.JSONDict)
				if !ok {
					rewritten.Add(partRaw)
					continue
				}
				partType, _ := part.GetString("type")
				if partType == "image_url" {
					if image, ok := imageInputFromChatPart(part); ok {
						available = append(available, image)
						placeholder := jsonutils.NewDict()
						placeholder.Set("type", jsonutils.NewString("text"))
						placeholder.Set("text", jsonutils.NewString(visualAttachmentText(len(available))))
						rewritten.Add(placeholder)
						continue
					}
				}
				rewritten.Add(part)
			}
			msg.Set("content", rewritten)
			continue
		}
	}
	return available, nil
}

func imageInputFromChatPart(part *jsonutils.JSONDict) (ImageInput, bool) {
	if part == nil {
		return ImageInput{}, false
	}
	var wire struct {
		Type     string          `json:"type"`
		ImageURL json.RawMessage `json:"image_url"`
	}
	if err := json.Unmarshal([]byte(part.String()), &wire); err != nil {
		return ImageInput{}, false
	}
	if wire.Type != "image_url" {
		return ImageInput{}, false
	}
	var url string
	if err := json.Unmarshal(wire.ImageURL, &url); err == nil {
		url = strings.TrimSpace(url)
		if isSupportedImageURL(url) {
			return ImageInput{URL: url}, true
		}
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(wire.ImageURL, &obj); err == nil {
		url = strings.TrimSpace(obj.URL)
		if isSupportedImageURL(url) {
			return ImageInput{URL: url}, true
		}
	}
	return ImageInput{}, false
}

func isSupportedImageURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:")
}

func splitDataURL(value string) (mediaType, data string) {
	header, payload, ok := strings.Cut(value, ",")
	if !ok {
		return "", value
	}
	mediaType = strings.TrimPrefix(header, "data:")
	if semicolon := strings.IndexByte(mediaType, ';'); semicolon >= 0 {
		mediaType = mediaType[:semicolon]
	}
	if mediaType == "" {
		mediaType = "image/png"
	}
	return mediaType, payload
}

func chatImagePart(image ImageInput) *jsonutils.JSONDict {
	part := jsonutils.NewDict()
	part.Set("type", jsonutils.NewString("image_url"))
	imgURL := jsonutils.NewDict()
	if strings.TrimSpace(image.URL) != "" {
		imgURL.Set("url", jsonutils.NewString(strings.TrimSpace(image.URL)))
	} else if strings.TrimSpace(image.Data) != "" {
		data := strings.TrimSpace(image.Data)
		mime := strings.TrimSpace(image.MimeType)
		if mime == "" {
			mime = "image/png"
		}
		if strings.HasPrefix(data, "data:") {
			imgURL.Set("url", jsonutils.NewString(data))
		} else {
			imgURL.Set("url", jsonutils.NewString("data:"+mime+";base64,"+data))
		}
	} else {
		return nil
	}
	part.Set("image_url", imgURL)
	return part
}

func textFromChatResponse(body []byte) string {
	var resp struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Choices) == 0 {
		return ""
	}
	raw := resp.Choices[0].Message.Content
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(p.Text)
			}
		}
		return strings.TrimSpace(b.String())
	}
	return ""
}
