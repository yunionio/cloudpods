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

type EsxiOptions struct {
	common_options.CommonOptions

	ListenInterface string `help;"Master address of host server" default:"br0"`
	ListenAddress string `help:"Host serve IP address to select when multiple address bind to
ListenInterface"`
	EsxiAgentPath           string `default:"/opt/cloud/workspace/esxi_agent" help:"Path for esxi agent configuration files"`
	ImageCachePath          string `help:"Path for storing image caches"`
	ImageCacheLimit         int    `help:"Maximal storage space for image caching, in GB" default:"20"`
	AgentTempPath           string `help:"Path for ESXI Agent"`
	AgentTempLimit          int    `help:"Maximal storage space for ESXi agent, in GB" default:"20"`
	LinuxDefaultRootUser    bool   `help:"Default account for Linux system is root" default:"false"`
	WindowsDefaultAdminUser bool   `help:"Default account for Windows system is Administrator" default:"true"`
	DefaultImageSaveFormat  string `help:"Default image save format, default is vmdk, canbe qcow2" default:"vmdk"`
	Zone                    string `help:"Zone where the agent locates"`
}

var (
	Options EsxiOptions
)
