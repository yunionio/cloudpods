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

package codexconfig

import (
	_ "embed"
	"strings"
)

//go:embed default_instructions.txt
var defaultPromptTemplate string

// defaultBaseInstructions returns the default base_instructions for a model.
func defaultBaseInstructions(slug string) string {
	modelName := extractModelName(slug)
	return strings.ReplaceAll(defaultPromptTemplate, "{{MODEL_NAME}}", modelName)
}

// extractModelName strips the provider suffix from a slug like "gpt-5.5(openai)".
func extractModelName(slug string) string {
	if idx := strings.Index(slug, "("); idx > 0 {
		return strings.TrimSpace(slug[:idx])
	}
	return slug
}
