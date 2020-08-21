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

	{{ range .Matches}}
	- 指标: {{.Metric}}
	- 当前值: {{.ValueStr}}

	### 触发条件:
	- {{ $.Description}}

	### 标签
	 > 名称: {{ GetValFromMap .Tags "name" }}
	 > ip: {{ GetValFromMap .Tags "ip" }}
	 > 平台: {{ GetValFromMap .Tags "brand" }}
    ------
	{{- end}}
`

func (c TemplateConfig) GenerateMarkdown() (string, error) {
	return CompileTEmplateFromMapText(DefaultMarkdownTemplate, c)
}

func (c TemplateConfig) GenerateEmailMarkdown() (string, error) {
	return CompileTemplateFromMapHtml(EmailMarkdownTemplate, c)
}

const EmailMarkdownTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="ie=edge">
  <title>{{.Title}}</title>
</head>
<style>
  .title {
    height: 40px;
    line-height: 40px;
    width: 960px;
    background-color: #4da1ff;
    color: #fff;
    font-size: 16px;
    text-align: center;
    margin: 0 auto;
  }
  .table {
    width: 960px;
    margin: 0 auto;
    padding: 10px 30px 0px 30px;
    font-family:'微软雅黑',Helvetica,Arial,sans-serif;
    font-size:14px;
    background-color: #fbfbfb;
  }
  .tr-title {
    height: 10px;
    border-left: 5px solid #4da1ff;
    margin-left: 10px;
    padding: 3px 8px;
    font-weight: bold;
  }
  .td {
    width: 80px;
    padding-left: 20px;
    height: 35px;
    font-weight: 400;
  }
  .link {
    text-decoration: none;
    color: #3591FF;
  }
  .thead-tr td {
    border-left: 1px solid #d7d7d7;
    border-top: 1px solid #d7d7d7;
    height: 32px;
    font-size: 14px;
    background-color: #d7d7d7;
    text-align: center;
  }
  .tbody-tr td {
    border-left: 1px solid #d7d7d7;
    border-top: 1px solid #d7d7d7;
    height: 32px;
    font-size: 14px;
    font-weight: 400;
    text-align: center;
  }
  .pb-3 {
    padding-bottom: 30px;
  }
  .resouce-table {
    width: 98%;
    color: #717171;
    border-right: 1px solid #d7d7d7;
    border-bottom: 1px solid #d7d7d7;
  }
</style>
<body>
  <h3 class="title">报警提醒</h3>
  <table border="0" cellspacing="0" cellpadding="0" class="table">
    <tr><td colspan="4" class="tr-title">报警信息</td></tr>
    <tr><td style="height: 10px;"></td></tr>
    <tr>
      <td class="td">报警策略：</td>
      <td>{{.Name}}</td>
    </tr>
    <tr>
      <td class="td">报警级别：</td>
      <td>{{.Level}}</td>
    </tr>
    <tr>
      <td class="td">报警时间：</td>
      <td>{{.StartTime}}</td>
    </tr>
    <tr>
      <td class="td">策略详情：</td>
      <td>{{.Description}}</td>
    </tr>
  </table>
  <table class="table" style="padding-top: 6px; padding-bottom: 10px;">
    <tr>
      <td style="padding-left: 20px; font-size: 14px;">若要查看详情信息，<a class="link" target="_blank" href="{{.WebUrl}}/commonalerts
">请登录平台进行查看</a></td>
    </tr>
  </table>
  <table border="0" cellspacing="0" cellpadding="0" class="table pb-3">
    <tr><td colspan="4" class="tr-title">报警资源</td></tr>
    <tr><td style="height: 10px;"></td></tr>
    <tr>
      <td colspan="4" style="padding: 10px 0 0 20px;">
        <table cellspacing="0" cellpadding="0" class="resouce-table">
          <thead>
            <tr class="thead-tr">
              <td>序号</td>
	              <td>名称</td>
              <td>IP</td>
              <td>平台</td>
              <td>当前值</td>
            </tr>
          </thead>
          <tbody>
			{{- range $i, $Matche := .Matches}}
				<tr class="tbody-tr">
				  <td>{{ Inc $i}}</td>
				  <td>
					{{- GetValFromMap .Tags "name"}}
                  </td>
				  <td>
					{{ GetValFromMap .Tags "ip" }}
                  </td>
				  <td>
				   	{{ GetValFromMap .Tags "brand" }}
				  </td>
				  <td>{{$Matche.ValueStr}}</td>
				</tr>
			{{end}}
          </tbody>
        </table>
      </td>
    </tr>
    <tr><td style="height: 10px;"></td></tr>
  </table>
  <p style="width: 960px; height: 40px; line-height: 40px; margin: 0 auto; background-color: #4da1ff; color: #fff; font-size: 12px; text-align: center; ">本邮件由系统自动发送，请勿直接回复！</p>
</body>
</html>
`
