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

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

var (
	Options WebConsoleOptions
)

type WebConsoleOptions struct {
	common_options.CommonOptions

	common_options.DBOptions

	KubectlPath       string `help:"kubectl binary path used to connect k8s cluster" default:"/usr/bin/kubectl"`
	AdbPath           string `help:"adb binary path" default:"/usr/bin/adb"`
	IpmitoolPath      string `help:"ipmitool binary path used to connect baremetal sol" default:"/usr/bin/ipmitool"`
	EnableAutoLogin   bool   `help:"allow webconsole to log in directly with the cloudroot public key" default:"false"`
	ApsaraConsoleAddr string `help:"Apsara console addr" default:"https://xxxx.com.cn/module/ecs/vnc/index.html"`
	AliyunConsoleAddr string `help:"Aliyun vnc addr" default:"https://ecs.console.aliyun.com/vnc/index.htm"`

	SshSessionTimeoutMinutes int `help:"ssh timeout session" default:"-1"`
	RdpSessionTimeoutMinutes int `help:"rdp timeout session" default:"-1"`

	EnableWatermark        bool `help:"enable water mark" default:"false"`
	EnableCommandRecording bool `help:"enable command recording" default:"false"`

	KeepWebsocketSession bool `help:"keep websocket session" default:"false"`
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*WebConsoleOptions)
	newOpts := newO.(*WebConsoleOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}

	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}

	return changed
}
