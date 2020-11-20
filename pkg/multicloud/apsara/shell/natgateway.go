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
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NatGatewayListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&NatGatewayListOptions{}, "natgateway-list", "List NAT gateways", func(cli *apsara.SRegion, args *NatGatewayListOptions) error {
		gws, total, e := cli.GetNatGateways("", "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(gws, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type NatSEntryListOptions struct {
		ID     string `help:"SNat Table ID"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&NatSEntryListOptions{}, "snat-entry-list", "List SNAT entries", func(cli *apsara.SRegion, args *NatSEntryListOptions) error {
		entries, total, e := cli.GetSNATEntries(args.ID, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(entries, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type NatDEntryListOptions struct {
		ID     string `help:"DNat Table ID"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&NatDEntryListOptions{}, "dnat-entry-list", "List DNAT entries", func(cli *apsara.SRegion, args *NatDEntryListOptions) error {
		entries, total, e := cli.GetForwardTableEntries(args.ID, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(entries, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type SCreateDNatOptions struct {
		GatewayID    string `help:"Nat Gateway ID" positional:"true"`
		Protocol     string `help:"Protocol(tcp/udp)" positional:"true"`
		ExternalIP   string `help:"External IP" positional:"true"`
		ExternalPort int    `help:"External Port" positional:"true"`
		InternalIP   string `help:"Internal IP" positional:"true"`
		InternalPort int    `help:"Nat Gateway ID" positional:"true"`
	}
	shellutils.R(&SCreateDNatOptions{}, "dnat-entry-create", "Create DNAT entry", func(region *apsara.SRegion, args *SCreateDNatOptions) error {
		rule := cloudprovider.SNatDRule{
			Protocol:     args.Protocol,
			ExternalIP:   args.ExternalIP,
			ExternalPort: args.ExternalPort,
			InternalIP:   args.InternalIP,
			InternalPort: args.InternalPort,
		}
		dnat, err := region.CreateForwardTableEntry(rule, args.GatewayID)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SCreateSNatOptions struct {
		GatewayID  string `help:"Nat Gateway ID" positional:"true"`
		SourceCIDR string `help:"Source cidr" positional:"true"`
		ExternalIP string `help:"External IP" positional:"true"`
	}
	shellutils.R(&SCreateSNatOptions{}, "snat-entry-create", "Create SNAT entry", func(region *apsara.SRegion, args *SCreateSNatOptions) error {
		rule := cloudprovider.SNatSRule{
			SourceCIDR: args.SourceCIDR,
			ExternalIP: args.ExternalIP,
		}
		snat, err := region.CreateSNATTableEntry(rule, args.GatewayID)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowSNatOptions struct {
		TableID string `help:"SNat Table ID" positional:"true"`
		NatID   string `help:"SNat Entry ID" positional:"true"`
	}
	shellutils.R(&SShowSNatOptions{}, "snat-entry-show", "show SNAT entry", func(region *apsara.SRegion, args *SShowSNatOptions) error {
		snat, err := region.GetSNATEntry(args.TableID, args.NatID)
		if err != nil {
			return err
		}
		printObject(snat)
		return nil
	})

	type SShowDNatOptions struct {
		TableID string `help:"DNat Table ID" positional:"true"`
		NatID   string `help:"DNat Entry ID" positional:"true"`
	}
	shellutils.R(&SShowDNatOptions{}, "dnat-entry-show", "show SNAT entry", func(region *apsara.SRegion, args *SShowDNatOptions) error {
		dnat, err := region.GetForwardTableEntry(args.TableID, args.NatID)
		if err != nil {
			return err
		}
		printObject(dnat)
		return nil
	})

	type SDeleteSNatOptions struct {
		TableID string `help:"DNat Table ID" positional:"true"`
		NatID   string `help:"SNat Entry ID" positional:"true"`
	}
	shellutils.R(&SDeleteSNatOptions{}, "snat-entry-delete", "Delete SNAT entry", func(region *apsara.SRegion, args *SDeleteSNatOptions) error {
		err := region.DeleteSnatEntry(args.TableID, args.NatID)
		if err != nil {
			return err
		}
		return nil
	})

	type SDeleteDNatOptions struct {
		TableID string `help:"DNat Table ID" positional:"true"`
		NatID   string `help:"DNat Entry ID" positional:"true"`
	}
	shellutils.R(&SDeleteDNatOptions{}, "dnat-entry-delete", "Delete DNAT entry", func(region *apsara.SRegion, args *SDeleteDNatOptions) error {
		err := region.DeleteForwardTableEntry(args.TableID, args.NatID)
		if err != nil {
			return err
		}
		return nil
	})
}
