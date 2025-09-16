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

package ovn

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/ovsdb/cli_util"
	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
	"yunion.io/x/onecloud/pkg/vpcagent/ovn/mac"
	"yunion.io/x/onecloud/pkg/vpcagent/ovnutil"
)

const (
	externalKeyOcVersion = "oc-version"
	externalKeyOcRef     = "oc-ref"
)

type OVNNorthboundKeeper struct {
	DB  ovn_nb.OVNNorthbound
	cli *ovnutil.OvnNbCtl
}

func DumpOVNNorthbound(ctx context.Context, cli *ovnutil.OvnNbCtl) (*OVNNorthboundKeeper, error) {
	db := ovn_nb.OVNNorthbound{}
	itbls := []types.ITable{
		&db.LogicalSwitch,
		&db.LogicalSwitchPort,
		&db.LogicalRouter,
		&db.LogicalRouterPort,
		&db.LogicalRouterStaticRoute,
		&db.ACL,
		&db.DHCPOptions,
		&db.QoS,
		&db.DNS,
	}
	args := []string{"--format=json", "list", "<tbl>"}
	for _, itbl := range itbls {
		tbl := itbl.OvsdbTableName()
		args[2] = tbl
		res := cli.Must(ctx, "List "+tbl, args)
		if err := cli_util.UnmarshalJSON([]byte(res.Output), itbl); err != nil {
			return nil, errors.Wrapf(err, "Unmarshal %s:\n%s",
				itbl.OvsdbTableName(), res.Output)
		}
	}
	keeper := &OVNNorthboundKeeper{
		DB:  db,
		cli: cli,
	}
	return keeper, nil
}

func ptr(s string) *string {
	return &s
}

func ovnCreateArgs(irow types.IRow, idRef string) []string {
	args := append([]string{
		"--", "--id=@" + idRef, "create", irow.OvsdbTableName(),
	}, irow.OvsdbCmdArgs()...)
	return args
}

func (keeper *OVNNorthboundKeeper) ClaimVpc(ctx context.Context, vpc *agentmodels.Vpc) error {
	var (
		args      []string
		ocVersion = fmt.Sprintf("%s.%d", vpc.UpdatedAt, vpc.UpdateVersion)
	)

	vpcLr := &ovn_nb.LogicalRouter{
		Name: vpcLrName(vpc.Id),
	}
	irows := []types.IRow{vpcLr}

	var (
		hasDistgw = vpcHasDistgw(vpc)
		hasEipgw  = vpcHasEipgw(vpc)
	)

	var (
		vpcExtLr        *ovn_nb.LogicalRouter
		vpcExtLs        *ovn_nb.LogicalSwitch
		vpcR1extp       *ovn_nb.LogicalRouterPort
		vpcExtr1p       *ovn_nb.LogicalSwitchPort
		vpcR2extp       *ovn_nb.LogicalRouterPort
		vpcExtr2p       *ovn_nb.LogicalSwitchPort
		vpcDefaultRoute *ovn_nb.LogicalRouterStaticRoute
	)
	if hasDistgw || hasEipgw {
		vpcExtLr = &ovn_nb.LogicalRouter{
			Name: vpcExtLrName(vpc.Id),
		}
		vpcExtLs = &ovn_nb.LogicalSwitch{
			Name: vpcExtLsName(vpc.Id),
		}
		vpcR1extp = &ovn_nb.LogicalRouterPort{
			Name:     vpcR1extpName(vpc.Id),
			Mac:      apis.VpcInterExtMac1,
			Networks: []string{fmt.Sprintf("%s/%d", apis.VpcInterExtIP1(), apis.VpcInterExtMask)},
		}
		vpcExtr1p = &ovn_nb.LogicalSwitchPort{
			Name:      vpcExtr1pName(vpc.Id),
			Type:      "router",
			Addresses: []string{"router"},
			Options: map[string]string{
				"router-port": vpcR1extpName(vpc.Id),
			},
		}
		vpcR2extp = &ovn_nb.LogicalRouterPort{
			Name:     vpcR2extpName(vpc.Id),
			Mac:      apis.VpcInterExtMac2,
			Networks: []string{fmt.Sprintf("%s/%d", apis.VpcInterExtIP2(), apis.VpcInterExtMask)},
		}
		vpcExtr2p = &ovn_nb.LogicalSwitchPort{
			Name:      vpcExtr2pName(vpc.Id),
			Type:      "router",
			Addresses: []string{"router"},
			Options: map[string]string{
				"router-port": vpcR2extpName(vpc.Id),
			},
		}
		vpcDefaultRoute = &ovn_nb.LogicalRouterStaticRoute{
			Policy:     ptr("dst-ip"),
			IpPrefix:   "0.0.0.0/0",
			Nexthop:    apis.VpcInterExtIP2().String(),
			OutputPort: ptr(vpcR1extpName(vpc.Id)),
		}
		irows = append(irows,
			vpcExtLr,
			vpcExtLs,
			vpcR1extp,
			vpcExtr1p,
			vpcR2extp,
			vpcExtr2p,
			vpcDefaultRoute,
		)
	}

	// distgw
	var (
		vpcHostLs *ovn_nb.LogicalSwitch
		vpcRhp    *ovn_nb.LogicalRouterPort
		vpcHrp    *ovn_nb.LogicalSwitchPort
	)
	if hasDistgw {
		vpcHostLs = &ovn_nb.LogicalSwitch{
			Name: vpcHostLsName(vpc.Id),
		}
		vpcRhp = &ovn_nb.LogicalRouterPort{
			Name:     vpcRhpName(vpc.Id),
			Mac:      apis.VpcMappedGatewayMac,
			Networks: []string{fmt.Sprintf("%s/%d", apis.VpcMappedGatewayIP(), apis.VpcMappedIPMask)},
		}
		vpcHrp = &ovn_nb.LogicalSwitchPort{
			Name:      vpcHrpName(vpc.Id),
			Type:      "router",
			Addresses: []string{"router"},
			Options: map[string]string{
				"router-port": vpcRhpName(vpc.Id),
			},
		}
		irows = append(irows,
			vpcHostLs,
			vpcRhp,
			vpcHrp,
		)
	}

	// eipgw
	var (
		vpcEipLs *ovn_nb.LogicalSwitch
		vpcRep   *ovn_nb.LogicalRouterPort
		vpcErp   *ovn_nb.LogicalSwitchPort
	)
	if hasEipgw {
		vpcEipLs = &ovn_nb.LogicalSwitch{
			Name: vpcEipLsName(vpc.Id),
		}
		vpcRep = &ovn_nb.LogicalRouterPort{
			Name:     vpcRepName(vpc.Id),
			Mac:      apis.VpcEipGatewayMac,
			Networks: []string{fmt.Sprintf("%s/%d", apis.VpcEipGatewayIP(), apis.VpcEipGatewayIPMask)},
		}
		vpcErp = &ovn_nb.LogicalSwitchPort{
			Name:      vpcErpName(vpc.Id),
			Type:      "router",
			Addresses: []string{"router"},
			Options: map[string]string{
				"router-port": vpcRepName(vpc.Id),
			},
		}
		irows = append(irows,
			vpcEipLs,
			vpcRep,
			vpcErp,
		)
	}

	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}
	args = append(args, ovnCreateArgs(vpcLr, vpcLr.Name)...)
	if hasDistgw || hasEipgw {
		args = append(args, ovnCreateArgs(vpcExtLr, vpcExtLr.Name)...)
		args = append(args, ovnCreateArgs(vpcExtLs, vpcExtLs.Name)...)
		args = append(args, ovnCreateArgs(vpcR1extp, vpcR1extp.Name)...)
		args = append(args, ovnCreateArgs(vpcExtr1p, vpcExtr1p.Name)...)
		args = append(args, ovnCreateArgs(vpcR2extp, vpcR2extp.Name)...)
		args = append(args, ovnCreateArgs(vpcExtr2p, vpcExtr2p.Name)...)
		args = append(args, ovnCreateArgs(vpcDefaultRoute, "vpcDefaultRoute")...)
		args = append(args, "--", "add", "Logical_Router", vpcLrName(vpc.Id), "static_routes", "@vpcDefaultRoute")
		args = append(args, "--", "add", "Logical_Switch", vpcExtLs.Name, "ports", "@"+vpcExtr1p.Name)
		args = append(args, "--", "add", "Logical_Router", vpcLr.Name, "ports", "@"+vpcR1extp.Name)
		args = append(args, "--", "add", "Logical_Switch", vpcExtLs.Name, "ports", "@"+vpcExtr2p.Name)
		args = append(args, "--", "add", "Logical_Router", vpcExtLr.Name, "ports", "@"+vpcR2extp.Name)
	}
	if hasDistgw {
		args = append(args, ovnCreateArgs(vpcHostLs, vpcHostLs.Name)...)
		args = append(args, ovnCreateArgs(vpcRhp, vpcRhp.Name)...)
		args = append(args, ovnCreateArgs(vpcHrp, vpcHrp.Name)...)
		args = append(args, "--", "add", "Logical_Switch", vpcHostLs.Name, "ports", "@"+vpcHrp.Name)
		args = append(args, "--", "add", "Logical_Router", vpcExtLr.Name, "ports", "@"+vpcRhp.Name)
	}
	if hasEipgw {
		args = append(args, ovnCreateArgs(vpcEipLs, vpcEipLs.Name)...)
		args = append(args, ovnCreateArgs(vpcRep, vpcRep.Name)...)
		args = append(args, ovnCreateArgs(vpcErp, vpcErp.Name)...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLs.Name, "ports", "@"+vpcErp.Name)
		args = append(args, "--", "add", "Logical_Router", vpcExtLr.Name, "ports", "@"+vpcRep.Name)
	}
	return keeper.cli.Must(ctx, "ClaimVpc", args)
}

