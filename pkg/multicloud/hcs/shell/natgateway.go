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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NatGatewayOptions struct {
		NatGatewayId string `help:"Nat Gateway Id"`
		VpcId        string `help:"Vpc Id"`
	}
	shellutils.R(&NatGatewayOptions{}, "nat-list", "List nat gateway", func(region *hcs.SRegion, args *NatGatewayOptions) error {
		natGateways, err := region.GetNatGateways(args.VpcId)
		if err != nil {
			return err
		}
		printList(natGateways, 0, 0, 0, nil)
		return nil
	})

	type NatGatewayIdOptions struct {
		Id string `help:"Nat Id"`
	}

	shellutils.R(&NatGatewayIdOptions{}, "nat-delete", "Delete nat gateways", func(cli *hcs.SRegion, args *NatGatewayIdOptions) error {
		return cli.DeleteNatGateway(args.Id)
	})

	shellutils.R(&NatGatewayIdOptions{}, "nat-show", "Show nat gateways", func(cli *hcs.SRegion, args *NatGatewayIdOptions) error {
		nat, err := cli.GetNatGateway(args.Id)
		if err != nil {
			return err
		}
		printObject(nat)
		return nil
	})

	type SCreateDNatOptions struct {
		GatewayId    string `help:"Nat Gateway Id" positional:"true"`
		Protocol     string `help:"Protocol(tcp/udp)" positional:"true"`
		ExternalIPId string `help:"External IP Id" positional:"true"`
		ExternalPort int    `help:"External Port" positional:"true"`
		InternalIP   string `help:"Internal IP" positional:"true"`
		InternalPort int    `help:"Nat Gateway Id" positional:"true"`
	}
	shellutils.R(&SCreateDNatOptions{}, "dnat-create", "Create dnat", func(region *hcs.SRegion, args *SCreateDNatOptions) error {
		rule := cloudprovider.SNatDRule{
			Protocol:     args.Protocol,
			ExternalIPID: args.ExternalIPId,
			ExternalPort: args.ExternalPort,
			InternalIP:   args.InternalIP,
			InternalPort: args.InternalPort,
		}
		dnat, err := region.CreateNatDEntry(rule, args.GatewayId)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SCreateSNatOptions struct {
		GatewayId    string `help:"Nat Gateway Id" positional:"true"`
		SourceCIDR   string `help:"Source cidr" positional:"true"`
		ExternalIPId string `help:"External IP Id" positional:"true"`
	}
	shellutils.R(&SCreateSNatOptions{}, "snat-create", "Create snat", func(region *hcs.SRegion, args *SCreateSNatOptions) error {
		rule := cloudprovider.SNatSRule{
			SourceCIDR:   args.SourceCIDR,
			ExternalIPID: args.ExternalIPId,
		}
		snat, err := region.CreateNatSEntry(rule, args.GatewayId)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowSNatOptions struct {
		NatId string `help:"SNat Id" positional:"true"`
	}
	shellutils.R(&SShowSNatOptions{}, "snat-show", "Show snat", func(region *hcs.SRegion, args *SShowSNatOptions) error {
		snat, err := region.GetNatSEntry(args.NatId)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowDNatOptions struct {
		NatId string `help:"DNat Id" positional:"true"`
	}
	shellutils.R(&SShowDNatOptions{}, "dnat-show", "Show dnat", func(region *hcs.SRegion, args *SShowDNatOptions) error {
		dnat, err := region.GetNatDEntry(args.NatId)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SDeleteSNatOptions struct {
		NatId string `help:"SNat Id" positional:"true"`
	}
	shellutils.R(&SDeleteSNatOptions{}, "snat-delete", "Delete snat", func(region *hcs.SRegion, args *SDeleteSNatOptions) error {
		err := region.DeleteNatSEntry(args.NatId)
		if err != nil {
			return err
		}
		return nil
	})

	type SDeleteDNatOptions struct {
		NatId string `help:"DNat Id" positional:"true"`
	}
	shellutils.R(&SDeleteDNatOptions{}, "dnat-delete", "Delete dnat", func(region *hcs.SRegion, args *SDeleteDNatOptions) error {
		err := region.DeleteNatDEntry(args.NatId)
		if err != nil {
			return err
		}
		return nil
	})
}
