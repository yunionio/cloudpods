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
	vpcHostLs := &ovn_nb.LogicalSwitch{
		Name: vpcHostLsName(vpc.Id),
	}
	vpcRhp := &ovn_nb.LogicalRouterPort{
		Name:     vpcRhpName(vpc.Id),
		Mac:      apis.VpcMappedGatewayMac,
		Networks: []string{fmt.Sprintf("%s/%d", apis.VpcMappedGatewayIP(), apis.VpcMappedIPMask)},
	}
	vpcHrp := &ovn_nb.LogicalSwitchPort{
		Name:      vpcHrpName(vpc.Id),
		Type:      "router",
		Addresses: []string{"router"},
		Options: map[string]string{
			"router-port": vpcRhpName(vpc.Id),
		},
	}
	allFound, args := cmp(&keeper.DB, ocVersion,
		vpcLr,
		vpcHostLs,
		vpcRhp,
		vpcHrp,
	)
	if allFound {
		return nil
	}
	args = append(args, ovnCreateArgs(vpcLr, vpcLr.Name)...)
	args = append(args, ovnCreateArgs(vpcHostLs, vpcHostLs.Name)...)
	args = append(args, ovnCreateArgs(vpcRhp, vpcRhp.Name)...)
	args = append(args, ovnCreateArgs(vpcHrp, vpcHrp.Name)...)
	args = append(args, "--", "add", "Logical_Switch", vpcHostLs.Name, "ports", "@"+vpcHrp.Name)
	args = append(args, "--", "add", "Logical_Router", vpcLr.Name, "ports", "@"+vpcRhp.Name)
	return keeper.cli.Must(ctx, "ClaimVpc", args)
}

func (keeper *OVNNorthboundKeeper) ClaimNetwork(ctx context.Context, network *agentmodels.Network) error {
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
	dhcpopts := &ovn_nb.DHCPOptions{
		Cidr: fmt.Sprintf("%s/%d", network.GuestIpStart, network.GuestIpMask),
		Options: map[string]string{
			"server_id":              network.GuestGateway,
			"server_mac":             dhcpMac,
			"lease_time":             fmt.Sprintf("%d", 86400),
			"router":                 network.GuestGateway,
			"classless_static_route": fmt.Sprintf("{%s}", strings.Join(routes, ",")),
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

func (keeper *OVNNorthboundKeeper) ClaimGuestnetwork(ctx context.Context, guestnetwork *agentmodels.Guestnetwork) error {
	var (
		guest   = guestnetwork.Guest
		network = guestnetwork.Network
		vpc     = network.Vpc
		host    = guest.Host

		lportName = gnpName(guestnetwork.NetworkId, guestnetwork.Ifname)
		ocVersion = fmt.Sprintf("%s.%d", guestnetwork.UpdatedAt, guestnetwork.UpdateVersion)
		ocGnrRef  = fmt.Sprintf("gnr/%s/%s/%s", vpc.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		ocAclRef  = fmt.Sprintf("acl/%s/%s/%s", network.Id, guestnetwork.GuestId, guestnetwork.Ifname)
		dhcpOpt   string
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
	}

	gnrPolicy := "src-ip"
	gnr := &ovn_nb.LogicalRouterStaticRoute{
		Policy:   &gnrPolicy,
		IpPrefix: guestnetwork.IpAddr + "/32",
		Nexthop:  host.OvnMappedIpAddr,
		ExternalIds: map[string]string{
			externalKeyOcRef: ocGnrRef,
		},
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
		gnp, gnr,
	}
	for _, acl := range acls {
		irows = append(irows, acl)
	}
	allFound, args := cmp(&keeper.DB, ocVersion, irows...)
	if allFound {
		return nil
	}
	args = append(args, ovnCreateArgs(gnp, gnp.Name)...)
	args = append(args, ovnCreateArgs(gnr, "gnr")...)
	args = append(args, "--", "add", "Logical_Switch", netLsName(guestnetwork.NetworkId), "ports", "@"+gnp.Name)
	args = append(args, "--", "add", "Logical_Router", vpcLrName(vpc.Id), "static_routes", "@gnr")
	for i, acl := range acls {
		ref := fmt.Sprintf("acl%d", i)
		args = append(args, ovnCreateArgs(acl, ref)...)
		args = append(args, "--", "add", "Logical_Switch", netLsName(guestnetwork.NetworkId), "acls", "@"+ref)
	}
	return keeper.cli.Must(ctx, "ClaimGuestnetwork", args)
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
				ref, ok := irow.GetExternalId(externalKeyOcRef)
				if ok {
					parts := strings.SplitN(ref, "/", 4)
					if len(parts) == 4 {
						vpcId := parts[1]
						args = append(args, "--", "remove", "Logical_Router", vpcLrName(vpcId), "static_routes", irow.OvsdbUuid())
					}
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
				ref, ok := irow.GetExternalId(externalKeyOcRef)
				if ok {
					parts := strings.SplitN(ref, "/", 4)
					if len(parts) == 4 {
						networkId := parts[1]
						args = append(args, "--", "remove", "Logical_Switch", netLsName(networkId), "acls", irow.OvsdbUuid())
					}
				}
			}
		}
		if len(args) > 0 {
			keeper.cli.Must(ctx, "Sweep acls", args)
		}
	}
	return nil
}