func (keeper *OVNNorthboundKeeper) ClaimNetwork(ctx context.Context, network *agentmodels.Network, opts *options.Options) error {
	var (
		vpc   = network.Vpc
		rpMac = mac.HashSubnetRouterPortMac(network.Id)
		mdMac = mac.HashSubnetMetadataMac(network.Id)
		mdIp  = "169.254.169.254"
	)
	netLs := &ovn_nb.LogicalSwitch{
		Name: netLsName(network.Id),
	}
	networks := []string{
		fmt.Sprintf("%s/%d", network.GuestGateway, network.GuestIpMask),
	}
	if len(network.GuestGateway6) > 0 && network.GuestIp6Mask > 0 {
		networks = append(networks, fmt.Sprintf("%s/%d", network.GuestGateway6, network.GuestIp6Mask))
	}
	netRnp := &ovn_nb.LogicalRouterPort{
		Name:     netRnpName(network.Id),
		Mac:      rpMac,
		Networks: networks,
	}
	if len(network.GuestGateway6) > 0 && network.GuestIp6Mask > 0 {
		netRnp.Ipv6RaConfigs = map[string]string{
			"address_mode": "dhcpv6_stateful",
			// "send_periodic": "false",
		}
	}
	netNrp := &ovn_nb.LogicalSwitchPort{
		Name:      netNrpName(network.Id),
		Type:      "router",
		Addresses: []string{"router"},
		Options: map[string]string{
			"router-port": netRnpName(network.Id),
		},
	}
	netMdp := &ovn_nb.LogicalSwitchPort{
		Name:      netMdpName(network.Id),
		Type:      "localport",
		Addresses: []string{fmt.Sprintf("%s %s", mdMac, mdIp)},
	}
	var (
		vpcExtBackRoute       *ovn_nb.LogicalRouterStaticRoute
		vpcExtBackRouteIdRef  string
		vpcExtBackRoute6      *ovn_nb.LogicalRouterStaticRoute
		vpcExtBackRoute6IdRef string
	)
	{
		if ipAddr, err := netutils.NewIPV4Addr(network.GuestGateway); err != nil {
			return errors.Wrap(err, "NewIPV4Addr GuestGateway")
		} else {
			netAddr := ipAddr.NetAddr(network.GuestIpMask)
			netAddrCidr := fmt.Sprintf("%s/%d", netAddr, network.GuestIpMask)
			ocStaticRouteRef := fmt.Sprintf("static-default-routes-%s", network.Id)
			vpcExtBackRoute = &ovn_nb.LogicalRouterStaticRoute{
				Policy:     ptr("dst-ip"),
				IpPrefix:   netAddrCidr,
				Nexthop:    apis.VpcInterExtIP1().String(),
				OutputPort: ptr(vpcR2extpName(vpc.Id)),
				ExternalIds: map[string]string{
					externalKeyOcRef: ocStaticRouteRef,
				},
			}
			vpcExtBackRouteIdRef = "vpcExtBackRoute"
		}
	}
	/*if len(network.GuestGateway6) > 0 && network.GuestIp6Mask > 0 {
		if ip6Addr, err := netutils.NewIPV6Addr(network.GuestGateway6); err != nil {
			return errors.Wrap(err, "NewIPV6Addr GuestGateway6")
		} else {
			netAddr6 := ip6Addr.NetAddr(network.GuestIp6Mask)
			netAddrCidr6 := fmt.Sprintf("%s/%d", netAddr6, network.GuestIp6Mask)
			ocStaticRoute6Ref := fmt.Sprintf("static-default-routes6-%s", network.Id)
			vpcExtBackRoute6 = &ovn_nb.LogicalRouterStaticRoute{
				Policy:     ptr("dst-ip"),
				IpPrefix:   netAddrCidr6,
				Nexthop:    apis.VpcInterExtIP1().String(),
				OutputPort: ptr(vpcR2extpName(vpc.Id)),
				ExternalIds: map[string]string{
					externalKeyOcRef: ocStaticRoute6Ref,
				},
			}
			vpcExtBackRoute6IdRef = "vpcExtBackRoute6"
		}
	}*/

	var (
		args      []string
		ocVersion = fmt.Sprintf("%s.%d", network.UpdatedAt, network.UpdateVersion)
	)
	irows := []types.IRow{
		netLs,
		netRnp,
		netNrp,
		netMdp,
		vpcExtBackRoute,
	}
	if vpcExtBackRoute6 != nil {
		irows = append(irows, vpcExtBackRoute6)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}

	args = append(args, ovnCreateArgs(netLs, netLs.Name)...)
	args = append(args, ovnCreateArgs(netRnp, netRnp.Name)...)
	args = append(args, ovnCreateArgs(netNrp, netNrp.Name)...)
	args = append(args, ovnCreateArgs(netMdp, netMdp.Name)...)
	args = append(args, ovnCreateArgs(vpcExtBackRoute, vpcExtBackRouteIdRef)...)
	if vpcExtBackRoute6 != nil {
		args = append(args, ovnCreateArgs(vpcExtBackRoute6, vpcExtBackRoute6IdRef)...)
	}
	args = append(args, "--", "add", "Logical_Switch", netLs.Name, "ports", "@"+netNrp.Name, "@"+netMdp.Name)
	args = append(args, "--", "add", "Logical_Router", vpcLrName(vpc.Id), "ports", "@"+netRnp.Name)
	args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpc.Id), "static_routes", "@"+vpcExtBackRouteIdRef)
	if vpcExtBackRoute6 != nil {
		args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpc.Id), "static_routes", "@"+vpcExtBackRoute6IdRef)
	}
	return keeper.cli.Must(ctx, "ClaimNetwork", args)
}

