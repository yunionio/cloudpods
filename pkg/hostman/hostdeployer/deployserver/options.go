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

package deployserver

import "yunion.io/x/onecloud/pkg/cloudcommon/options"

type SDeployOptions struct {
	options.BaseOptions

	DeployServerSocketPath string   `help:"Deploy server listen socket path" default:"/var/run/deploy.sock"`
	PrivatePrefixes        []string `help:"IPv4 private prefixes"`
	ChntpwPath             string   `help:"path to chntpw tool" default:"/usr/local/bin/chntpw.static"`
	EnableRemoteExecutor   bool     `help:"Enable remote executor" default:"false"`
	ExecSocketPath         string   `help:"Exec socket paht" default:"/var/run/exec.sock"`
}

var DeployOption SDeployOptions
