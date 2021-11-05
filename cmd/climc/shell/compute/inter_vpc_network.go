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

package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.InterVpcNetworks).WithKeyword("inter-vpc-network")
	cmd.List(&options.InterVpcNetworkListOPtions{})
	cmd.Show(&options.InterVpcNetworkIdOPtions{})
	cmd.Create(&options.InterVpcNetworkCreateOPtions{})
	cmd.Delete(&options.InterVpcNetworkIdOPtions{})
	cmd.Perform("syncstatus", &options.InterVpcNetworkIdOPtions{})
	cmd.Perform("addvpc", &options.InterVpcNetworkAddVpcOPtions{})
	cmd.Perform("removevpc", &options.InterVpcNetworkRemoveVpcOPtions{})
}