func (keeper *OVNNorthboundKeeper) ClaimVpcHost(ctx context.Context, vpc *agentmodels.Vpc, host *agentmodels.Host) error {
	var (
		ocVersion = fmt.Sprintf("%s.%d", host.UpdatedAt, host.UpdateVersion)
	)
	vpcHostLsp := &ovn_nb.LogicalSwitchPort{
		Name:      vpcHostLspName(vpc.Id, host.Id),
		Addresses: []string{fmt.Sprintf("%s %s", mac.HashVpcHostDistgwMac(host.Id), host.OvnMappedIpAddr)},
	}
	if m := keeper.DB.LogicalSwitchPort.FindOneMatchNonZeros(vpcHostLsp); m != nil {
		m.SetExternalId(externalKeyOcVersion, ocVersion)
		return nil
	} else {
		args := []string{
			"--bare", "--columns=_uuid", "find", vpcHostLsp.OvsdbTableName(),
			fmt.Sprintf("name=%q", vpcHostLsp.Name),
		}
		res := keeper.cli.Must(ctx, "find vpcHostLsp", args)
		vpcHostLspUuid := strings.TrimSpace(res.Output)
		if vpcHostLspUuid != "" {
			return nil
		}
	}
	var args []string
	args = append(args, ovnCreateArgs(vpcHostLsp, vpcHostLsp.Name)...)
	args = append(args, "--", "add", "Logical_Switch", vpcHostLsName(vpc.Id), "ports", "@"+vpcHostLsp.Name)
	return keeper.cli.Must(ctx, "ClaimVpcHost", args)
}

func (keeper *OVNNorthboundKeeper) ClaimVpcEipgw(ctx context.Context, vpc *agentmodels.Vpc) error {
	var (
		ocVersion = fmt.Sprintf("%s.%d", vpc.UpdatedAt, vpc.UpdateVersion)
		eipgwVip  = apis.VpcEipGatewayIP3().String()
	)
	vpcEipLsp := &ovn_nb.LogicalSwitchPort{
		Name:      vpcEipLspName(vpc.Id, eipgwVip),
		Addresses: []string{fmt.Sprintf("%s %s", apis.VpcEipGatewayMac3, eipgwVip)},
	}
	if m := keeper.DB.LogicalSwitchPort.FindOneMatchNonZeros(vpcEipLsp); m != nil {
		m.SetExternalId(externalKeyOcVersion, ocVersion)
		return nil
	} else {
		args := []string{
			"--bare", "--columns=_uuid", "find", vpcEipLsp.OvsdbTableName(),
			fmt.Sprintf("name=%q", vpcEipLsp.Name),
		}
		res := keeper.cli.Must(ctx, "find vpcEipLsp", args)
		vpcEipLspUuid := strings.TrimSpace(res.Output)
		if vpcEipLspUuid != "" {
			return nil
		}
	}
	var args []string
	args = append(args, ovnCreateArgs(vpcEipLsp, vpcEipLsp.Name)...)
	args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "ports", "@"+vpcEipLsp.Name)
	return keeper.cli.Must(ctx, "ClaimVpcEipgw", args)
}

func formatNtpServers(srvs string) string {
	srv := make([]string, 0)
	for _, part := range strings.Split(srvs, ",") {
		part = strings.TrimSpace(part)
		if len(part) > 0 {
			if regutils.MatchIPAddr(part) {
				srv = append(srv, part)
			} else {
				srv = append(srv, "\""+part+"\"")
			}
		}
	}
	return strings.Join(srv, ",")
}

