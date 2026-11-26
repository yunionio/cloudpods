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

package llm_test

import (
	"reflect"
	"testing"

	"yunion.io/x/onecloud/pkg/apis/llm"
)

func TestNormalizeLLMModelCategories(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{nil, []string{llm.LLM_MODEL_CATEGORY_LLM}},
		{[]string{"", "  "}, []string{llm.LLM_MODEL_CATEGORY_LLM}},
		{[]string{"Embedding"}, []string{llm.LLM_MODEL_CATEGORY_EMBEDDING}},
		{[]string{"embedding", "EMBEDDING", "reranker"}, []string{llm.LLM_MODEL_CATEGORY_EMBEDDING, llm.LLM_MODEL_CATEGORY_RERANKER}},
	}
	for _, c := range cases {
		if got := llm.NormalizeLLMModelCategories(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("llm.NormalizeLLMModelCategories(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidateLLMModelCategories(t *testing.T) {
	if err := llm.ValidateLLMModelCategories(nil); err != nil {
		t.Fatalf("nil categories: %v", err)
	}
	if err := llm.ValidateLLMModelCategories([]string{"embedding", "llm"}); err != nil {
		t.Fatalf("valid categories rejected: %v", err)
	}
	if err := llm.ValidateLLMModelCategories([]string{"chat"}); err == nil {
		t.Fatal("unknown category accepted")
	}
}

func TestCategoryPredicates(t *testing.T) {
	cats := []string{llm.LLM_MODEL_CATEGORY_EMBEDDING}
	if !llm.IsLLMEmbeddingCategory(cats) {
		t.Fatal("embedding not detected")
	}
	if llm.IsLLMRerankerCategory(cats) {
		t.Fatal("embedding misdetected as reranker")
	}
	if !llm.HasLLMModelCategory([]string{" RERANKER "}, llm.LLM_MODEL_CATEGORY_RERANKER) {
		t.Fatal("case/space insensitive match failed")
	}
	if llm.IsLLMEmbeddingCategory(nil) {
		t.Fatal("nil categories must not match")
	}
}
