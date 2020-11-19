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

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
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
		vpcExtLr           *ovn_nb.LogicalRouter
		vpcExtLs           *ovn_nb.LogicalSwitch
		vpcR1extp          *ovn_nb.LogicalRouterPort
		vpcExtr1p          *ovn_nb.LogicalSwitchPort
		vpcR2extp          *ovn_nb.LogicalRouterPort
		vpcExtr2p          *ovn_nb.LogicalSwitchPort
		vpcDefaultRoute    *ovn_nb.LogicalRouterStaticRoute
		vpcExtDefaultRoute *ovn_nb.LogicalRouterStaticRoute
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
		vpcExtDefaultRoute = &ovn_nb.LogicalRouterStaticRoute{
			Policy:     ptr("dst-ip"),
			IpPrefix:   "0.0.0.0/0",
			Nexthop:    apis.VpcInterExtIP1().String(),
			OutputPort: ptr(vpcR2extpName(vpc.Id)),
		}
		irows = append(irows,
			vpcExtLr,
			vpcExtLs,
			vpcR1extp,
			vpcExtr1p,
			vpcR2extp,
			vpcExtr2p,
			vpcDefaultRoute,
			vpcExtDefaultRoute,
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
		args = append(args, ovnCreateArgs(vpcExtDefaultRoute, "vpcExtDefaultRoute")...)
		args = append(args, "--", "add", "Logical_Router", vpcLrName(vpc.Id), "static_routes", "@vpcDefaultRoute")
		args = append(args, "--", "add", "Logical_Router", vpcExtLrName(vpc.Id), "static_routes", "@vpcExtDefaultRoute")
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

func (keeper *OVNNorthboundKeeper) ClaimNetwork(ctx context.Context, network *agentmodels.Network, mtu int) error {
	var (
		rpMac   = mac.HashMac(network.Id, "rp")
		dhcpMac = mac.HashMac(network.Id, "dhcp")
		mdMac   = mac.HashMac(network.Id, "md")
		mdIp    = "169.254.169.254"
	)
	netLs := &ovn_nb.LogicalSwitch{
		Name: netLsName(network.Id),
	}
	netRnp := &ovn_nb.LogicalRouterPort{
		Name:     netRnpName(network.Id),
		Mac:      rpMac,
		Networks: []string{fmt.Sprintf("%s/%d", network.GuestGateway, network.GuestIpMask)},
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
	routes := []string{
		mdIp, "0.0.0.0",
		"0.0.0.0/0", network.GuestGateway,
	}
	mtu -= 58
	const (
		leaseTime  = 86400 * 365 * 3
		renewTime  = 86400
		rebindTime = 86400 * 3
	)
	dhcpopts := &ovn_nb.DHCPOptions{
		Cidr: fmt.Sprintf("%s/%d", network.GuestIpStart, network.GuestIpMask),
		Options: map[string]string{
			"server_id":              network.GuestGateway,
			"server_mac":             dhcpMac,
			"router":                 network.GuestGateway,
			"classless_static_route": fmt.Sprintf("{%s}", strings.Join(routes, ",")),
			"mtu":                    fmt.Sprintf("%d", mtu),
			"lease_time":             fmt.Sprintf("%d", leaseTime),
			"T1":                     fmt.Sprintf("%d", renewTime),
			"T2":                     fmt.Sprintf("%d", rebindTime),
		},
		ExternalIds: map[string]string{
			externalKeyOcRef: network.Id,
		},
	}
	if network.GuestDns != "" {
		dhcpopts.Options["dns_server"] = "{" + network.GuestDns + "}"
	} else {
		dhcpopts.Options["dns_server"] = "{223.5.5.5,223.6.6.6}"
	}

	var (
		args      []string
		ocVersion = fmt.Sprintf("%s.%d", network.UpdatedAt, network.UpdateVersion)
	)
	allFound, args := cmp(&keeper.DB, ocVersion,
		netLs,
		netRnp,
		netNrp,
		netMdp,
		dhcpopts,
	)
	if allFound {
		return nil
	}
	args = append(args, ovnCreateArgs(netLs, netLs.Name)...)
	args = append(args, ovnCreateArgs(netRnp, netRnp.Name)...)
	args = append(args, ovnCreateArgs(netNrp, netNrp.Name)...)
	args = append(args, ovnCreateArgs(netMdp, netMdp.Name)...)
	args = append(args, ovnCreateArgs(dhcpopts, "dhcpopts")...)
	args = append(args, "--", "add", "Logical_Switch", netLs.Name, "ports", "@"+netNrp.Name, "@"+netMdp.Name)
	args = append(args, "--", "add", "Logical_Router", vpcLrName(network.Vpc.Id), "ports", "@"+netRnp.Name)
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

func (keeper *OVNNorthboundKeeper) ClaimGuestnetwork(ctx context.Context, guestnetwork *agentmodels.Guestnetwork) error {
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
		ocQosEipRef     = fmt.Sprintf("qos-eip/%s/%s/%s", vpc.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		dhcpOpt         string
	)

	{
		dhcpOptQuery := &ovn_nb.DHCPOptions{
			ExternalIds: map[string]string{
				externalKeyOcRef: guestnetwork.NetworkId,
			},
		}
		if m := keeper.DB.DHCPOptions.FindOneMatchNonZeros(dhcpOptQuery); m != nil {
			dhcpOpt = m.OvsdbUuid()
		} else {
			args := []string{
				"--bare", "--columns=_uuid", "find", "DHCP_Options",
				fmt.Sprintf("external_ids:%s=%q", externalKeyOcRef, guestnetwork.NetworkId),
			}
			res := keeper.cli.Must(ctx, "find dhcpopt", args)
			dhcpOpt = strings.TrimSpace(res.Output)
		}
		if dhcpOpt == "" {
			return fmt.Errorf("cannot find dhcpopt for subnet %s", guestnetwork.NetworkId)
		}
	}

	var (
		subIPs  = []string{guestnetwork.IpAddr}
		subIPms = []string{fmt.Sprintf("%s/%d", guestnetwork.IpAddr, guestnetwork.Network.GuestIpMask)}
	)
	for _, na := range guestnetwork.SubIPs {
		subIPs = append(subIPs, na.IpAddr)
		subIPms = append(subIPms, fmt.Sprintf("%s/%d", na.IpAddr, na.Network.GuestIpMask))
	}
	sort.Strings(subIPs[1:])
	sort.Strings(subIPms[1:])
	gnp := &ovn_nb.LogicalSwitchPort{
		Name:          lportName,
		Addresses:     []string{fmt.Sprintf("%s %s", guestnetwork.MacAddr, strings.Join(subIPs, " "))},
		Dhcpv4Options: &dhcpOpt,
		Options:       map[string]string{},
	}
	if guest.SrcMacCheck.IsFalse() {
		gnp.Addresses = append(gnp.Addresses, "unknown")
		// empty, not nil, as match condition
		gnp.PortSecurity = []string{}
	} else if guest.SrcIpCheck.IsFalse() {
		gnp.PortSecurity = []string{
			fmt.Sprintf("%s", guestnetwork.MacAddr),
		}
	} else {
		gnp.PortSecurity = []string{
			fmt.Sprintf("%s %s",
				guestnetwork.MacAddr,
				strings.Join(subIPms, " "),
			),
		}
	}

	var qosVif []*ovn_nb.QoS
	if bwMbps := guestnetwork.BwLimit; bwMbps > 0 {
		var (
			kbps = int64(bwMbps * 1000)
			kbur = int64(kbps * 2)
		)
		qosVif = []*ovn_nb.QoS{
			&ovn_nb.QoS{
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
			&ovn_nb.QoS{
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
		qosEip     []*ovn_nb.QoS
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
				qosEip = []*ovn_nb.QoS{
					&ovn_nb.QoS{
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
					},
					&ovn_nb.QoS{
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
		sgrs := guest.OrderedSecurityGroupRules()
		for _, sgr := range sgrs {
			acl, err := ruleToAcl(lportName, sgr)
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
	for _, qos := range qosEip {
		irows = append(irows, qos)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}

	args = append(args, ovnCreateArgs(gnp, gnp.Name)...)
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
	for i, qos := range qosEip {
		ref := fmt.Sprintf("qosEip%d", i)
		args = append(args, ovnCreateArgs(qos, ref)...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLsName(vpc.Id), "qos_rules", "@"+ref)
	}
	return keeper.cli.Must(ctx, "ClaimGuestnetwork", args)
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
					name = guest.Name
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
		name := dnsrecord.Name
		for _, r := range dnsrecord.GetInfo() {
			prefs := []string{"A:", "AAAA:"}
			for _, pref := range prefs {
				if strings.HasPrefix(r, pref) {
					names[name] = append(names[name], r[len(pref):])
				}
			}
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