func generateDhcpOptions(ctx context.Context, guestnetwork *agentmodels.Guestnetwork, opts *options.Options) *ovn_nb.DHCPOptions {
	var (
		network = guestnetwork.Network
		dhcpMac = mac.HashSubnetDhcpMac(network.Id)
		mdIp    = "169.254.169.254/32"

		ocDhcpRef = fmt.Sprintf("dhcp/%s/%s", guestnetwork.GuestId, guestnetwork.Ifname)
	)

	mtu := opts.OvnUnderlayMtu
	mtu -= apis.VPC_OVN_ENCAP_COST
	var (
		leaseTime  = opts.DhcpLeaseTime
		renewTime  = opts.DhcpRenewalTime
		rebindTime = opts.DhcpRenewalTime / 2
	)
	guestStartIp, _ := netutils.NewIPV4Addr(network.GuestIpStart)
	cidr := fmt.Sprintf("%s/%d", guestStartIp.NetAddr(network.GuestIpMask).String(), network.GuestIpMask)
	dhcpopts := &ovn_nb.DHCPOptions{
		Cidr: cidr,
		Options: map[string]string{
			"server_id":  network.GuestGateway,
			"server_mac": dhcpMac,
			"router":     network.GuestGateway,
			"mtu":        fmt.Sprintf("%d", mtu),
			"lease_time": fmt.Sprintf("%d", leaseTime),
			"T1":         fmt.Sprintf("%d", renewTime),
			"T2":         fmt.Sprintf("%d", rebindTime),
		},
		ExternalIds: map[string]string{
			externalKeyOcRef: ocDhcpRef,
		},
	}
	{
		routes := []string{}
		if guestnetwork.IsDefault {
			routes = append(routes,
				mdIp, "0.0.0.0",
				"0.0.0.0/0", network.GuestGateway,
			)
		} else {
			routes = append(routes,
				cidr, "0.0.0.0",
			)
		}
		if len(routes) > 0 {
			dhcpopts.Options["classless_static_route"] = fmt.Sprintf("{%s}", strings.Join(routes, ","))
		}
	}
	{
		dnsSrvs := network.GuestDns
		if dnsSrvs == "" {
			dns, err := auth.GetDNSServers(opts.Region, "")
			if err != nil {
				// ignore the error
				// log.Errorf("auth.GetDNSServers fail %s", err)
			} else {
				dnsSrvs = strings.Join(dns, ",")
			}
		}
		if dnsSrvs == "" {
			dnsSrvs = opts.DNSServer
		}
		if len(dnsSrvs) > 0 {
			dhcpopts.Options["dns_server"] = "{" + dnsSrvs + "}"
		}
	}
	{
		dnsDomain := network.GuestDomain
		if dnsDomain == "" {
			dnsDomain = opts.DNSDomain
		}
		if len(dnsDomain) > 0 && !commonapis.IsIllegalSearchDomain(dnsDomain) {
			dhcpopts.Options["domain_name"] = "\"" + dnsDomain + "\""
		}
	}
	{
		ntpSrvs := ""
		if network.GuestNtp != "" {
			ntpSrvs = network.GuestNtp
		} else {
			ntp, err := auth.GetNTPServers(opts.Region, "")
			if err != nil {
				// ignore
				// log.Errorf("auth.GetNTPServers fail %s", err)
			} else {
				ntpSrvs = strings.Join(ntp, ",")
			}
		}
		if len(ntpSrvs) > 0 {
			// bug on OVN, should not use ntp server: QiuJian
			dhcpopts.Options["ntp_server"] = "{" + formatNtpServers(ntpSrvs) + "}"
		}
	}
	return dhcpopts
}

func generateDhcp6Options(ctx context.Context, guestnetwork *agentmodels.Guestnetwork, opts *options.Options) *ovn_nb.DHCPOptions {
	var (
		network   = guestnetwork.Network
		dhcpMac   = mac.HashSubnetDhcpMac(network.Id)
		ocDhcpRef = fmt.Sprintf("dhcp6/%s/%s", guestnetwork.GuestId, guestnetwork.Ifname)
	)

	guestStartIp6, _ := netutils.NewIPV6Addr(network.GuestIp6Start)
	cidr6 := fmt.Sprintf("%s/%d", guestStartIp6.NetAddr(network.GuestIp6Mask).String(), network.GuestIp6Mask)
	dhcpopts := &ovn_nb.DHCPOptions{
		Cidr: cidr6,
		Options: map[string]string{
			"server_id": dhcpMac,
			// "dhcpv6_stateless": "false",
		},
		ExternalIds: map[string]string{
			externalKeyOcRef: ocDhcpRef,
		},
	}
	return dhcpopts
}

