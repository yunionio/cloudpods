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

	"yunion.io/x/jsonutils"
)

const (
	ToolVisualBrief = "visual_brief"
	ToolVisualQA    = "visual_qa"
)

func visualBriefSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_urls": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "HTTP(S) image URLs or data URLs only. Do not put attachment labels like Image #1 here.",
			},
			"image_refs": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Attached image labels such as Image #1.",
			},
			"images": map[string]interface{}{
				"type":        "array",
				"description": "Structured images with url or base64 data plus mime_type.",
				"items":       imageInputSchema(),
			},
			"context": map[string]interface{}{
				"type":        "string",
				"description": "User task context that helps decide which visual facts matter.",
			},
			"focus": map[string]interface{}{
				"type":        "string",
				"description": "Optional focus area such as UI layout, OCR, or chart details.",
			},
		},
	}
}

func visualQASchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "A specific visual question to answer or clarify.",
			},
			"image_urls": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"image_refs": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"images": map[string]interface{}{
				"type":  "array",
				"items": imageInputSchema(),
			},
			"prior_visual_context": map[string]interface{}{
				"type": "string",
			},
			"context": map[string]interface{}{
				"type": "string",
			},
			"conversation": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
		},
		"required": []string{"question"},
	}
}

func imageInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url":       map[string]interface{}{"type": "string"},
			"data":      map[string]interface{}{"type": "string"},
			"mime_type": map[string]interface{}{"type": "string"},
			"detail":    map[string]interface{}{"type": "string"},
		},
	}
}

// ChatTools returns OpenAI chat tool definitions for visual analysis.
func ChatTools() []*jsonutils.JSONDict {
	return []*jsonutils.JSONDict{
		chatTool(ToolVisualBrief, "Visual Brief. Use this as the first visual pass when image understanding is needed. For attached images, pass image_refs like Image #1 or omit image fields; pass image_urls only for real HTTP(S)/data URLs.", visualBriefSchema()),
		chatTool(ToolVisualQA, "Visual QA. Use this after a Visual Brief, or for targeted follow-up questions about an image.", visualQASchema()),
	}
}

func chatTool(name, description string, schema map[string]interface{}) *jsonutils.JSONDict {
	tool := jsonutils.NewDict()
	tool.Set("type", jsonutils.NewString("function"))
	fn := jsonutils.NewDict()
	fn.Set("name", jsonutils.NewString(name))
	fn.Set("description", jsonutils.NewString(description))
	schemaBytes, _ := json.Marshal(schema)
	params, _ := jsonutils.Parse(schemaBytes)
	fn.Set("parameters", params)
	tool.Set("function", fn)
	return tool
}

// IsVisualTool reports whether name is a visual extension tool.
func IsVisualTool(name string) bool {
	return name == ToolVisualBrief || name == ToolVisualQA
}

// InjectChatTools merges visual tools into a chat/completions request body.
func InjectChatTools(body *jsonutils.JSONDict) {
	if body == nil {
		return
	}
	existing := map[string]struct{}{}
	if toolsRaw, err := body.Get("tools"); err == nil {
		if arr, ok := toolsRaw.(*jsonutils.JSONArray); ok {
			for i := 0; i < arr.Size(); i++ {
				tool, _ := arr.GetAt(i)
				if d, ok := tool.(*jsonutils.JSONDict); ok {
					if fn, err := d.Get("function"); err == nil {
						if fnDict, ok := fn.(*jsonutils.JSONDict); ok {
							if name, _ := fnDict.GetString("name"); name != "" {
								existing[name] = struct{}{}
							}
						}
					}
				}
			}
		}
	}
	tools := jsonutils.NewArray()
	if toolsRaw, err := body.Get("tools"); err == nil {
		if arr, ok := toolsRaw.(*jsonutils.JSONArray); ok {
			for i := 0; i < arr.Size(); i++ {
				item, _ := arr.GetAt(i)
				tools.Add(item)
			}
		}
	}
	for _, tool := range ChatTools() {
		if fn, err := tool.Get("function"); err == nil {
			if fnDict, ok := fn.(*jsonutils.JSONDict); ok {
				if name, _ := fnDict.GetString("name"); name != "" {
					if _, ok := existing[name]; ok {
						continue
					}
				}
			}
		}
		tools.Add(tool)
	}
	if tools.Length() > 0 {
		body.Set("tools", tools)
	}
}
