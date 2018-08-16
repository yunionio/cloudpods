package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type NetworkListOptions struct {
		BaseListOptions
		Ip   string `help:"search networks that contain this IP"`
		Zone string `help:"search networks in a zone"`
		Wire string `help:"search networks belongs to a wire"`
		Vpc  string `help:"search networks belongs to a VPC"`
	}
	R(&NetworkListOptions{}, "network-list", "List networks", func(s *mcclient.ClientSession, args *NetworkListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Ip) > 0 {
			params.Add(jsonutils.NewString(args.Ip), "ip")
		}
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewString(args.Zone), "zone")
		}
		if len(args.Vpc) > 0 {
			params.Add(jsonutils.NewString(args.Vpc), "vpc")
		}
		var result *modules.ListResult
		var err error
		if len(args.Wire) > 0 {
			result, err = modules.Networks.ListInContext(s, params, &modules.Wires, args.Wire)
		} else {
			result, err = modules.Networks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Networks.GetColumns(s))
		return nil
	})

	type NetworkUpdateOptions struct {
		ID          string `help:"ID or Name of zone to update"`
		Name        string `help:"Name of zone"`
		Desc        string `metavar:"<DESCRIPTION>" help:"Description"`
		ServerType  string `help:"server type," choices:"baremetal|guest|container"`
		StartIp     string `help:"Start ip"`
		EndIp       string `help:"end ip"`
		NetMask     int64  `help:"Netmask"`
		Gateway     string `help:"IP of gateway"`
		Dns         string `help:"IP of DNS server"`
		Domain      string `help:"Domain"`
		Dhcp        string `help:"DHCP server IP"`
		VlanId      int64  `help:"Vlan ID" default:"1"`
		ExternalId  string `help:"External ID"`
		AllocPolicy string `help:"Address allocation policy" choices:"none|stepdown|stepup|random"`
	}
	R(&NetworkUpdateOptions{}, "network-update", "Update network", func(s *mcclient.ClientSession, args *NetworkUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.ServerType) > 0 {
			params.Add(jsonutils.NewString(args.ServerType), "server_type")
		}
		if len(args.StartIp) > 0 {
			params.Add(jsonutils.NewString(args.StartIp), "guest_ip_start")
		}
		if len(args.EndIp) > 0 {
			params.Add(jsonutils.NewString(args.EndIp), "guest_ip_end")
		}
		if args.NetMask > 0 {
			params.Add(jsonutils.NewInt(args.NetMask), "guest_ip_mask")
		}
		if len(args.Gateway) > 0 {
			params.Add(jsonutils.NewString(args.Gateway), "guest_gateway")
		}
		if len(args.Dns) > 0 {
			if args.Dns == "none" {
				params.Add(jsonutils.NewString(""), "guest_dns")
			} else {
				params.Add(jsonutils.NewString(args.Dns), "guest_dns")
			}
		}
		if len(args.Domain) > 0 {
			if args.Domain == "none" {
				params.Add(jsonutils.NewString(""), "guest_domain")
			} else {
				params.Add(jsonutils.NewString(args.Domain), "guest_domain")
			}
		}
		if len(args.Dhcp) > 0 {
			if args.Dhcp == "none" {
				params.Add(jsonutils.NewString(""), "guest_dhcp")
			} else {
				params.Add(jsonutils.NewString(args.Dhcp), "guest_dhcp")
			}
		}
		if args.VlanId > 0 {
			params.Add(jsonutils.NewInt(args.VlanId), "vlan_id")
		}
		if len(args.ExternalId) > 0 {
			params.Add(jsonutils.NewString(args.ExternalId), "external_id")
		}
		if len(args.AllocPolicy) > 0 {
			params.Add(jsonutils.NewString(args.AllocPolicy), "alloc_policy")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Networks.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkShowOptions struct {
		ID string `help:"ID or Name of the zone to show"`
	}
	R(&NetworkShowOptions{}, "network-show", "Show network details", func(s *mcclient.ClientSession, args *NetworkShowOptions) error {
		result, err := modules.Networks.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkShowOptions{}, "network-private", "Make a network private", func(s *mcclient.ClientSession, args *NetworkShowOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkShowOptions{}, "network-public", "Make a network public", func(s *mcclient.ClientSession, args *NetworkShowOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "public", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkShowOptions{}, "network-delete", "Delete a network", func(s *mcclient.ClientSession, args *NetworkShowOptions) error {
		result, err := modules.Networks.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkShowOptions{}, "network-purge", "Purge a managed network, not delete the remote entity", func(s *mcclient.ClientSession, args *NetworkShowOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkCreateOptions struct {
		WIRE        string `help:"ID or Name of wire in wihich the network is created"`
		NETWORK     string `help:"Name of new network"`
		STARTIP     string `help:"Start of IPv4 address range"`
		ENDIP       string `help:"End of IPv4 address rnage"`
		NETMASK     int64  `help:"Length of network mask"`
		Gateway     string `help:"Default gateway"`
		VlanId      int64  `help:"Vlan ID" default:"1"`
		AllocPolicy string `help:"Address allocation policy" choices:"none|stepdown|stepup|random"`
		ServerType  string `help:"Server type" choices:"baremetal|guest|container"`
		Desc        string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions{}, "network-create", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NETWORK), "name")
		params.Add(jsonutils.NewString(args.STARTIP), "guest_ip_start")
		params.Add(jsonutils.NewString(args.ENDIP), "guest_ip_end")
		params.Add(jsonutils.NewInt(args.NETMASK), "guest_ip_mask")
		if len(args.Gateway) > 0 {
			params.Add(jsonutils.NewString(args.Gateway), "guest_gateway")
		}
		if args.VlanId > 0 {
			params.Add(jsonutils.NewInt(args.VlanId), "vlan_id")
		}
		if len(args.ServerType) > 0 {
			params.Add(jsonutils.NewString(args.ServerType), "server_type")
		}
		if len(args.AllocPolicy) > 0 {
			params.Add(jsonutils.NewString(args.AllocPolicy), "alloc_policy")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		net, e := modules.Networks.CreateInContext(s, params, &modules.Wires, args.WIRE)
		if e != nil {
			return e
		}
		printObject(net)
		return nil
	})

	type NetworkCreateOptions2 struct {
		Wire    string `help:"ID or Name of wire in which the network is created"`
		Vpc     string `help:"ID or Name of vpc in which the network is created"`
		Zone    string `help:"ID or Name of zone in which the network is created"`
		NETWORK string `help:"Name of new network"`
		PREFIX  string `help:"Start of IPv4 address range"`
		Desc    string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions2{}, "network-create2", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions2) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NETWORK), "name")
		params.Add(jsonutils.NewString(args.PREFIX), "guest_ip_prefix")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Wire) > 0 {
			params.Add(jsonutils.NewString(args.Wire), "wire")
		} else if len(args.Vpc) > 0 {
			if len(args.Zone) > 0 {
				params.Add(jsonutils.NewString(args.Zone), "zone")
				params.Add(jsonutils.NewString(args.Vpc), "vpc")
			} else {
				return fmt.Errorf("Either wire or VPC/Zone must be provided")
			}
		} else {
			return fmt.Errorf("Either wire or VPC/Zone must be provided")
		}
		net, e := modules.Networks.Create(s, params)
		if e != nil {
			return e
		}
		printObject(net)
		return nil
	})

	type NetworkSplitOptions struct {
		NETWORK string `help:"ID or name of network to split"`
		IP      string `help:"Start ip of the split network"`
		Name    string `help:"Name of the new network"`
	}
	R(&NetworkSplitOptions{}, "network-split", "Split a network at the specified IP address", func(s *mcclient.ClientSession, args *NetworkSplitOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IP), "split_ip")
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		net, err := modules.Networks.PerformAction(s, args.NETWORK, "split", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type NetworkMergeOptions struct {
		FROM   string `help:"ID or name of merge network from"`
		TARGET string `help:"ID or name of merge network target"`
	}
	R(&NetworkMergeOptions{}, "network-merge", "Merge two network to be one", func(s *mcclient.ClientSession, args *NetworkMergeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TARGET), "target")
		net, err := modules.Networks.PerformAction(s, args.FROM, "merge", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

}
