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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/ecloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VNetworkListOptions struct {
		VpcId string `help:"Vpc ID"`
	}
	shellutils.R(&VNetworkListOptions{}, "subnet-list", "List subnets", func(cli *ecloud.SRegion, args *VNetworkListOptions) error {
		ivpc, err := cli.GetIVpcById(args.VpcId)
		if err != nil {
			return err
		}
		vpc := ivpc.(*ecloud.SVpc)
		networks, e := cli.GetNetworks(vpc.RouterId, "")
		if e != nil {
			return e
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})
}