func (keeper *OVNNorthboundKeeper) ClaimGuestnetwork(ctx context.Context, guestnetwork *agentmodels.Guestnetwork, opts *options.Options) error {
	var (
		// Callers assure that guestnetwork.Guest is not nil
		guest   = guestnetwork.Guest
		network = guestnetwork.Network
		vpc     = network.Vpc
		host    = guest.Host
		eip     = guestnetwork.Elasticip

		lportName       = gnpName(guestnetwork.NetworkId, guestnetwork.Ifname)
		ocVersion       = fmt.Sprintf("%s.%d", guestnetwork.UpdatedAt, guestnetwork.UpdateVersion)
		ocGnrDefaultRef = fmt.Sprintf("gnrDefault/%s/%s/%s", vpc.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		ocAclRef        = fmt.Sprintf("acl/%s/%s/%s", network.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		ocQosRef        = fmt.Sprintf("qos/%s/%s/%s", network.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		ocQosEipRef     = fmt.Sprintf("qos-eip/%s/%s/%s/v2", vpc.Id, guestnetwork.GuestId, guestnetwork.Ifname)
	)

	var (
		subIPs  = []string{guestnetwork.IpAddr}
		subIPms = []string{fmt.Sprintf("%s/%d", guestnetwork.IpAddr, guestnetwork.Network.GuestIpMask)}
	)
	for _, na := range guestnetwork.SubIPs {
		subIPs = append(subIPs, na.IpAddr)
		subIPms = append(subIPms, fmt.Sprintf("%s/%d", na.IpAddr, na.Network.GuestIpMask))
	}
	subIPms = append(subIPms, guestnetwork.Guest.GetVips()...)
	sort.Strings(subIPs[1:])
	sort.Strings(subIPms[1:])

	if len(guestnetwork.Ip6Addr) > 0 {
		// ipv6
		subIPs = append(subIPs, guestnetwork.Ip6Addr)
		subIPms = append(subIPms, fmt.Sprintf("%s/%d", guestnetwork.Ip6Addr, guestnetwork.Network.GuestIp6Mask), "fe80::/64")
	}

	gnp := &ovn_nb.LogicalSwitchPort{
		Name:      lportName,
		Addresses: []string{fmt.Sprintf("%s %s", guestnetwork.MacAddr, strings.Join(subIPs, " "))},
		Options:   map[string]string{},
	}
	if guest.SrcMacCheck.IsFalse() {
		gnp.Addresses = append(gnp.Addresses, "unknown")
		// empty, not nil, as match condition
		gnp.PortSecurity = []string{}
	} else if guest.SrcIpCheck.IsFalse() {
		gnp.PortSecurity = []string{
			guestnetwork.MacAddr,
		}
	} else {
		gnp.PortSecurity = []string{
			fmt.Sprintf("%s %s",
				guestnetwork.MacAddr,
				strings.Join(subIPms, " "),
			),
		}
	}

	var (
		dhcpOpt     = generateDhcpOptions(ctx, guestnetwork, opts)
		dhcpOptName = fmt.Sprintf("dhcp-opt-%s-%s", guestnetwork.GuestId, guestnetwork.Ifname)

		dhcp6Opt     *ovn_nb.DHCPOptions
		dhcp6OptName string
	)

	if len(guestnetwork.Ip6Addr) > 0 {
		dhcp6Opt = generateDhcp6Options(ctx, guestnetwork, opts)
		dhcp6OptName = fmt.Sprintf("dhcp6-opt-%s-%s", guestnetwork.GuestId, guestnetwork.Ifname)
	}

	var qosVif []*ovn_nb.QoS
	if bwMbps := guestnetwork.BwLimit; bwMbps > 0 {
		var (
			kbps = int64(bwMbps * 1000)
			kbur = int64(kbps * 2)
		)
		qosVif = []*ovn_nb.QoS{
			{
				Priority:  2000,
				Direction: "from-lport",
				Match:     fmt.Sprintf("inport == %q", lportName),
				Bandwidth: map[string]int64{
					"rate":  kbps,
					"burst": kbur,
				},
				ExternalIds: map[string]string{
					externalKeyOcRef: ocQosRef,
				},
			},
			{
				Priority:  1000,
				Direction: "to-lport",
				Match:     fmt.Sprintf("outport == %q", lportName),
				Bandwidth: map[string]int64{
					"rate":  kbps,
					"burst": kbur,
				},
				ExternalIds: map[string]string{
					externalKeyOcRef: ocQosRef,
				},
			},
		}
	}

	var (
		gnrDefault *ovn_nb.LogicalRouterStaticRoute
		qosEipIn   *ovn_nb.QoS
		qosEipOut  *ovn_nb.QoS
		hasQoSEip  bool
	)
	{
		gnrDefaultPolicy := "src-ip"
		if eip != nil && vpcHasEipgw(vpc) {
			gnrDefault = &ovn_nb.LogicalRouterStaticRoute{
				Policy:     &gnrDefaultPolicy,
				IpPrefix:   guestnetwork.IpAddr + "/32",
				Nexthop:    apis.VpcEipGatewayIP3().String(),
				OutputPort: ptr(vpcRepName(vpc.Id)),
				ExternalIds: map[string]string{
					externalKeyOcRef: ocGnrDefaultRef,
				},
			}
			if bwMbps := eip.Bandwidth; bwMbps > 0 {
				var (
					kbps     = int64(bwMbps * 1000)
					kbur     = int64(kbps * 2)
					eipgwVip = apis.VpcEipGatewayIP3().String()
				)
				hasQoSEip = true
				qosEipIn = &ovn_nb.QoS{
					Priority:  2000,
					Direction: "from-lport",
					Match:     fmt.Sprintf("inport == %q && ip4 && ip4.dst == %s", vpcEipLspName(vpc.Id, eipgwVip), guestnetwork.IpAddr),
					Bandwidth: map[string]int64{
						"rate":  kbps,
						"burst": kbur,
					},
					ExternalIds: map[string]string{
						externalKeyOcRef: ocQosEipRef,
					},
				}
				qosEipOut = &ovn_nb.QoS{
					Priority:  3000,
					Direction: "from-lport",
					Match:     fmt.Sprintf("inport == %q && ip4 && ip4.src == %s", vpcErpName(vpc.Id), guestnetwork.IpAddr),
					Bandwidth: map[string]int64{
						"rate":  kbps,
						"burst": kbur,
					},
					ExternalIds: map[string]string{
						externalKeyOcRef: ocQosEipRef,
					},
				}
			}

		} else if vpcHasDistgw(vpc) {
			gnrDefault = &ovn_nb.LogicalRouterStaticRoute{
				Policy:     &gnrDefaultPolicy,
				IpPrefix:   guestnetwork.IpAddr + "/32",
				Nexthop:    host.OvnMappedIpAddr,
				OutputPort: ptr(vpcRhpName(vpc.Id)),
				ExternalIds: map[string]string{
					externalKeyOcRef: ocGnrDefaultRef,
				},
			}
		}
	}

	var acls []*ovn_nb.ACL
	{
		enableIPv6 := false
		if len(guestnetwork.Ip6Addr) > 0 {
			enableIPv6 = true
		}
		sgrs := guest.OrderedSecurityGroupRules()
		for _, sgr := range sgrs {
			// kvm not support peer secgroup
			acl, err := ruleToAcl(lportName, sgr, enableIPv6)
			if err != nil {
				log.Errorf("converting security group rule to acl: %v", err)
				break
			}
			acl.ExternalIds = map[string]string{
				externalKeyOcRef: ocAclRef,
			}
			acls = append(acls, acl)
		}
	}

	irows := []types.IRow{
		gnp,
		dhcpOpt,
	}
	if len(guestnetwork.Ip6Addr) > 0 {
		irows = append(irows, dhcp6Opt)
	}
	if gnrDefault != nil {
		irows = append(irows, gnrDefault)
	}
	for _, acl := range acls {
		irows = append(irows, acl)
	}
	for _, qos := range qosVif {
		irows = append(irows, qos)
	}
	if hasQoSEip {
		irows = append(irows, qosEipIn, qosEipOut)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}

	args = append(args, ovnCreateArgs(gnp, gnp.Name)...)
	args = append(args, ovnCreateArgs(dhcpOpt, dhcpOptName)...)
	args = append(args, "--", "add", "Logical_Switch_Port", gnp.Name, "dhcpv4_options", "@"+dhcpOptName)
	if len(guestnetwork.Ip6Addr) > 0 {
		args = append(args, ovnCreateArgs(dhcp6Opt, dhcp6OptName)...)
		args = append(args, "--", "add", "Logical_Switch_Port", gnp.Name, "dhcpv6_options", "@"+dhcp6OptName)
	}
	args = append(args, "--", "add", "Logical_Switch", netLsName(guestnetwork.NetworkId), "ports", "@"+gnp.Name)
	if gnrDefault != nil {
		args = append(args, ovnCreateArgs(gnrDefault, "gnrDefault")...)
		args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpc.Id), "static_routes", "@gnrDefault")
	}
	for i, acl := range acls {
		ref := fmt.Sprintf("acl%d", i)
		args = append(args, ovnCreateArgs(acl, ref)...)
		args = append(args, "--", "add", "Logical_Switch", netLsName(guestnetwork.NetworkId), "acls", "@"+ref)
	}
	for i, qos := range qosVif {
		ref := fmt.Sprintf("qosVif%d", i)
		args = append(args, ovnCreateArgs(qos, ref)...)
		args = append(args, "--", "add", "Logical_Switch", netLsName(guestnetwork.NetworkId), "qos_rules", "@"+ref)
	}
	if hasQoSEip {
		args = append(args, ovnCreateArgs(qosEipIn, "qosEipIn")...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "qos_rules", "@qosEipIn")
		args = append(args, ovnCreateArgs(qosEipOut, "qosEipOut")...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "qos_rules", "@qosEipOut")
	}
	return keeper.cli.Must(ctx, "ClaimGuestnetwork", args)
}

func (keeper *OVNNorthboundKeeper) ClaimRoutes(ctx context.Context, vpc *agentmodels.Vpc, routes resolvedRoutes) error {
	var irows []types.IRow
	for _, route := range routes {
		irows = append(irows, &ovn_nb.LogicalRouterStaticRoute{
			Policy:   ptr("dst-ip"),
			IpPrefix: route.Cidr,
			Nexthop:  route.NextHop,
		})
	}
	ocVersion := fmt.Sprintf("%s.%d", vpc.UpdatedAt, vpc.UpdateVersion)
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}
	lrName := vpcLrName(vpc.Id)
	for i, irow := range irows {
		ref := fmt.Sprintf("r%d", i)
		args = append(args, ovnCreateArgs(irow, ref)...)
		args = append(args, "--", "add", "Logical_Router", lrName, "static_routes", "@"+ref)
	}
	if len(args) > 0 {
		return keeper.cli.Must(ctx, "ClaimGuestnetwork", args)
	}
	return nil
}

