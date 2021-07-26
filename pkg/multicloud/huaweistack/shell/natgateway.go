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
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NatGatewayOptions struct {
		NatGatewayID string `help:"Nat Gateway ID"`
		VpcID        string `help:"Vpc ID"`
	}
	shellutils.R(&NatGatewayOptions{}, "nat-list", "List nat gateway", func(region *huawei.SRegion, args *NatGatewayOptions) error {
		natGateways, err := region.GetNatGateways(args.VpcID, args.NatGatewayID)
		if err != nil {
			return err
		}
		printList(natGateways, 0, 0, 0, nil)
		return nil
	})

	type NatGatewayIdOptions struct {
		ID string `help:"Nat Id"`
	}

	shellutils.R(&NatGatewayIdOptions{}, "nat-delete", "Delete nat gateways", func(cli *huawei.SRegion, args *NatGatewayIdOptions) error {
		return cli.DeleteNatGateway(args.ID)
	})

	shellutils.R(&NatGatewayIdOptions{}, "nat-show", "Show nat gateways", func(cli *huawei.SRegion, args *NatGatewayIdOptions) error {
		nat, err := cli.GetNatGateway(args.ID)
		if err != nil {
			return err
		}
		printObject(nat)
		return nil
	})

	type SCreateDNatOptions struct {
		GatewayID    string `help:"Nat Gateway ID" positional:"true"`
		Protocol     string `help:"Protocol(tcp/udp)" positional:"true"`
		ExternalIPID string `help:"External IP ID" positional:"true"`
		ExternalPort int    `help:"External Port" positional:"true"`
		InternalIP   string `help:"Internal IP" positional:"true"`
		InternalPort int    `help:"Nat Gateway ID" positional:"true"`
	}
	shellutils.R(&SCreateDNatOptions{}, "dnat-create", "Create dnat", func(region *huawei.SRegion, args *SCreateDNatOptions) error {
		rule := cloudprovider.SNatDRule{
			Protocol:     args.Protocol,
			ExternalIPID: args.ExternalIPID,
			ExternalPort: args.ExternalPort,
			InternalIP:   args.InternalIP,
			InternalPort: args.InternalPort,
		}
		dnat, err := region.CreateNatDEntry(rule, args.GatewayID)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SCreateSNatOptions struct {
		GatewayID    string `help:"Nat Gateway ID" positional:"true"`
		SourceCIDR   string `help:"Source cidr" positional:"true"`
		ExternalIPID string `help:"External IP ID" positional:"true"`
	}
	shellutils.R(&SCreateSNatOptions{}, "snat-create", "Create snat", func(region *huawei.SRegion, args *SCreateSNatOptions) error {
		rule := cloudprovider.SNatSRule{
			SourceCIDR:   args.SourceCIDR,
			ExternalIPID: args.ExternalIPID,
		}
		snat, err := region.CreateNatSEntry(rule, args.GatewayID)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowSNatOptions struct {
		NatID string `help:"SNat ID" positional:"true"`
	}
	shellutils.R(&SShowSNatOptions{}, "snat-show", "Show snat", func(region *huawei.SRegion, args *SShowSNatOptions) error {
		snat, err := region.GetNatSEntryByID(args.NatID)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowDNatOptions struct {
		NatID string `help:"DNat ID" positional:"true"`
	}
	shellutils.R(&SShowDNatOptions{}, "dnat-show", "Show dnat", func(region *huawei.SRegion, args *SShowDNatOptions) error {
		dnat, err := region.GetNatDEntryByID(args.NatID)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SDeleteSNatOptions struct {
		NatID string `help:"SNat ID" positional:"true"`
	}
	shellutils.R(&SDeleteSNatOptions{}, "snat-delete", "Delete snat", func(region *huawei.SRegion, args *SDeleteSNatOptions) error {
		err := region.DeleteNatSEntry(args.NatID)
		if err != nil {
			return err
		}
		return nil
	})

	type SDeleteDNatOptions struct {
		NatID string `help:"DNat ID" positional:"true"`
	}
	shellutils.R(&SDeleteDNatOptions{}, "dnat-delete", "Delete dnat", func(region *huawei.SRegion, args *SDeleteDNatOptions) error {
		err := region.DeleteNatDEntry(args.NatID)
		if err != nil {
			return err
		}
		return nil
	})
}
