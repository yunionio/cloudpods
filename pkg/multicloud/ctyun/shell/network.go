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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VNetworkListOptions struct {
		Vpc string `help:"Vpc ID"`
	}
	shellutils.R(&VNetworkListOptions{}, "subnet-list", "List subnets", func(cli *ctyun.SRegion, args *VNetworkListOptions) error {
		vswitches, e := cli.GetNetwroks(args.Vpc)
		if e != nil {
			return e
		}
		printList(vswitches, 0, 0, 0, nil)
		return nil
	})

	type NetworkCreateOptions struct {
		VpcId      string `help:"vpc id"`
		ZoneId     string `help:"zone id"`
		Name       string `help:"subnet name"`
		Cidr       string `help:"cidr"`
		GatewayIp  string `help:"gateway ip"`
		DhcpEnable string `help:"gateway ip" choice:"true|false"`
	}
	shellutils.R(&NetworkCreateOptions{}, "subnet-create", "Create subnet", func(cli *ctyun.SRegion, args *NetworkCreateOptions) error {
		vpc, e := cli.CreateNetwork(args.VpcId, args.ZoneId, args.Name, args.Cidr, args.GatewayIp, args.DhcpEnable)
		if e != nil {
			return e
		}
		printObject(vpc)
		return nil
	})
}
