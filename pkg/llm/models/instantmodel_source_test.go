package models

import (
	"testing"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestDefaultInstantModelSource(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", apis.InstantModelSourceHuggingFace},
		{"huggingface", apis.InstantModelSourceHuggingFace},
		{"HuggingFace", apis.InstantModelSourceHuggingFace},
		{"model_scope", apis.InstantModelSourceModelScope},
		{"MODEL_SCOPE", apis.InstantModelSourceModelScope},
	}
	for _, tc := range cases {
		if got := defaultInstantModelSource(tc.in); got != tc.want {
			t.Fatalf("defaultInstantModelSource(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
