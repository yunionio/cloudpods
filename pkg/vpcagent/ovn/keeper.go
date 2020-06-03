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

	// distgw
	var (
		vpcHostLs *ovn_nb.LogicalSwitch
		vpcRhp    *ovn_nb.LogicalRouterPort
		vpcHrp    *ovn_nb.LogicalSwitchPort
	)
	if vpcHasDistgw(vpc) {
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
	if vpcHasEipgw(vpc) {
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
	if vpcHasDistgw(vpc) {
		args = append(args, ovnCreateArgs(vpcHostLs, vpcHostLs.Name)...)
		args = append(args, ovnCreateArgs(vpcRhp, vpcRhp.Name)...)
		args = append(args, ovnCreateArgs(vpcHrp, vpcHrp.Name)...)
		args = append(args, "--", "add", "Logical_Switch", vpcHostLs.Name, "ports", "@"+vpcHrp.Name)
		args = append(args, "--", "add", "Logical_Router", vpcLr.Name, "ports", "@"+vpcRhp.Name)
	}
	if vpcHasEipgw(vpc) {
		args = append(args, ovnCreateArgs(vpcEipLs, vpcEipLs.Name)...)
		args = append(args, ovnCreateArgs(vpcRep, vpcRep.Name)...)
		args = append(args, ovnCreateArgs(vpcErp, vpcErp.Name)...)
		args = append(args, "--", "add", "Logical_Switch", vpcEipLs.Name, "ports", "@"+vpcErp.Name)
		args = append(args, "--", "add", "Logical_Router", vpcLr.Name, "ports", "@"+vpcRep.Name)
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
	dhcpopts := &ovn_nb.DHCPOptions{
		Cidr: fmt.Sprintf("%s/%d", network.GuestIpStart, network.GuestIpMask),
		Options: map[string]string{
			"server_id":              network.GuestGateway,
			"server_mac":             dhcpMac,
			"lease_time":             fmt.Sprintf("%d", 86400),
			"router":                 network.GuestGateway,
			"classless_static_route": fmt.Sprintf("{%s}", strings.Join(routes, ",")),
			"mtu":                    fmt.Sprintf("%d", mtu),
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

	gnp := &ovn_nb.LogicalSwitchPort{
		Name:          lportName,
		Addresses:     []string{fmt.Sprintf("%s %s", guestnetwork.MacAddr, guestnetwork.IpAddr)},
		PortSecurity:  []string{fmt.Sprintf("%s %s/%d", guestnetwork.MacAddr, guestnetwork.IpAddr, guestnetwork.Network.GuestIpMask)},
		Dhcpv4Options: &dhcpOpt,
		Options:       map[string]string{},
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
		ptr := func(s string) *string {
			return &s
		}
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
		args = append(args, "--", "add", "Logical_Router", vpcLrName(vpc.Id), "static_routes", "@gnrDefault")
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
		for _, guestnetwork := range network.Guestnetworks {
			var (
				guest = guestnetwork.Guest
				ip    = guestnetwork.IpAddr
			)
			grs[guest.Name] = append(grs[guest.Name], ip)
		}
		if len(network.Guestnetworks) > 0 {
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
