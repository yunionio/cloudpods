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

package aiproxy

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/util/sets"
)

// Model categories mirror pkg/apis/llm LLM_MODEL_CATEGORY_* values. They are
// duplicated here so aiproxy does not depend on the llm api package.
const (
	ModelCategoryLLM          = "llm"
	ModelCategoryEmbedding    = "embedding"
	ModelCategoryImage        = "image"
	ModelCategoryReranker     = "reranker"
	ModelCategorySpeechToText = "speech_to_text"
	ModelCategoryTextToSpeech = "text_to_speech"
)

var ModelCategories = sets.NewString(
	ModelCategoryLLM,
	ModelCategoryEmbedding,
	ModelCategoryImage,
	ModelCategoryReranker,
	ModelCategorySpeechToText,
	ModelCategoryTextToSpeech,
)

// NormalizeModelCategory lowercases and trims a model category. Unknown values
// are returned as-is (lowercased) so callers can validate separately.
func NormalizeModelCategory(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// IsKnownModelCategory reports whether raw (after normalization) is a supported category.
func IsKnownModelCategory(raw string) bool {
	c := NormalizeModelCategory(raw)
	return c != "" && ModelCategories.Has(c)
}

// ValidateModelCategory rejects non-empty values outside ModelCategories.
func ValidateModelCategory(raw string) error {
	c := NormalizeModelCategory(raw)
	if c == "" {
		return nil
	}
	if !ModelCategories.Has(c) {
		return fmt.Errorf("invalid category %q, must be one of %s", raw, strings.Join(ModelCategories.List(), ","))
	}
	return nil
}

// IsEmbeddingModelCategory reports whether the category serves the embeddings API.
func IsEmbeddingModelCategory(raw string) bool {
	return NormalizeModelCategory(raw) == ModelCategoryEmbedding
}

// IsRerankModelCategory reports whether the category serves the rerank API.
func IsRerankModelCategory(raw string) bool {
	return NormalizeModelCategory(raw) == ModelCategoryReranker
}

// IsChatModelCategory reports whether the category serves chat/completions-style APIs.
// An empty category is treated as chat for backward compatibility.
func IsChatModelCategory(raw string) bool {
	c := NormalizeModelCategory(raw)
	return c == "" || c == ModelCategoryLLM
}

// EndpointPathForCategory returns the OpenAI-compatible endpoint path that should
// be used for a model of the given category (empty when it is chat-like).
func EndpointPathForCategory(raw string) string {
	switch NormalizeModelCategory(raw) {
	case ModelCategoryEmbedding:
		return "/v1/embeddings"
	case ModelCategoryReranker:
		return "/v1/rerank"
	case ModelCategoryImage:
		return "/v1/images/generations"
	case ModelCategorySpeechToText:
		return "/v1/audio/transcriptions"
	case ModelCategoryTextToSpeech:
		return "/v1/audio/speech"
	default:
		return ""
	}
}
