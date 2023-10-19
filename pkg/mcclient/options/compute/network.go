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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type NetworkListOptions struct {
	options.BaseListOptions

	Ip         string   `help:"search networks that contain this IP"`
	ZoneIds    []string `help:"search networks in zones"`
	Wire       string   `help:"search networks belongs to a wire" json:"-"`
	Host       string   `help:"search networks attached to a host"`
	Vpc        string   `help:"search networks belongs to a VPC"`
	Region     string   `help:"search networks belongs to a CloudRegion" json:"cloudregion"`
	City       string   `help:"search networks belongs to a city"`
	Usable     *bool    `help:"search usable networks"`
	ServerType string   `help:"search networks belongs to a ServerType" choices:"baremetal|container|eip|guest|ipmi|pxe"`
	Schedtag   string   `help:"filter networks by schedtag"`

	HostSchedtagId string `help:"filter by host schedtag"`

	IsAutoAlloc *bool `help:"search network with is_auto_alloc"`
	IsClassic   *bool `help:"search classic on-premise network"`

	Status string `help:"filter by network status"`

	GuestIpStart []string `help:"search by guest_ip_start"`
	GuestIpEnd   []string `help:"search by guest_ip_end"`
	IpMatch      []string `help:"search by network ips"`

	BgpType      []string `help:"filter by bgp_type"`
	HostType     string   `help:"filter by host_type"`
	RouteTableId string   `help:"Filter by RouteTable"`

	OrderByIpStart string
	OrderByIpEnd   string
}

func (opts *NetworkListOptions) GetContextId() string {
	return opts.Wire
}

func (opts *NetworkListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type NetworkUpdateOptions struct {
	options.BaseUpdateOptions

	StartIp     string `help:"Start ip"`
	EndIp       string `help:"end ip"`
	NetMask     int64  `help:"Netmask"`
	Gateway     string `help:"IP of gateway"`
	Dns         string `help:"IP of DNS server"`
	Domain      string `help:"Domain"`
	Dhcp        string `help:"DHCP server IP"`
	Ntp         string `help:"Ntp server domain names"`
	VlanId      int64  `help:"Vlan ID"`
	ExternalId  string `help:"External ID"`
	AllocPolicy string `help:"Address allocation policy" choices:"none|stepdown|stepup|random"`
	IsAutoAlloc *bool  `help:"Add network into auto-allocation pool" negative:"no_auto_alloc"`
}

func (opts *NetworkUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if len(opts.StartIp) > 0 {
		params.Add(jsonutils.NewString(opts.StartIp), "guest_ip_start")
	}
	if len(opts.EndIp) > 0 {
		params.Add(jsonutils.NewString(opts.EndIp), "guest_ip_end")
	}
	if opts.NetMask > 0 {
		params.Add(jsonutils.NewInt(opts.NetMask), "guest_ip_mask")
	}
	if len(opts.Gateway) > 0 {
		params.Add(jsonutils.NewString(opts.Gateway), "guest_gateway")
	}
	if len(opts.Dns) > 0 {
		if opts.Dns == "none" {
			params.Add(jsonutils.NewString(""), "guest_dns")
		} else {
			params.Add(jsonutils.NewString(opts.Dns), "guest_dns")
		}
	}
	if len(opts.Domain) > 0 {
		if opts.Domain == "none" {
			params.Add(jsonutils.NewString(""), "guest_domain")
		} else {
			params.Add(jsonutils.NewString(opts.Domain), "guest_domain")
		}
	}
	if len(opts.Dhcp) > 0 {
		if opts.Dhcp == "none" {
			params.Add(jsonutils.NewString(""), "guest_dhcp")
		} else {
			params.Add(jsonutils.NewString(opts.Dhcp), "guest_dhcp")
		}
	}
	if len(opts.Ntp) > 0 {
		if opts.Ntp == "none" {
			params.Add(jsonutils.NewString(""), "guest_ntp")
		} else {
			params.Add(jsonutils.NewString(opts.Ntp), "guest_ntp")
		}
	}
	if opts.VlanId > 0 {
		params.Add(jsonutils.NewInt(opts.VlanId), "vlan_id")
	}
	if len(opts.ExternalId) > 0 {
		params.Add(jsonutils.NewString(opts.ExternalId), "external_id")
	}
	if len(opts.AllocPolicy) > 0 {
		params.Add(jsonutils.NewString(opts.AllocPolicy), "alloc_policy")
	}
	if opts.IsAutoAlloc != nil {
		params.Add(jsonutils.NewBool(*opts.IsAutoAlloc), "is_auto_alloc")
	}
	if params.Size() == 0 {
		return nil, shell.InvalidUpdateError()
	}
	return params, nil
}

type NetworkIdOptions struct {
	ID string `help:"ID or Name of the network to show"`
}

func (opts *NetworkIdOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type NetworkIpMacIdOptions struct {
	ID string `help:"ID or Name of the network_ip_mac to show"`
}

func (opts *NetworkIpMacIdOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkIpMacIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type NetworkIpMacListOptions struct {
	options.BaseListOptions

	Network string   `help:"search networks" json:"network_id"`
	MacAddr []string `help:"search by mac addr"`
	IpAddr  []string `help:"search by ip addr"`
}

func (opts *NetworkIpMacListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type NetworkIpMacUpdateOptions struct {
	ID string `help:"ID or Name of resource to update"`

	MacAddr string `help:"update mac addr"`
	IpAddr  string `help:"update ip addr"`
}

func (opts *NetworkIpMacUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkIpMacUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type NetworkIpMacCreateOptions struct {
	NETWORK string `help:"network id" json:"network_id"`
	MACADDR string `help:"mac address" json:"mac_addr"`
	IPADDR  string `help:"ip address" json:"ip_addr"`
}

func (opts *NetworkIpMacCreateOptions) Params() (jsonutils.JSONObject, error) {
	if opts.NETWORK == "" {
		return nil, errors.Errorf("missing network params")
	}
	if opts.MACADDR == "" {
		return nil, errors.Errorf("missing mac_addr params")
	}
	if opts.IPADDR == "" {
		return nil, errors.Errorf("missing ip_addr params")
	}
	return options.ListStructToParams(opts)
}

type NetworkSwitchWireOptions struct {
	ID string `help:"ID or Name of resource to update"`

	api.NetworkSwitchWireInput
}

func (opts *NetworkSwitchWireOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkSwitchWireOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type NetworkSyncAdditionalWiresOptions struct {
	ID string `help:"ID or Name of resource to update"`

	api.NetworSyncAdditionalWiresInput
}

func (opts *NetworkSyncAdditionalWiresOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkSyncAdditionalWiresOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
