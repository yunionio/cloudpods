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

package llm

import (
	"encoding/json"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

// ILLMSpec is the interface for type-specific LLM SKU spec (LLM or Dify).
// It extends gotypes.ISerializable for DB/JSON storage.
type ILLMSpec interface {
	gotypes.ISerializable
	GetLLMType() string
}

// LLMSpecLLM holds type-specific fields for ollama/vllm SKUs.
type LLMSpecLLM struct {
	Type          string   `json:"type"` // ollama or vllm
	LLMImageId    string   `json:"llm_image_id"`
	MountedModels []string `json:"mounted_models"`
}

func (s *LLMSpecLLM) String() string {
	return jsonutils.Marshal(s).String()
}

func (s *LLMSpecLLM) IsZero() bool {
	if s == nil {
		return true
	}
	return s.Type == "" && s.LLMImageId == "" && len(s.MountedModels) == 0
}

func (s *LLMSpecLLM) GetLLMType() string {
	if s != nil && s.Type != "" {
		return s.Type
	}
	return string(LLM_CONTAINER_VLLM)
}

// LLMSpecDify holds type-specific fields for Dify SKUs (multiple image ids).
type LLMSpecDify struct {
	PostgresImageId     string `json:"postgres_image_id"`
	RedisImageId        string `json:"redis_image_id"`
	NginxImageId        string `json:"nginx_image_id"`
	DifyApiImageId      string `json:"dify_api_image_id"`
	DifyPluginImageId   string `json:"dify_plugin_image_id"`
	DifyWebImageId      string `json:"dify_web_image_id"`
	DifySandboxImageId  string `json:"dify_sandbox_image_id"`
	DifySSRFImageId     string `json:"dify_ssrf_image_id"`
	DifyWeaviateImageId string `json:"dify_weaviate_image_id"`
}

func (s *LLMSpecDify) String() string {
	return jsonutils.Marshal(s).String()
}

func (s *LLMSpecDify) IsZero() bool {
	if s == nil {
		return true
	}
	return s.PostgresImageId == "" && s.RedisImageId == "" && s.NginxImageId == "" &&
		s.DifyApiImageId == "" && s.DifyPluginImageId == "" && s.DifyWebImageId == "" &&
		s.DifySandboxImageId == "" && s.DifySSRFImageId == "" && s.DifyWeaviateImageId == ""
}

func (s *LLMSpecDify) GetLLMType() string {
	return string(LLM_CONTAINER_DIFY)
}

// LLMSpecHolder wraps ILLMSpec for DB/JSON with type discriminator.
type LLMSpecHolder struct {
	Value ILLMSpec
}

func (h *LLMSpecHolder) String() string {
	if h == nil || h.Value == nil {
		return "{}"
	}
	b, _ := h.MarshalJSON()
	return string(b)
}

func (h *LLMSpecHolder) IsZero() bool {
	return h == nil || h.Value == nil
}

// MarshalJSON implements json.Marshaler for polymorphic serialization.
func (h *LLMSpecHolder) MarshalJSON() ([]byte, error) {
	if h == nil || h.Value == nil {
		return []byte("{}"), nil
	}
	jo := jsonutils.Marshal(h.Value)
	wrap := map[string]interface{}{
		"type": h.Value.GetLLMType(),
		"data": json.RawMessage([]byte(jo.String())),
	}
	return json.Marshal(wrap)
}

// UnmarshalJSON implements json.Unmarshaler for polymorphic deserialization.
func (h *LLMSpecHolder) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Type == "" && len(raw.Data) == 0 {
		h.Value = nil
		return nil
	}
	var spec ILLMSpec
	switch raw.Type {
	case string(LLM_CONTAINER_OLLAMA), string(LLM_CONTAINER_VLLM):
		spec = &LLMSpecLLM{}
	case string(LLM_CONTAINER_DIFY):
		spec = &LLMSpecDify{}
	default:
		// Try LLM as default for backward compat
		spec = &LLMSpecLLM{}
	}
	if len(raw.Data) > 0 {
		jo, err := jsonutils.Parse(raw.Data)
		if err != nil {
			return err
		}
		if err := jo.Unmarshal(spec); err != nil {
			return err
		}
	}
	h.Value = spec
	return nil
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&LLMSpecLLM{}), func() gotypes.ISerializable {
		return &LLMSpecLLM{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&LLMSpecDify{}), func() gotypes.ISerializable {
		return &LLMSpecDify{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&LLMSpecHolder{}), func() gotypes.ISerializable {
		return &LLMSpecHolder{}
	})
}
