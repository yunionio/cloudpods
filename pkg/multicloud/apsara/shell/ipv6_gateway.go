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
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type IPv6GatewayListOptions struct {
		VpcId      string
		PageSize   int `default:"10"`
		PageNumber int `default:"1"`
	}
	shellutils.R(&IPv6GatewayListOptions{}, "ipv6-gateway-list", "List IPv6 gateways", func(cli *apsara.SRegion, args *IPv6GatewayListOptions) error {
		gateways, _, err := cli.GetIPv6Gateways(args.VpcId, args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(gateways, 0, 0, 0, []string{})
		return nil
	})

	type IPv6GatewayIdOptions struct {
		ID string
	}

	shellutils.R(&IPv6GatewayIdOptions{}, "ipv6-gateway-show", "Show IPv6 gateway", func(cli *apsara.SRegion, args *IPv6GatewayIdOptions) error {
		gateway, err := cli.GetIPv6Gateway(args.ID)
		if err != nil {
			return err
		}
		printObject(gateway)
		return nil
	})

}
