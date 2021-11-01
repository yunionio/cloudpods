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

package cloudproxy

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	options "yunion.io/x/onecloud/pkg/mcclient/options/cloudproxy"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ProxyAgents)
	cmd = cmd.WithKeyword("proxy-agent")
	cmd.Create(&options.ProxyAgentCreateOptions{})
	cmd.Update(&options.ProxyAgentUpdateOptions{})
	cmd.List(&options.ProxyAgentListOptions{})
	cmd.Show(&options.ProxyAgentShowOptions{})
	cmd.Delete(&options.ProxyAgentDeleteOptions{})
	cmd.Perform("purge-forwards", &options.ProxyAgentPurgeForwardsOptions{})
}
