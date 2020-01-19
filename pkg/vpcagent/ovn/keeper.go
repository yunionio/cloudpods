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
	"crypto/md5"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
	"yunion.io/x/onecloud/pkg/vpcagent/ovnutil"
)

const (
	externalKeyOcVersion = "oc-version"
	externalKeyOcRef     = "oc-ref"
)

type OVNNorthboundKeeper struct {
	DB  ovnutil.OVNNorthbound
	cli *ovnutil.OvnNbCtl
}

func DumpOVNNorthbound(ctx context.Context, cli *ovnutil.OvnNbCtl) (*OVNNorthboundKeeper, error) {
	db := ovnutil.OVNNorthbound{}
	itbls := []ovnutil.ITable{
		&db.LogicalSwitch,
		&db.LogicalSwitchPort,
		&db.LogicalRouter,
		&db.LogicalRouterPort,
		&db.DHCPOptions,
	}
	args := []string{"--format=json", "list", "<tbl>"}
	for _, itbl := range itbls {
		tbl := itbl.OvnTableName()
		args[2] = tbl
		res := cli.Must(ctx, "List "+tbl, args)
		if err := ovnutil.UnmarshalJSON([]byte(res.Output), itbl); err != nil {
			return nil, errors.Wrapf(err, "Unmarshal %s:\n%s",
				itbl.OvnTableName(), res.Output)
		}
	}
	keeper := &OVNNorthboundKeeper{
		DB:  db,
		cli: cli,
	}
	return keeper, nil
}

func ovnCreateArgs(irow ovnutil.IRow, idRef string) []string {
	args := append([]string{
		"--", "--id=@" + idRef, "create", irow.OvnTableName(),
	}, irow.OvnArgs()...)
	return args
}

func (keeper *OVNNorthboundKeeper) ClaimVpc(ctx context.Context, vpc *agentmodels.Vpc) error {
	var (
		args      []string
		ocVersion = fmt.Sprintf("%s.%d", vpc.UpdatedAt, vpc.UpdateVersion)
	)

	lrvpc := &ovnutil.LogicalRouter{
		Name: fmt.Sprintf("vpc-lr-%s", vpc.Id),
	}
	if m := keeper.DB.LogicalRouter.FindOneMatchNonZeros(lrvpc); m != nil {
		m.OvnSetExternalIds(externalKeyOcVersion, ocVersion)
		return nil
	}
	args = append(args, ovnCreateArgs(lrvpc, "lrvpc")...)
	return keeper.cli.Must(ctx, "ClaimVpc", args)
}

func hashMac(in ...string) string {
	h := md5.New()
	for _, s := range in {
		h.Write([]byte(s))
	}
	sum := h.Sum(nil)
	b := sum[0]
	b &= 0xfe
	b |= 0x02
	mac := fmt.Sprintf("%02x", b)
	for _, b := range sum[1:6] {
		mac += fmt.Sprintf(":%02x", b)
	}
	return mac
}

func (keeper *OVNNorthboundKeeper) ClaimNetwork(ctx context.Context, network *agentmodels.Network) error {
	var (
		lsnetName   = fmt.Sprintf("subnet-ls-%s", network.Id)
		lrnetpName  = fmt.Sprintf("subnet-lrp-%s", network.Id)
		lsnetpName  = fmt.Sprintf("subnet-lsp-%s", network.Id)
		lsnetmpName = fmt.Sprintf("subnet-lsmp-%s", network.Id)
		lrvpcName   = fmt.Sprintf("vpc-lr-%s", network.VpcId)

		rpMac   = hashMac(network.Id, "rp")
		dhcpMac = hashMac(network.Id, "dhcp")
		mdMac   = hashMac(network.Id, "md")
		mdIp    = "169.254.169.254"
	)
	lsnet := &ovnutil.LogicalSwitch{
		Name: lsnetName,
	}
	lrnetp := &ovnutil.LogicalRouterPort{
		Name:     lrnetpName,
		Mac:      rpMac,
		Networks: []string{fmt.Sprintf("%s/%d", network.GuestGateway, network.GuestIpMask)},
	}
	lsnetp := &ovnutil.LogicalSwitchPort{
		Name:      lsnetpName,
		Type:      "router",
		Addresses: []string{"router"},
		Options: map[string]string{
			"router-port": lrnetpName,
		},
	}
	lsnetmp := &ovnutil.LogicalSwitchPort{
		Name:      lsnetmpName,
		Type:      "localport",
		Addresses: []string{fmt.Sprintf("%s %s", mdMac, mdIp)},
	}
	dhcpopts := &ovnutil.DHCPOptions{
		Cidr: fmt.Sprintf("%s/%d", network.GuestIpStart, network.GuestIpMask),
		Options: map[string]string{
			"server_id":              network.GuestGateway,
			"server_mac":             dhcpMac,
			"lease_time":             fmt.Sprintf("%d", 86400),
			"router":                 network.GuestGateway,
			"classless_static_route": fmt.Sprintf("{%s/32,0.0.0.0}", mdIp),
		},
		ExternalIds: map[string]string{
			externalKeyOcRef: network.Id,
		},
	}

	var (
		args      []string
		ocVersion = fmt.Sprintf("%s.%d", network.UpdatedAt, network.UpdateVersion)
	)
	irows := []ovnutil.IRow{
		lsnet,
		lrnetp,
		lsnetp,
		lsnetmp,
		dhcpopts,
	}
	{
		irowsFound := make([]ovnutil.IRow, 0, len(irows))
		for _, irow := range irows {
			irowFound := keeper.DB.FindOneMatchNonZeros(irow)
			if irowFound != nil {
				irowsFound = append(irowsFound, irowFound)
			}
		}
		// mark them anyway even if not all found, to avoid the destroy
		// call at sweep stage
		for _, irowFound := range irowsFound {
			irowFound.OvnSetExternalIds(externalKeyOcVersion, ocVersion)
		}
		if len(irowsFound) == len(irows) {
			return nil
		}
		args := ovnutil.OvnNbctlArgsDestroy(irowsFound)
		if len(args) > 0 {
			keeper.cli.Must(ctx, "ClaimNetwork cleanup", args)
		}
	}
	args = append(args, ovnCreateArgs(lsnet, "lsnet")...)
	args = append(args, ovnCreateArgs(lrnetp, "lrnetp")...)
	args = append(args, ovnCreateArgs(lsnetp, "lsnetp")...)
	args = append(args, ovnCreateArgs(lsnetmp, "lsnetmp")...)
	args = append(args, ovnCreateArgs(dhcpopts, "dhcpopts")...)
	args = append(args, "--", "add", "Logical_Switch", lsnetName, "ports", "@lsnetp", "@lsnetmp")
	args = append(args, "--", "add", "Logical_Router", lrvpcName, "ports", "@lrnetp")
	return keeper.cli.Must(ctx, "ClaimNetwork", args)
}

