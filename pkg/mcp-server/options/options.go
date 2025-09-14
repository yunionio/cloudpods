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

package options

type MCPServerOptions struct {
	// 服务基础信息
	Host                 string `help:"Service listening address (default: localhost)" default:"localhost"`
	Port                 int    `help:"Service listening port" default:"8080"`
	MCPServerName        string `help:"MCP service name"`
	MCPServerVersion     string `help:"MCP service version"`
	MCPServerDescription string `help:"MCP service description"`

	// 认证服务集成
	IdentityBaseURL string `help:"Authentication service entry URL"`

	// 连接超时配置
	Timeout int `help:"SDK connection timeout to cloudpods service (seconds)" default:"30"`

	// 日志配置
	LogLevel  string `help:"Log level (e.g., debug/info/warn/error)" default:"info"`
	LogFormat string `elp:"Log format (e.g., text/json)" default:"text"`
}

var (
	Options MCPServerOptions
)
