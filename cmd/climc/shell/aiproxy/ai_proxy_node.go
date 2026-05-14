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
	"yunion.io/x/onecloud/cmd/climc/shell"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
	apoptions "yunion.io/x/onecloud/pkg/mcclient/options/aiproxy"
)

func init() {
	cmd := shell.NewResourceCmd(&apmodules.AiProxyNodes)
	cmd.Create(new(apoptions.AiProxyNodeCreateOptions))
	cmd.List(new(apoptions.AiProxyNodeListOptions))
	cmd.Show(new(apoptions.AiProxyNodeShowOptions))
	cmd.Update(new(apoptions.AiProxyNodeUpdateOptions))
	cmd.Delete(new(apoptions.AiProxyNodeDeleteOptions))
	registerEnableDisable(cmd)
	cmd.PerformClass("register", new(apoptions.AiProxyNodeRegisterOptions))
}