func (keeper *OVNNorthboundKeeper) ClaimLoadbalancerNetwork(ctx context.Context, loadbalancerNetwork *agentmodels.LoadbalancerNetwork) error {
	var (
		// Callers assure that loadbalancerNetwork.Network is not nil
		network      = loadbalancerNetwork.Network
		vpc          = network.Vpc
		lbIntIp      = loadbalancerNetwork.IpAddr
		lbIntIpMask  = network.GuestIpMask
		lbIntMacAddr = loadbalancerNetwork.MacAddr
		lbId         = loadbalancerNetwork.LoadbalancerId
		lbUpdVer     = loadbalancerNetwork.UpdateVersion
		lbUpdAt      = loadbalancerNetwork.UpdatedAt
		networkId    = network.Id
		vpcId        = vpc.Id
		vpcHasEipgw  = vpcHasEipgw(vpc)
		eip          = loadbalancerNetwork.Elasticip

		lportName       = lbpName(lbId)
		ocVersion       = fmt.Sprintf("%s.%d", lbUpdAt, lbUpdVer)
		ocLnrDefaultRef = fmt.Sprintf("lnrDefault/%s/%s", vpcId, lbId)
		ocQosEipRef     = fmt.Sprintf("qos-eip/%s/%s/v2", vpcId, lbId)
	)

	lbp := &ovn_nb.LogicalSwitchPort{
		Name:         lportName,
		Addresses:    []string{fmt.Sprintf("%s %s", lbIntMacAddr, lbIntIp)},
		PortSecurity: []string{fmt.Sprintf("%s %s/%d", lbIntMacAddr, lbIntIp, lbIntIpMask)},
	}

	var (
		lnrDefault *ovn_nb.LogicalRouterStaticRoute
		qosEipIn   *ovn_nb.QoS
		qosEipOut  *ovn_nb.QoS
		hasQoSEip  bool
	)
	if eip != nil && vpcHasEipgw {
		lnrDefault = &ovn_nb.LogicalRouterStaticRoute{
			Policy:     ptr("src-ip"),
			IpPrefix:   lbIntIp + "/32",
			Nexthop:    apis.VpcEipGatewayIP3().String(),
			OutputPort: ptr(vpcRepName(vpcId)),
			ExternalIds: map[string]string{
				externalKeyOcRef: ocLnrDefaultRef,
			},
		}
		if bwMbps := eip.Bandwidth; bwMbps > 0 {
			var (
				kbps     = int64(bwMbps * 1000)
				kbur     = int64(kbps * 2)
				eipgwVip = apis.VpcEipGatewayIP3().String()
			)
			hasQoSEip = true
			qosEipIn = &ovn_nb.QoS{
				Priority:  2000,
				Direction: "from-lport",
				Match:     fmt.Sprintf("inport == %q && ip4 && ip4.dst == %s", vpcEipLspName(vpcId, eipgwVip), lbIntIp),
				Bandwidth: map[string]int64{
					"rate":  kbps,
					"burst": kbur,
				},
				ExternalIds: map[string]string{
					externalKeyOcRef: ocQosEipRef,
				},
			}
			qosEipOut = &ovn_nb.QoS{
				Priority:  3000,
				Direction: "from-lport",
				Match:     fmt.Sprintf("inport == %q", lportName),
				Bandwidth: map[string]int64{
					"rate":  kbps,
					"burst": kbur,
				},
				ExternalIds: map[string]string{
					externalKeyOcRef: ocQosEipRef,
				},
			}
		}
	}
	var acls []*ovn_nb.ACL
	{
		lblisteners := loadbalancerNetwork.OrderedLoadbalancerListeners()
		for _, lblistener := range lblisteners {
			acls0, err := lblistenerToAcls(lportName, lblistener)
			if err != nil {
				log.Errorf("converting acl for listener %s(%s): %v",
					lblistener.Name, lblistener.Id, err,
				)
				continue
			}
			acls = append(acls, acls0...)
		}
	}

	irows := []types.IRow{
		lbp,
	}
	if lnrDefault != nil {
		irows = append(irows, lnrDefault)
	}
	if hasQoSEip {
		irows = append(irows, qosEipIn, qosEipOut)
	}
	for _, acl := range acls {
		irows = append(irows, acl)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}

	args = append(args, ovnCreateArgs(lbp, lbp.Name)...)
	args = append(args, "--", "add", "Logical_Switch", netLsName(networkId), "ports", "@"+lbp.Name)
	if lnrDefault != nil {
		args = append(args, ovnCreateArgs(lnrDefault, "lnrDefault")...)
		args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpcId), "static_routes", "@lnrDefault")
	}
	if hasQoSEip {
		args = append(args, ovnCreateArgs(qosEipIn, "qosEipIn")...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpcId), "qos_rules", "@qosEipIn")
		args = append(args, ovnCreateArgs(qosEipOut, "qosEipOut")...)
		args = append(args, "--", "add", "Logical_Switch", netLsName(networkId), "qos_rules", "@qosEipOut")
	}
	for i, acl := range acls {
		ref := fmt.Sprintf("acl%d", i)
		args = append(args, ovnCreateArgs(acl, ref)...)
		args = append(args, "--", "add", "Logical_Switch", netLsName(networkId), "acls", "@"+ref)
	}
	return keeper.cli.Must(ctx, "ClaimLoadbalancerNetwork", args)
}

