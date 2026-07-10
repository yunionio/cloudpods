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

package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

// CodexToolKind identifies how a Responses tool was flattened for upstream.
type CodexToolKind string

const (
	CodexToolFunction    CodexToolKind = "function"
	CodexToolNestedOneOf CodexToolKind = "nested_oneof"
)

// CodexToolSpec records metadata for reversing namespace tool calls.
type CodexToolSpec struct {
	Kind       CodexToolKind `json:"kind"`
	OpenAIName string        `json:"openai_name,omitempty"`
	Namespace  string        `json:"namespace,omitempty"`
	Actions    []string      `json:"actions,omitempty"`
}

// CodexToolMap maps flattened upstream tool names to Codex metadata.
type CodexToolMap map[string]CodexToolSpec

// responsesTool is a Responses API tool definition.
type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
	Format      json.RawMessage `json:"format,omitempty"`
	Tools       []responsesTool `json:"tools,omitempty"`
}

// FlattenResponsesTools converts Responses tools (including namespace) to OpenAI chat tools.
func FlattenResponsesTools(raw *jsonutils.JSONArray) (*jsonutils.JSONArray, CodexToolMap, error) {
	if raw == nil || raw.Length() == 0 {
		return nil, nil, nil
	}
	var toolsIn []responsesTool
	if err := json.Unmarshal([]byte(raw.String()), &toolsIn); err != nil {
		return nil, nil, fmt.Errorf("invalid tools: %w", err)
	}
	toolMap := make(CodexToolMap)
	out := jsonutils.NewArray()
	for _, t := range toolsIn {
		flat, specs := flattenResponsesTool(t, "")
		for name, spec := range specs {
			toolMap[name] = spec
		}
		for _, ft := range flat {
			obj := jsonutils.NewDict()
			obj.Set("type", jsonutils.NewString("function"))
			obj.Set("function", ft)
			out.Add(obj)
		}
	}
	if out.Length() == 0 {
		return nil, nil, nil
	}
	return out, toolMap, nil
}

func flattenResponsesTool(tool responsesTool, parentNS string) ([]*jsonutils.JSONDict, CodexToolMap) {
	toolType := strings.TrimSpace(tool.Type)
	specs := make(CodexToolMap)
	switch toolType {
	case "namespace":
		ns := strings.TrimSpace(tool.Name)
		if ns == "" {
			ns = parentNS
		}
		subNames := make([]string, 0, len(tool.Tools))
		subMap := make(map[string]responsesTool)
		for _, sub := range tool.Tools {
			name := strings.TrimSpace(sub.Name)
			if name == "" {
				continue
			}
			subNames = append(subNames, name)
			subMap[name] = sub
		}
		if len(subNames) == 0 {
			return nil, specs
		}
		schema := buildNestedOneOfSchema(subNames, subMap)
		fn := jsonutils.NewDict()
		fn.Set("name", jsonutils.NewString(ns))
		if tool.Description != "" {
			fn.Set("description", jsonutils.NewString(tool.Description))
		} else {
			fn.Set("description", jsonutils.NewString(fmt.Sprintf("Namespace tool with %d sub-tools. Pick the matching action.", len(subNames))))
		}
		fn.Set("parameters", mustParseJSONObject(schema))
		specs[ns] = CodexToolSpec{
			Kind:       CodexToolNestedOneOf,
			OpenAIName: ns,
			Namespace:  ns,
			Actions:    subNames,
		}
		return []*jsonutils.JSONDict{fn}, specs
	case "function":
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			return nil, specs
		}
		if parentNS != "" {
			name = NamespacedToolName(parentNS, name)
		}
		fn := jsonutils.NewDict()
		fn.Set("name", jsonutils.NewString(name))
		if tool.Description != "" {
			fn.Set("description", jsonutils.NewString(tool.Description))
		}
		if len(tool.Parameters) > 0 {
			fn.Set("parameters", mustParseJSONObject(tool.Parameters))
		} else {
			fn.Set("parameters", mustParseJSONObject([]byte(`{"type":"object","properties":{}}`)))
		}
		specs[name] = CodexToolSpec{Kind: CodexToolFunction, OpenAIName: strings.TrimSpace(tool.Name)}
		return []*jsonutils.JSONDict{fn}, specs
	default:
		return nil, specs
	}
}

func buildNestedOneOfSchema(subNames []string, subMap map[string]responsesTool) map[string]interface{} {
	oneOf := make([]map[string]interface{}, 0, len(subNames))
	for _, name := range subNames {
		sub, ok := subMap[name]
		if !ok {
			continue
		}
		props := map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"enum": []string{name},
			},
		}
		required := []string{"action"}
		if len(sub.Parameters) > 0 {
			var schema map[string]interface{}
			if json.Unmarshal(sub.Parameters, &schema) == nil {
				if p, ok := schema["properties"].(map[string]interface{}); ok {
					for k, v := range p {
						props[k] = v
					}
				}
				if r, ok := schema["required"].([]interface{}); ok {
					for _, rv := range r {
						if rs, ok := rv.(string); ok {
							required = append(required, rs)
						}
					}
				}
			}
		}
		oneOf = append(oneOf, map[string]interface{}{
			"type":                 "object",
			"title":                name,
			"properties":           props,
			"required":             required,
			"additionalProperties": false,
		})
	}
	return map[string]interface{}{
		"type":  "object",
		"oneOf": oneOf,
	}
}

// NamespacedToolName joins namespace and tool name.
func NamespacedToolName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	if name == "" {
		return namespace
	}
	return namespace + "_" + name
}

// DecodeNamespaceToolCall splits a flattened namespace tool call back to Codex shape.
func DecodeNamespaceToolCall(toolMap CodexToolMap, toolName, args string) (namespace, name, arguments string) {
	spec, ok := toolMap[toolName]
	if !ok || spec.Kind != CodexToolNestedOneOf {
		return "", toolName, args
	}
	var parsed map[string]json.RawMessage
	if json.Unmarshal([]byte(args), &parsed) != nil {
		return spec.Namespace, toolName, args
	}
	actionRaw, ok := parsed["action"]
	if !ok {
		return spec.Namespace, toolName, args
	}
	var action string
	if json.Unmarshal(actionRaw, &action) != nil || action == "" {
		return spec.Namespace, toolName, args
	}
	rest := make(map[string]interface{})
	for k, v := range parsed {
		if k == "action" {
			continue
		}
		var val interface{}
		if json.Unmarshal(v, &val) == nil {
			rest[k] = val
		}
	}
	restBytes, _ := json.Marshal(rest)
	return spec.Namespace, action, string(restBytes)
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func mustParseJSONObject(v interface{}) jsonutils.JSONObject {
	var raw []byte
	switch x := v.(type) {
	case []byte:
		raw = x
	case json.RawMessage:
		raw = x
	default:
		raw = mustMarshal(v)
	}
	obj, err := jsonutils.Parse(raw)
	if err != nil {
		panic(fmt.Sprintf("parse parameters JSON: %v", err))
	}
	return obj
}
