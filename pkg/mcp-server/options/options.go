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

import (
	"strings"
	"time"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

const DefaultPlatformName = "Cloudpods"

type MCPServerOptions struct {
	common_options.CommonOptions
	// 服务基础信息
	MCPServerName        string `help:"MCP service name"`
	MCPServerVersion     string `help:"MCP service version"`
	MCPServerDescription string `help:"MCP service description"`

	// 连接超时配置
	Timeout int `help:"SDK connection timeout to platform API (seconds)" default:"30"`
	// ServerCreateWaitSeconds 创建后等待 running/ready 的最长时间；超时仍返回 server_id，便于 agent 用 server-show 继续轮询（宜小于 LLM MCPAgentTimeout）
	ServerCreateWaitSeconds int `help:"max seconds to wait for server running/ready after create; on timeout still return server_id" default:"90"`
}

var (
	Options MCPServerOptions
)

// ResolvedPlatformName 返回配置中的平台展示名，空则回退 DefaultPlatformName。
func ResolvedPlatformName() string {
	name := strings.TrimSpace(Options.PlatformName)
	if name == "" {
		return DefaultPlatformName
	}
	return name
}

// ServerCreateWaitDuration 创建等待时长。
func ServerCreateWaitDuration() time.Duration {
	sec := Options.ServerCreateWaitSeconds
	if sec <= 0 {
		sec = 90
	}
	return time.Duration(sec) * time.Second
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*MCPServerOptions)
	newOpts := newO.(*MCPServerOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	// PlatformName 写入 ServerInstructions 需重启进程
	if oldOpts.PlatformName != newOpts.PlatformName {
		changed = true
	}
	// ServerCreateWaitSeconds / Timeout 热更新即可（OptionManager 会拷贝到 Options）
	return changed
}
