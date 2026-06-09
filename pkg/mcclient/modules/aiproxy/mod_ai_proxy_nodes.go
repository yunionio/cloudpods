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

package aiproxy

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type AiProxyNodeManager struct {
	modulebase.ResourceManager
}

var AiProxyNodes AiProxyNodeManager

func init() {
	AiProxyNodes = AiProxyNodeManager{
		modules.NewAIProxyManager("ai_proxy_node", "ai_proxy_nodes",
			[]string{"Id", "Name", "Address", "Domain", "Last_seen", "Hb_timeout", "Is_active", "Enabled", "Status"},
			[]string{}),
	}
	modules.Register(&AiProxyNodes)
}
