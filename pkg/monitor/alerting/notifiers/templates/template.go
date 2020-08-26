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

const DefaultMarkdownTemplate = `
	## {{.Title}}

	- 时间: {{.StartTime}}
	- 级别: {{.Level}}

	{{range .Matches}}

	- 指标: {{.Metric}}
	- 当前值: {{.Value | FormateFloat}}

	### 触发条件:

	- {{.Condition}}

	### 标签

	{{range $key, $value := .Tags}}
		> {{ $key }}: {{ $value}}
	{{end}}
	{{end}}
`

const EmailMarkdownTemplate = `
<!DOCTYPE html>
<html lang="en">
	<head>
	  <meta charset="UTF-8">
	</head>
	<body>
		<p>## {{.Title}}</p>
		<p>- 时间: {{.StartTime}}</p>
		<p>- 级别: {{.Level}}</p>
		<p>{{range .Matches}}</p>
		<p>- 指标: {{.Metric}}</p>
		<p>- 当前值: {{.Value | FormateFloat}}</p>
		</br><p>## 触发条件:</p>
		<p> {{.Condition}}</p>
		</br><p>## 标签</p>
		<p>
			{{range $key, $value := .Tags}}
				{{ $key }}: {{ $value}} </br>
			{{end}}
			{{end}}
		</p>
	</body>
</html>
`

func (c TemplateConfig) GenerateMarkdown() (string, error) {
	return CompileTEmplateFromMapText(DefaultMarkdownTemplate, c)
}

func (c TemplateConfig) GenerateEmailMarkdown() (string, error) {
	return CompileTemplateFromMapHtml(EmailMarkdownTemplate, c)
}
