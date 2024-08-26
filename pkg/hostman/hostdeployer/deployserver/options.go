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
	host_options "yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SDeployOptions struct {
	host_options.SHostBaseOptions

	// PrivatePrefixes []string `help:"IPv4 private prefixes"`
	ChntpwPath string `help:"path to chntpw tool" default:"/usr/local/bin/chntpw.static"`

	CloudrootDir     string `help:"User cloudroot home dir" default:"/opt"`
	CommonConfigFile string `help:"common config file for container"`

	DeployTempDir string `help:"temp dir for deployer" default:"/opt/cloud/workspace/run/deploy"`

	AllowVmSELinux bool `help:"turn off vm selinux" default:"false" json:"allow_vm_selinux"`

	HugepagesOption string `help:"Hugepages option: disable|native|transparent" default:"transparent"`
	HugepageSizeMb  int    `help:"hugepage size mb default 1G" default:"1024"`
	// DefaultQemuVersion   string   `help:"Default qemu version" default:"4.2.0"`
	DeployGuestMemSizeMb int      `help:"Deploy guest mem size mb" default:"320"`
	ListenInterface      string   `help:"Master address of host server"`
	Networks             []string `help:"Network interface information"`

	DeployAction     string `help:"local deploy action"`
	DeployParams     string `help:"params for deploy action"`
	DeployParamsFile string `help:"file store params for deploy action"`
}

var DeployOption SDeployOptions

func Parse() SDeployOptions {
	var hostOpts SDeployOptions
	common_options.ParseOptionsIgnoreNoConfigfile(&hostOpts, os.Args, "host.conf", "host")
	if len(hostOpts.CommonConfigFile) > 0 && fileutils2.Exists(hostOpts.CommonConfigFile) {
		commonCfg := &host_options.SHostBaseOptions{}
		commonCfg.Config = hostOpts.CommonConfigFile
		common_options.ParseOptions(commonCfg, []string{os.Args[0]}, "common.conf", "host")
		baseOpt := hostOpts.BaseOptions.BaseOptions
		hostOpts.SHostBaseOptions = *commonCfg
		// keep base options
		hostOpts.BaseOptions.BaseOptions = baseOpt
	}
	return hostOpts
}

func optionsInit() {
	DeployOption = Parse()
}
