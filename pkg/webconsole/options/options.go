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

	ApiServer       string `help:"API server url to handle websocket connection, usually with public access" default:"http://webconsole.yunion.io"`
	KubectlPath     string `help:"kubectl binary path used to connect k8s cluster" default:"/usr/bin/kubectl"`
	IpmitoolPath    string `help:"ipmitool binary path used to connect baremetal sol" default:"/usr/bin/ipmitool"`
	SshToolPath     string `help:"sshtool binary path used to connect server sol" default:"/usr/bin/ssh"`
	SshpassToolPath string `help:"sshpass tool binary path used to connect server sol" default:"/usr/bin/sshpass"`
	EnableAutoLogin bool   `help:"allow webconsole to log in directly with the cloudroot public key" default:"false"`
}
