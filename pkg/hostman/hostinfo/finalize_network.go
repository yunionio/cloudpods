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

func (h *SHostInfo) finalizeNetworkSetup(ctx context.Context) error {
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].SetupDhcpRelay(); err != nil {
			return errors.Wrapf(err, "SetupDhcpRelay %s", h.Nics[i])
		}
	}
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].setupHostLocalNetworks(ctx); err != nil {
			return errors.Wrapf(err, "SetupHostLocalNetworks %s", h.Nics[i])
		}
	}
	return nil
}

func (n *SNIC) String() string {
	return fmt.Sprintf("%s/%s/%s", n.Inter, n.Bridge, n.Ip)
}

func (n *SNIC) setupHostLocalNetworks(ctx context.Context) error {
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
		err := n.setupHostLocalNet(ctx, net)
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

func (n *SNIC) setupHostLocalNet(ctx context.Context, netInfo computeapis.NetworkDetails) error {
	// setup gateway ip
	if err := n.setupSlaveIp(ctx, netInfo.GuestGateway, netInfo.GuestIpMask); err != nil {
		return errors.Wrapf(err, "setupSlaveIp %s %s", n, netInfo.GuestGateway)
	}
	return nil
}

func (n *SNIC) setupSlaveIp(ctx context.Context, gatewayIp string, maskLen byte) error {
	bridgeIf := netutils2.NewNetInterface(n.Bridge)
	slaveAddrs := bridgeIf.GetSlaveAddresses()
	var curGatewayIp string
	var curMaskLen string
	isMaskUpdate := false
	for i := range slaveAddrs {
		addr := slaveAddrs[i]
		curGatewayIp = addr[0]
		curMaskLen = addr[1]
		if curGatewayIp == gatewayIp {
			if curMaskLen == fmt.Sprintf("%d", maskLen) {
				// already configured, skip
				return nil
			} else {
				isMaskUpdate = true
				break
			}
		}
	}
	if err := n.BridgeDev.SetupSlaveAddresses([][]string{
		{gatewayIp, fmt.Sprintf("%d", maskLen)},
	}); err != nil {
		return errors.Wrap(err, "SetupSlaveAddresses")
	}
	if isMaskUpdate {
		brName := n.BridgeDev.Bridge()
		logPrefix := fmt.Sprintf("%s slave address %s mask is update from %s to %d", brName, gatewayIp, curMaskLen, maskLen)
		addr := fmt.Sprintf("%s/%s", gatewayIp, curMaskLen)
		log.Infof("%s: delete addr %s", logPrefix, addr)
		if err := iproute2.NewAddress(n.BridgeDev.Bridge(), addr).Del().Err(); err != nil {
			log.Warningf("%s: new addr %s: %v", logPrefix, addr, err)
		}
	}
	return nil
}

func (n *SNIC) fetchHostLocalNetworks(ctx context.Context) ([]computeapis.NetworkDetails, error) {
	s := hostutils.GetComputeSession(ctx)
	params := computeoptions.NetworkListOptions{}
	params.ServerType = "hostlocal"
	limit := 50
	params.Limit = &limit
	params.Wire = n.WireId
	params.Scope = "system"
	total := -1
	nets := make([]computeapis.NetworkDetails, 0)
	for total < 0 || len(nets) < total {
		offset := len(nets)
		params.Offset = &offset
		results, err := computemodules.Networks.List(s, jsonutils.Marshal(params))
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