func (keeper *OVNNorthboundKeeper) ClaimGuestnetwork(ctx context.Context, guestnetwork *agentmodels.Guestnetwork) error {
	var (
		lsName    = fmt.Sprintf("subnet-ls-%s", guestnetwork.NetworkId)
		lspName   = fmt.Sprintf("iface-%s-%s", guestnetwork.NetworkId, guestnetwork.Ifname)
		ocVersion = fmt.Sprintf("%s.%d", guestnetwork.UpdatedAt, guestnetwork.UpdateVersion)
		dhcpOpt   string
	)

	{
		dhcpOptQuery := &ovnutil.DHCPOptions{
			ExternalIds: map[string]string{
				externalKeyOcRef: guestnetwork.NetworkId,
			},
		}
		if m := keeper.DB.DHCPOptions.FindOneMatchNonZeros(dhcpOptQuery); m != nil {
			dhcpOpt = m.OvnUuid()
		} else {
			args := []string{
				"--bare", "--columns=_uuid", "find", "DHCP_Options",
				fmt.Sprintf("external_ids:%s=%q", externalKeyOcRef, guestnetwork.NetworkId),
			}
			res := keeper.cli.Must(ctx, "find dhcpopt", args)
			dhcpOpt = strings.TrimSpace(res.Output)
		}
	}

	lsp := &ovnutil.LogicalSwitchPort{
		Name:          lspName,
		Addresses:     []string{fmt.Sprintf("%s %s", guestnetwork.MacAddr, guestnetwork.IpAddr)},
		PortSecurity:  []string{fmt.Sprintf("%s %s/%d", guestnetwork.MacAddr, guestnetwork.IpAddr, guestnetwork.Network.GuestIpMask)},
		Dhcpv4Options: &dhcpOpt,
	}
	if m := keeper.DB.LogicalSwitchPort.FindOneMatchNonZeros(lsp); m != nil {
		m.OvnSetExternalIds(externalKeyOcVersion, ocVersion)
		return nil
	}
	var args []string
	args = append(args, ovnCreateArgs(lsp, "lsp")...)
	args = append(args, "--", "add", "Logical_Switch", lsName, "ports", "@lsp")
	return keeper.cli.Must(ctx, "ClaimGuestnetwork", args)
}

func (keeper *OVNNorthboundKeeper) Mark(ctx context.Context) {
	db := &keeper.DB
	itbls := []ovnutil.ITable{
		&db.LogicalSwitch,
		&db.LogicalSwitchPort,
		&db.LogicalRouter,
		&db.LogicalRouterPort,
		&db.DHCPOptions,
	}
	for _, itbl := range itbls {
		for _, irow := range itbl.Rows() {
			irow.OvnRemoveExternalIds(externalKeyOcVersion)
		}
	}
}

func (keeper *OVNNorthboundKeeper) Sweep(ctx context.Context) error {
	db := &keeper.DB
	// isRoot=false tables at the end
	itbls := []ovnutil.ITable{
		&db.LogicalSwitchPort,
		&db.LogicalRouterPort,
		&db.LogicalSwitch,
		&db.LogicalRouter,
		&db.DHCPOptions,
	}
	var irows []ovnutil.IRow
	for _, itbl := range itbls {
		for _, irow := range itbl.Rows() {
			_, ok := irow.OvnGetExternalIds(externalKeyOcVersion)
			if !ok {
				irows = append(irows, irow)
			}
		}
	}
	args := ovnutil.OvnNbctlArgsDestroy(irows)
	if len(args) > 0 {
		return keeper.cli.Must(ctx, "Sweep", args)
	}
	return nil
}