func (keeper *OVNNorthboundKeeper) ClaimVpcGuestDnsRecords(ctx context.Context, vpc *agentmodels.Vpc) error {
	var (
		grs = map[string][]string{}
		has = map[string]struct{}{}
	)
	for _, network := range vpc.Networks {
		hasValid := false
		for _, guestnetwork := range network.Guestnetworks {
			if guest := guestnetwork.Guest; guest != nil {
				var (
					name = guest.Hostname
					ip   = guestnetwork.IpAddr
				)
				grs[name] = append(grs[name], ip)
				if !hasValid {
					hasValid = true
				}
			}
		}
		if hasValid {
			has[network.Id] = struct{}{}
		}
	}

	if len(has) > 0 {
		var (
			grs_      = map[string]string{}
			ocVersion = fmt.Sprintf("%s.%d", vpc.Id, vpc.UpdateVersion)
		)
		for name, addrs := range grs {
			sort.Strings(addrs)
			grs_[name] = strings.Join(addrs, " ")
		}
		dns := &ovn_nb.DNS{
			Records: grs_,
		}
		allFound, args := cmp(&keeper.DB, ocVersion, dns)
		if allFound {
			return nil
		}
		args = append(args, ovnCreateArgs(dns, "dns")...)
		for networkId := range has {
			args = append(args, "--", "add", "Logical_Switch", netLsName(networkId), "dns_records", "@dns")
		}
		return keeper.cli.Must(ctx, "ClaimVpcGuestDnsRecords", args)
	}
	return nil
}

func (keeper *OVNNorthboundKeeper) ClaimDnsRecords(ctx context.Context, vpcs agentmodels.Vpcs, dnsrecords agentmodels.DnsRecords) error {
	var (
		names = map[string][]string{}
	)
	for _, dnsrecord := range dnsrecords {
		if !dnsrecord.Enabled.Bool() {
			continue
		}
		name := fmt.Sprintf("%s.%s", dnsrecord.Name, dnsrecord.DnsZone.Name)
		if utils.IsInStringArray(dnsrecord.DnsType, []string{"A", "AAAA"}) {
			names[name] = append(names[name], dnsrecord.DnsValue)
		}
	}
	if len(names) == 0 {
		return nil
	}

	var (
		has = map[string]struct{}{}
	)
	for _, vpc := range vpcs {
		if vpc.Id == apis.DEFAULT_VPC_ID {
			continue
		}
		for _, network := range vpc.Networks {
			if len(network.Guestnetworks) > 0 {
				has[network.Id] = struct{}{}
			}
		}
	}
	if len(has) == 0 {
		return nil
	}

	var (
		names_    = map[string]string{}
		ocVersion = "dnsrecords"
	)
	for name, addrs := range names {
		names_[name] = strings.Join(addrs, " ")
	}
	dns := &ovn_nb.DNS{
		Records: names_,
	}
	allFound, args := cmp(&keeper.DB, ocVersion, dns)
	if allFound {
		return nil
	}
	args = append(args, ovnCreateArgs(dns, "dns")...)
	for networkId := range has {
		args = append(args, "--", "add", "Logical_Switch", netLsName(networkId), "dns_records", "@dns")
	}
	return keeper.cli.Must(ctx, "ClaimDnsRecords", args)
}

