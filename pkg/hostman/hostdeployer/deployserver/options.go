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

import (
	"os"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SDeployOptions struct {
	common_options.HostCommonOptions

	PrivatePrefixes      []string `help:"IPv4 private prefixes"`
	ChntpwPath           string   `help:"path to chntpw tool" default:"/usr/local/bin/chntpw.static"`
	EnableRemoteExecutor bool     `help:"Enable remote executor" default:"false"`
	CloudrootDir         string   `help:"User cloudroot home dir" default:"/opt"`
	ImageDeployDriver    string   `help:"Image deploy driver" default:"nbd" choices:"nbd|libguestfs"`
	CommonConfigFile     string   `help:"common config file for container"`
}

var DeployOption SDeployOptions

func Parse() (hostOpts SDeployOptions) {
	common_options.ParseOptions(&hostOpts, os.Args, "host.conf", "host")
	if len(hostOpts.CommonConfigFile) > 0 {
		commonCfg := &common_options.HostCommonOptions{}
		commonCfg.Config = hostOpts.CommonConfigFile
		common_options.ParseOptions(commonCfg, []string{os.Args[0]}, "common.conf", "host")
		baseOpt := hostOpts.BaseOptions.BaseOptions
		hostOpts.HostCommonOptions = *commonCfg
		// keep base options
		hostOpts.BaseOptions.BaseOptions = baseOpt
	}
	return hostOpts
}

func init() {
	DeployOption = Parse()
}
