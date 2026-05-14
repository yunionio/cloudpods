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

package hostinfo

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/iproute2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func (h *SHostInfo) findExternalInterfaces() []string {
	interfaces := make([]string, 0)
	for i := 0; i < len(h.Nics); i++ {
		if !h.Nics[i].IsHostLocal() && len(h.Nics[i].Ip) > 0 {
			interfaces = append(interfaces, h.Nics[i].Bridge)
		}
	}
	if len(options.HostOptions.ListenInterface) > 0 {
		interfaces = append(interfaces, options.HostOptions.ListenInterface)
	}
	return interfaces
}

func (h *SHostInfo) finalizeNetworkSetup(ctx context.Context) error {
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].SetupDhcpRelay(); err != nil {
			return errors.Wrapf(err, "SetupDhcpRelay %s", h.Nics[i])
		}
	}
	extInterfaces := h.findExternalInterfaces()
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].setupHostLocalNetworks(ctx, extInterfaces); err != nil {
			return errors.Wrapf(err, "SetupHostLocalNetworks %s", h.Nics[i])
		}
	}
	return nil
}

func (n *SNIC) String() string {
	return fmt.Sprintf("%s/%s/%s", n.Inter, n.Bridge, n.Ip)
}

func (n *SNIC) setupHostLocalNetworks(ctx context.Context, extInterfaces []string) error {
	nets, err := n.fetchHostLocalNetworks(ctx)
	if err != nil {
		return errors.Wrap(err, "fetchHostLocalNetworks")
	}
	hostLocalNics := make([]computeapis.NetworkDetails, 0)
	for i := range nets {
		net := nets[i]
		if len(net.GuestGateway) == 0 {
			continue
		}
		err := n.setupHostLocalNet(ctx, net, extInterfaces)
		if err != nil {
			return errors.Wrapf(err, "setupHostLocalNet of %s", jsonutils.Marshal(net))
		}
		hostLocalNics = append(hostLocalNics, nets[i])
	}
	// save hostLocalNics
	fn := options.HostOptions.HostLocalNetconfPath(n.Bridge)
	fileutils2.FilePutContents(fn, jsonutils.Marshal(hostLocalNics).PrettyString(), false)
	return nil
}

func (n *SNIC) setupHostLocalNet(ctx context.Context, netInfo computeapis.NetworkDetails, extInterfaces []string) error {
	// setup gateway ip
	if err := n.setupSlaveIp(ctx, netInfo.GuestGateway, netInfo.GuestIpMask, extInterfaces); err != nil {
		return errors.Wrapf(err, "setupSlaveIp %s %s", n, netInfo.GuestGateway)
	}
	return nil
}

func (n *SNIC) setupSlaveIp(ctx context.Context, gatewayIp string, maskLen byte, extInterfaces []string) error {
	bridgeIf := netutils2.NewNetInterface(n.Bridge)
	slaveAddrs := bridgeIf.GetSlaveAddresses()
	var curGatewayIp string
	var curMaskLen int
	isMaskUpdate := false
	for i := range slaveAddrs {
		addr := slaveAddrs[i]
		curGatewayIp = addr.Addr
		curMaskLen = addr.MaskLen
		if curGatewayIp == gatewayIp {
			if curMaskLen == int(maskLen) {
				// already configured, skip
				return nil
			} else {
				isMaskUpdate = true
				break
			}
		}
	}
	if err := n.BridgeDev.SetupSlaveAddresses([]netutils2.SNicAddress{
		{Addr: gatewayIp, MaskLen: int(maskLen)},
	}); err != nil {
		return errors.Wrap(err, "SetupSlaveAddresses")
	}
	if isMaskUpdate {
		brName := n.BridgeDev.Bridge()
		logPrefix := fmt.Sprintf("%s slave address %s mask is update from %d to %d", brName, gatewayIp, curMaskLen, maskLen)
		addr := fmt.Sprintf("%s/%d", gatewayIp, curMaskLen)
		log.Infof("%s: delete addr %s", logPrefix, addr)
		if err := iproute2.NewAddress(n.BridgeDev.Bridge(), addr).Del().Err(); err != nil {
			log.Warningf("%s: new addr %s: %v", logPrefix, addr, err)
		}
	}
	return nil
}

func (n *SNIC) fetchHostLocalNetworks(ctx context.Context) ([]computeapis.NetworkDetails, error) {
	return fetchHostLocalNetworksByWireId(ctx, n.WireId)
}

func fetchHostLocalNetworksByWireId(ctx context.Context, wireId string) ([]computeapis.NetworkDetails, error) {
	s := hostutils.GetComputeSession(ctx)
	params := computeoptions.NetworkListOptions{}
	params.ServerType = "hostlocal"
	limit := 50
	params.Limit = &limit
	params.Scope = "system"
	total := -1
	nets := make([]computeapis.NetworkDetails, 0)
	for total < 0 || len(nets) < total {
		offset := len(nets)
		params.Offset = &offset
		results, err := computemodules.Networks.ListInContext(s, jsonutils.Marshal(params), &computemodules.Wires, wireId)
		if err != nil {
			return nil, errors.Wrap(err, "Networks.List")
		}
		total = results.Total
		for i := range results.Data {
			netDetails := computeapis.NetworkDetails{}
			err := results.Data[i].Unmarshal(&netDetails)
			if err != nil {
				return nil, errors.Wrap(err, "Unmarshal")
			}
			nets = append(nets, netDetails)
		}
	}
	return nets, nil
}