func (keeper *OVNNorthboundKeeper) ClaimGroupnetwork(ctx context.Context, groupnetwork *agentmodels.Groupnetwork) error {
	var (
		network = groupnetwork.Network
		vpc     = network.Vpc
		eip     = groupnetwork.Elasticip

		lportName       = vipName(groupnetwork.NetworkId, groupnetwork.GroupId, groupnetwork.IpAddr)
		ocVersion       = fmt.Sprintf("vip.%s.%d", groupnetwork.UpdatedAt, groupnetwork.UpdateVersion)
		ocGnrDefaultRef = fmt.Sprintf("gnrDefault-vip/%s/%s/%s", vpc.Id, groupnetwork.GroupId, groupnetwork.IpAddr)
		ocAclRef        = fmt.Sprintf("acl-eip/%s/%s/%s", network.Id, groupnetwork.GroupId, groupnetwork.IpAddr)
		ocQosEipRef     = fmt.Sprintf("qos-eip-vip/%s/%s/%s/v2", vpc.Id, groupnetwork.GroupId, groupnetwork.IpAddr)
	)

	gns := groupnetwork.GetGuestNetworks()
	gnsNames := make([]string, len(gns))
	for i, gn := range gns {
		gnsNames[i] = gnpName(gn.NetworkId, gn.Ifname)
	}
	sort.Strings(gnsNames)
	gnp := &ovn_nb.LogicalSwitchPort{
		Name: lportName,
		Type: "virtual",
		Options: map[string]string{
			"virtual-ip":      groupnetwork.IpAddr,
			"virtual-parents": strings.Join(gnsNames, ","),
		},
		PortSecurity: []string{},
	}

	var (
		gnrDefault *ovn_nb.LogicalRouterStaticRoute
		qosEipIn   *ovn_nb.QoS
		qosEipOut  *ovn_nb.QoS
		hasQoSEip  bool
	)
	{
		gnrDefaultPolicy := "src-ip"
		if eip != nil && vpcHasEipgw(vpc) {
			log.Infof("groupnetwork %s has eip %s", groupnetwork.IpAddr, eip.IpAddr)
			gnrDefault = &ovn_nb.LogicalRouterStaticRoute{
				Policy:     &gnrDefaultPolicy,
				IpPrefix:   groupnetwork.IpAddr + "/32",
				Nexthop:    apis.VpcEipGatewayIP3().String(),
				OutputPort: ptr(vpcRepName(vpc.Id)),
				ExternalIds: map[string]string{
					externalKeyOcRef: ocGnrDefaultRef,
				},
			}
			if bwMbps := eip.Bandwidth; bwMbps > 0 {
				var (
					kbps     = int64(bwMbps * 1000)
					kbur     = int64(kbps * 2)
					eipgwVip = apis.VpcEipGatewayIP3().String()
				)
				hasQoSEip = true
				qosEipIn = &ovn_nb.QoS{
					Priority:  2000,
					Direction: "from-lport",
					Match:     fmt.Sprintf("inport == %q && ip4 && ip4.dst == %s", vpcEipLspName(vpc.Id, eipgwVip), groupnetwork.IpAddr),
					Bandwidth: map[string]int64{
						"rate":  kbps,
						"burst": kbur,
					},
					ExternalIds: map[string]string{
						externalKeyOcRef: ocQosEipRef,
					},
				}
				qosEipOut = &ovn_nb.QoS{
					Priority:  3000,
					Direction: "from-lport",
					Match:     fmt.Sprintf("inport == %q && ip4 && ip4.src == %s", vpcErpName(vpc.Id), groupnetwork.IpAddr),
					Bandwidth: map[string]int64{
						"rate":  kbps,
						"burst": kbur,
					},
					ExternalIds: map[string]string{
						externalKeyOcRef: ocQosEipRef,
					},
				}
			}
		}
	}

	var acl *ovn_nb.ACL
	{
		acl = &ovn_nb.ACL{
			Priority:  1,
			Direction: aclDirToLport,
			Match:     fmt.Sprintf(`is_chassis_resident("%s") && ip4`, gnp.Name),
			Action:    "allow-related",
			ExternalIds: map[string]string{
				externalKeyOcRef: ocAclRef,
			},
		}
	}

	irows := []types.IRow{
		gnp,
		acl,
	}
	if gnrDefault != nil {
		irows = append(irows, gnrDefault)
	}
	if hasQoSEip {
		irows = append(irows, qosEipIn, qosEipOut)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}

	args = append(args, ovnCreateArgs(gnp, gnp.Name)...)
	args = append(args, "--", "add", "Logical_Switch", netLsName(groupnetwork.NetworkId), "ports", "@"+gnp.Name)
	aclRef := "vipacl"
	args = append(args, ovnCreateArgs(acl, aclRef)...)
	args = append(args, "--", "add", "Logical_Switch", netLsName(groupnetwork.NetworkId), "acls", "@"+aclRef)
	if gnrDefault != nil {
		args = append(args, ovnCreateArgs(gnrDefault, "vipGnrDefault")...)
		args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpc.Id), "static_routes", "@vipGnrDefault")
	}
	if hasQoSEip {
		args = append(args, ovnCreateArgs(qosEipIn, "vipQosEipIn")...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "qos_rules", "@vipQosEipIn")
		args = append(args, ovnCreateArgs(qosEipOut, "vipQosEipOut")...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "qos_rules", "@vipQosEipOut")
	}

	return keeper.cli.Must(ctx, "ClaimGroupnetworks", args)
}

func (keeper *OVNNorthboundKeeper) Mark(ctx context.Context) {
	db := &keeper.DB
	itbls := []types.ITable{
		&db.LogicalSwitch,
		&db.LogicalSwitchPort,
		&db.LogicalRouter,
		&db.LogicalRouterPort,
		&db.LogicalRouterStaticRoute,
		&db.ACL,
		&db.DHCPOptions,
		&db.QoS,
		&db.DNS,
	}
	for _, itbl := range itbls {
		for _, irow := range itbl.Rows() {
			irow.RemoveExternalId(externalKeyOcVersion)
		}
	}
}

func (keeper *OVNNorthboundKeeper) Sweep(ctx context.Context) error {
	db := &keeper.DB
	// isRoot=false tables at the end
	itbls := []types.ITable{
		&db.LogicalSwitchPort,
		&db.LogicalRouterPort,
		&db.LogicalSwitch,
		&db.LogicalRouter,
		&db.DHCPOptions,
		&db.DNS,
	}
	var irows []types.IRow
	for _, itbl := range itbls {
		for _, irow := range itbl.Rows() {
			_, ok := irow.GetExternalId(externalKeyOcVersion)
			if !ok {
				irows = append(irows, irow)
			}
		}
	}
	args := ovnutil.OvnNbctlArgsDestroy(irows)
	if len(args) > 0 {
		keeper.cli.Must(ctx, "Sweep", args)
	}

	{
		var args []string
		for _, irow := range db.LogicalRouterStaticRoute.Rows() {
			_, ok := irow.GetExternalId(externalKeyOcVersion)
			if !ok {
				for _, lr := range db.LogicalRouter.FindLogicalRouterStaticRouteReferrer_static_routes(irow.OvsdbUuid()) {
					args = append(args, "--", "--if-exists", "remove", "Logical_Router", lr.Name, "static_routes", irow.OvsdbUuid())
				}
			}
		}
		if len(args) > 0 {
			keeper.cli.Must(ctx, "Sweep static routes", args)
		}
	}
	{
		var args []string
		for _, irow := range db.ACL.Rows() {
			_, ok := irow.GetExternalId(externalKeyOcVersion)
			if !ok {
				for _, ls := range db.LogicalSwitch.FindACLReferrer_acls(irow.OvsdbUuid()) {
					args = append(args, "--", "--if-exists", "remove", "Logical_Switch", ls.Name, "acls", irow.OvsdbUuid())
				}
			}
		}
		if len(args) > 0 {
			keeper.cli.Must(ctx, "Sweep acls", args)
		}
	}
	{ //  remove unused QoS rows
		var args []string
		for _, irow := range db.QoS.Rows() {
			_, ok := irow.GetExternalId(externalKeyOcVersion)
			if !ok {
				for _, ls := range db.LogicalSwitch.FindQoSReferrer_qos_rules(irow.OvsdbUuid()) {
					args = append(args, "--", "--if-exists", "remove", "Logical_Switch", ls.Name, "qos_rules", irow.OvsdbUuid())
				}
			}
		}
		if len(args) > 0 {
			keeper.cli.Must(ctx, "Sweep qos", args)
		}
	}
	return nil
}
