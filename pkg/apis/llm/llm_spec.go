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

// LLMSpec is the flat spec for LLM SKU: optional ollama/vllm/dify payload. Type is on LLMSku.LLMType.
type LLMSpec struct {
	Ollama *LLMSpecOllama `json:"ollama,omitempty"`
	Vllm   *LLMSpecVllm   `json:"vllm,omitempty"`
	Dify   *LLMSpecDify   `json:"dify,omitempty"`
}

func (s *LLMSpec) String() string {
	return jsonutils.Marshal(s).String()
}

func (s *LLMSpec) IsZero() bool {
	if s == nil {
		return true
	}
	return s.Ollama == nil && s.Vllm == nil && s.Dify == nil
}

// UnmarshalJSON supports both new format (type + ollama/vllm/dify) and legacy format (type + data).
func (s *LLMSpec) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type   string          `json:"type"`
		Ollama *LLMSpecOllama  `json:"ollama,omitempty"`
		Vllm   *LLMSpecVllm    `json:"vllm,omitempty"`
		Dify   *LLMSpecDify    `json:"dify,omitempty"`
		Data   json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.Ollama = raw.Ollama
	s.Vllm = raw.Vllm
	s.Dify = raw.Dify
	if len(raw.Data) > 0 && s.Ollama == nil && s.Vllm == nil && s.Dify == nil {
		switch raw.Type {
		case string(LLM_CONTAINER_OLLAMA):
			s.Ollama = &LLMSpecOllama{}
			if err := json.Unmarshal(raw.Data, s.Ollama); err != nil {
				return err
			}
		case string(LLM_CONTAINER_VLLM):
			s.Vllm = &LLMSpecVllm{}
			if err := json.Unmarshal(raw.Data, s.Vllm); err != nil {
				return err
			}
		case string(LLM_CONTAINER_DIFY):
			s.Dify = &LLMSpecDify{}
			if err := json.Unmarshal(raw.Data, s.Dify); err != nil {
				return err
			}
		default:
			s.Ollama = &LLMSpecOllama{}
			_ = json.Unmarshal(raw.Data, s.Ollama)
		}
	}
	return nil
}

// LLMSpecOllama holds type-specific fields for ollama SKUs.
type LLMSpecOllama struct {
	LLMImageId    string   `json:"llm_image_id"`
	MountedModels []string `json:"mounted_models"`
}

func (s *LLMSpecOllama) String() string {
	return jsonutils.Marshal(s).String()
}

func (s *LLMSpecOllama) IsZero() bool {
	if s == nil {
		return true
	}
	return s.LLMImageId == "" && len(s.MountedModels) == 0
}

// LLMSpecVllm holds type-specific fields for vllm SKUs (includes PreferredModel).
type LLMSpecVllm struct {
	LLMImageId     string   `json:"llm_image_id"`
	MountedModels  []string `json:"mounted_models"`
	PreferredModel string   `json:"preferred_model"`
}

func (s *LLMSpecVllm) String() string {
	return jsonutils.Marshal(s).String()
}

func (s *LLMSpecVllm) IsZero() bool {
	if s == nil {
		return true
	}
	return s.LLMImageId == "" && len(s.MountedModels) == 0 && s.PreferredModel == ""
}

// LLMSpecDify holds type-specific fields for Dify SKUs (multiple image ids + customized envs).
type LLMSpecDify struct {
	PostgresImageId     string               `json:"postgres_image_id"`
	RedisImageId        string               `json:"redis_image_id"`
	NginxImageId        string               `json:"nginx_image_id"`
	DifyApiImageId      string               `json:"dify_api_image_id"`
	DifyPluginImageId   string               `json:"dify_plugin_image_id"`
	DifyWebImageId      string               `json:"dify_web_image_id"`
	DifySandboxImageId  string               `json:"dify_sandbox_image_id"`
	DifySSRFImageId     string               `json:"dify_ssrf_image_id"`
	DifyWeaviateImageId string               `json:"dify_weaviate_image_id"`
	CustomizedEnvs      []*DifyCustomizedEnv `json:"customized_envs,omitempty"`
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
		s.DifySandboxImageId == "" && s.DifySSRFImageId == "" && s.DifyWeaviateImageId == "" &&
		len(s.CustomizedEnvs) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(new(LLMSpec)), func() gotypes.ISerializable {
		return new(LLMSpec)
	})
}
