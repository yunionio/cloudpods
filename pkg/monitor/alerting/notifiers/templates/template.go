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

package templates

import "yunion.io/x/onecloud/pkg/apis/monitor"

type TemplateConfig struct {
	monitor.NotificationTemplateConfig
}

func NewTemplateConfig(c monitor.NotificationTemplateConfig) *TemplateConfig {
	return &TemplateConfig{
		NotificationTemplateConfig: c,
	}
}

const MarkdownTemplate = `
## {{.Title}}

- 时间: {{.StartTime}}
- 级别: {{.Level}}

{{range .Matches}}

- 指标: {{.Metric}}
- 当前值: {{.Value}}

### 触发条件:

> {{.Condition}}

### 标签

{{range $key, $value := .Tags}}
> {{ $key }}: {{ $value}}
{{end}}
{{end}}
`

func (c TemplateConfig) GenerateMarkdown() (string, error) {
	return CompileTEmplateFromMap(MarkdownTemplate, c)
}
