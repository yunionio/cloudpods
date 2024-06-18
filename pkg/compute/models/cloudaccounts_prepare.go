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

package models

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type sNetworkInfo struct {
	esxi.SNetworkInfo
	prefix string
}

func (scm *SCloudaccountManager) hostVMIPsPrepareNets(ctx context.Context, client *esxi.SESXiClient,
	input api.CloudaccountPerformPrepareNetsInput) ([]sNetworkInfo, error) {
	caName := input.Name
	wireLevel := input.WireLevelForVmware
	ret := make([]sNetworkInfo, 0)
	if len(wireLevel) == 0 {
		wireLevel = api.CLOUD_ACCOUNT_WIRE_LEVEL_VCENTER
	}
	switch wireLevel {
	case api.CLOUD_ACCOUNT_WIRE_LEVEL_VCENTER:
		nInfo, err := client.HostVmIPs(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch ips of hosts and vms")
		}
		ret = append(ret, sNetworkInfo{
			SNetworkInfo: nInfo,
			prefix:       caName,
		})
	case api.CLOUD_ACCOUNT_WIRE_LEVEL_DATACENTER:
		dcs, err := client.GetDatacenters()
		if err != nil {
			return ret, errors.Wrap(err, "GetDatacenters")
		}
		for _, dc := range dcs {
			nInfo, err := client.HostVmIPsInDc(ctx, dc)
			if err != nil {
				return ret, errors.Wrapf(err, "unable to fetch ips of hosts and vms for dc %q", dc.GetName())
			}
			ret = append(ret, sNetworkInfo{
				SNetworkInfo: nInfo,
				prefix:       fmt.Sprintf("%s/%s", caName, dc.GetName()),
			})
		}
	case api.CLOUD_ACCOUNT_WIRE_LEVEL_CLUSTER:
		dcs, err := client.GetDatacenters()
		if err != nil {
			return ret, errors.Wrap(err, "GetDatacenters")
		}
		for _, dc := range dcs {
			clusters, err := dc.ListClusters()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to ListCluster for dc %q", dc.GetName())
			}
			for _, cluster := range clusters {
				nInfo, err := client.HostVmIPsInCluster(ctx, cluster)
				if err != nil {
					return ret, errors.Wrapf(err, "unable to fetch ips of hosts and vms for dc %q cluster %q", dc.GetName(), cluster.GetName())
				}
				ret = append(ret, sNetworkInfo{
					SNetworkInfo: nInfo,
					prefix:       fmt.Sprintf("%s/%s/%s", caName, dc.GetName(), cluster.GetName()),
				})
			}
		}
	default:
		return nil, httperrors.NewInputParameterError("valid wire_level_for_vmware, accept vcenter, datacenter, cluster")
	}
	return ret, nil
}

// Performpreparenets searches for suitable network facilities for physical and virtual machines under the cloud account or provides configuration recommendations for network facilities before importing a cloud account.
func (scm *SCloudaccountManager) PerformPrepareNets(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudaccountPerformPrepareNetsInput) (api.CloudaccountPerformPrepareNetsOutput, error) {
	var (
		err    error
		output api.CloudaccountPerformPrepareNetsOutput
	)
	if input.Provider != api.CLOUD_PROVIDER_VMWARE {
		return output, httperrors.NewNotSupportedError("not support for cloudaccount with provider '%s'", input.Provider)
	}
	// validate first
	ownerId, err := scm.FetchOwnerId(ctx, jsonutils.Marshal(input))
	if err != nil {
		return output, errors.Wrap(err, "FetchOwnerId in PerformPrepareNets")
	}
	if ownerId == nil {
		ownerId = userCred
	}
	// validate domain
	if len(input.ProjectDomainId) > 0 {
		_, input.DomainizedResourceInput, err = db.ValidateDomainizedResourceInput(ctx, input.DomainizedResourceInput)
		if err != nil {
			return output, err
		}
	}

	// Determine the zoneids according to esxiagent. If there is no esxiagent, zone0 is used by default. And the wires are filtered according to the specified domain and zoneids
	// make sure zone
	zoneids, err := scm.FetchEsxiZoneIds()
	if err != nil {
		return output, errors.Wrap(err, "unable to FetchEsxiZoneIds")
	}
	if len(zoneids) == 0 {
		id, err := scm.defaultZoneId(ctx, userCred)
		if err != nil {
			return output, errors.Wrap(err, "unable to fetch defaultZoneId")
		}
		zoneids = append(zoneids, id)
	}

	// Find the appropriate wire from above wires according to the Host's IP. The logic for finding is that the networks in the wire can contain the Host as much as possible. If no suitable one is found, a new wire is used.
	// fetch all Host
	factory, err := cloudprovider.GetProviderFactory(input.Provider)
	if err != nil {
		return output, errors.Wrap(err, "cloudprovider.GetProviderFactory")
	}
	input.SCloudaccount, err = factory.ValidateCreateCloudaccountData(ctx, input.SCloudaccountCredential)
	if err != nil {
		return output, errors.Wrap(err, "providerDriver.ValidateCreateCloudaccountData")
	}
	var proxyFunc httputils.TransportProxyFunc
	{
		if input.ProxySettingId == "" {
			input.ProxySettingId = proxyapi.ProxySettingId_DIRECT
		}
		var proxySetting *proxy.SProxySetting
		proxySetting, input.ProxySettingResourceInput, err = proxy.ValidateProxySettingResourceInput(ctx, userCred, input.ProxySettingResourceInput)
		if err != nil {
			return output, errors.Wrap(err, "ValidateProxySettingResourceInput")
		}
		proxyFunc = proxySetting.HttpTransportProxyFunc()
	}
	provider, err := factory.GetProvider(cloudprovider.ProviderConfig{
		Vendor:    input.Provider,
		URL:       input.AccessUrl,
		Account:   input.Account,
		Secret:    input.Secret,
		ProxyFunc: proxyFunc,
		Name:      input.Name,
		RegionId:  input.RegionId,

		AliyunResourceGroupIds: options.Options.AliyunResourceGroups,

		Options: input.Options,
	})
	if err != nil {
		return output, errors.Wrap(err, "factory.GetProvider")
	}
	iregion, err := provider.GetOnPremiseIRegion()
	if err != nil {
		return output, errors.Wrap(err, "provider.GetOnPremiseIRegion")
	}
	// hack
	client := iregion.(*esxi.SESXiClient)
	return scm.prepareNets(ctx, userCred, client, zoneids, input)
}

func (cam *SCloudaccountManager) prepareNets(ctx context.Context, userCred mcclient.TokenCredential,
	client *esxi.SESXiClient, zoneids []string, input api.CloudaccountPerformPrepareNetsInput) (api.CloudaccountPerformPrepareNetsOutput, error) {
	var output api.CloudaccountPerformPrepareNetsOutput
	obj, err := VpcManager.FetchById(api.DEFAULT_VPC_ID)
	if err != nil {
		return output, errors.Wrapf(err, "can not fetch vpc %q", api.DEFAULT_VPC_ID)
	}
	allNetworks, err := obj.(*SVpc).GetNetworks()
	if err != nil {
		return output, errors.Wrapf(err, "can not get networks of vpc %q", api.DEFAULT_VPC_ID)
	}
	existedIpPool := newIPPool(len(allNetworks))
	for i := range allNetworks {
		startIp, _ := netutils.NewIPV4Addr(allNetworks[i].GuestIpStart)
		endIp, _ := netutils.NewIPV4Addr(allNetworks[i].GuestIpEnd)
		existedIpPool.Insert(startIp, sSimpleNet{
			Diff: endIp - startIp,
			Vlan: int32(allNetworks[i].VlanId),
			Id:   allNetworks[i].Id,
		})
	}

	if !input.Dvs {
		// fetch all wire candidate
		wires, err := cam.fetchWires(ctx, userCred, input.ProjectDomainId, zoneids)
		if err != nil {
			return output, errors.Wrap(err, "unable to fetch wires")
		}
		// fetch networks
		networks := make([][]SNetwork, len(wires))
		for i := range networks {
			nets, err := wires[i].getNetworks(ctx, userCred, userCred, rbacscope.ScopeSystem)
			if err != nil {
				return output, errors.Wrap(err, "wire.getNetwork")
			}
			networks[i] = nets
		}
		nInfos, err := cam.hostVMIPsPrepareNets(ctx, client, input)
		if err != nil {
			return output, err
		}

		return cam.parseAndSuggestSingleWire(sParseAndSuggest{
			NInfos:        nInfos,
			AccountName:   input.Name,
			ZoneIds:       zoneids,
			Wires:         wires,
			Networks:      networks,
			ExistedIpPool: existedIpPool,
		}), nil
	}

	nInfo, err := client.HostVmIPsPro(ctx)
	if err != nil {
		return output, err
	}

	for name, ips := range nInfo.HostIps {
		ipNets := make([]api.CAIPNet, len(ips))
		for i, ip := range ips {
			var suitNetwork string
			if p, ok := existedIpPool.Get(ip); ok {
				suitNetwork = p.Id
			}
			ipNets[i] = api.CAIPNet{
				SuitableNetwork: suitNetwork,
				IP:              ip.String(),
			}
		}
		output.Hosts = append(output.Hosts, api.CAGuestNet{
			Name:   name,
			IPNets: ipNets,
		})
	}
	// output.Guests = cam.parseSimpleVms(nInfo.VMs, existedIpPool)

	vsList := nInfo.VsMap.List()
	for i := range vsList {
		vs := vsList[i]
		hosts := make([]api.SimpleHost, len(vs.Hosts))
		for i := range hosts {
			hosts[i] = api.SimpleHost{
				Id:   vs.Hosts[i].Id,
				Name: vs.Hosts[i].Name,
			}
		}
		wire := api.CAPWire{
			Name:        vs.Name,
			Id:          vs.Id,
			Distributed: vs.Distributed,
			Hosts:       hosts,
		}
		// preparenets for host
		for hName, ips := range vs.HostIps {
			if len(ips) == 0 {
				output.Hosts = append(output.Hosts, api.CAGuestNet{
					Name: hName,
				})
			}
			for j := 0; j < len(ips); j++ {
				baseNet := ips[j].NetAddr(24)
				ipWithSameNet := []netutils.IPV4Addr{ips[j]}
				for k := j + 1; k < len(ips); k++ {
					net := ips[k].NetAddr(24)
					if net != baseNet {
						break
					}
					ipWithSameNet = append(ipWithSameNet, ips[k])
				}
				ipLimitLow, ipLimitUp := ips[0], ips[len(ips)-1]
				simNetConfs := cam.expandIPRange(ips, ipLimitLow, ipLimitUp, func(proc esxi.SIPProc) bool {
					return proc.IsHost
				}, nInfo.IPPool, existedIpPool)
				for k := range simNetConfs {
					wire.HostNetworks = append(wire.HostNetworks, api.CANetConf{
						Name:            fmt.Sprintf("host-network-%d", len(wire.HostNetworks)+1),
						CASimpleNetConf: simNetConfs[k],
					})
				}
			}
		}
		/*for vlan, ips := range vs.Vlans {
			simNetConfs := cam.expandIPRange(ips, 0, 0, func(proc esxi.SIPProc) bool {
				return proc.VlanId == vlan
			}, nInfo.IPPool, existedIpPool)
			for j := range simNetConfs {
				simNetConfs[j].VlanID = vlan
				wire.GuestNetworks = append(wire.GuestNetworks, api.CANetConf{
					Name:            fmt.Sprintf("guest-network-%d", len(wire.GuestNetworks)+1),
					CASimpleNetConf: simNetConfs[j],
				})
			}
		}*/
		output.Wires = append(output.Wires, wire)
	}

	return output, nil
}

func (cam *SCloudaccountManager) parseSimpleVms(vms []esxi.SSimpleVM, existedIpPool *sIPPool) []api.CAGuestNet {
	guests := make([]api.CAGuestNet, len(vms))
	for i := range guests {
		guests[i].Name = vms[i].Name
		for _, ipVlan := range vms[i].IPVlans {
			var suitableNetwork string
			p, ok := existedIpPool.Get(ipVlan.IP)
			if ok {
				suitableNetwork = p.Id
			}
			guests[i].IPNets = append(guests[i].IPNets, api.CAIPNet{
				IP:              ipVlan.IP.String(),
				VlanID:          ipVlan.VlanId,
				SuitableNetwork: suitableNetwork,
			})
		}
	}
	return guests
}

func (cam *SCloudaccountManager) expandIPRange(ips []netutils.IPV4Addr, limitLow, limitUp netutils.IPV4Addr,
	expand func(esxi.SIPProc) bool, ipPool esxi.SIPPool, existedIpPool *sIPPool) []api.CASimpleNetConf {
	ret := make([]api.CASimpleNetConf, 0)
	for i := 0; i < len(ips); i++ {
		ip := ips[i]
		if _, ok := existedIpPool.Get(ip); ok {
			continue
		}
		net := ip.NetAddr(24)
		netLimitLow := net + 1
		netLimitUp := net + 254
		if limitLow != 0 && limitLow > netLimitLow {
			netLimitLow = limitLow
		}
		if limitUp != 0 && limitUp < netLimitUp {
			netLimitUp = limitUp
		}
		// find startip
		startIp := ip - 1
		for ; startIp >= netLimitLow; startIp-- {
			if _, ok := existedIpPool.Get(startIp); ok {
				break
			}
			if _, ok := ipPool.Get(startIp); ok {
				break
			}
		}
		endIp := ip + 1
		for ; endIp <= netLimitUp; endIp++ {
			if _, ok := existedIpPool.Get(endIp); ok {
				break
			}
			if proc, ok := ipPool.Get(endIp); ok {
				if expand(proc) {
					i++
					continue
				}
				break
			}
		}
		ret = append(ret, api.CASimpleNetConf{
			GuestIpStart: (startIp + 1).String(),
			GuestIpEnd:   (endIp - 1).String(),
			GuestIpMask:  24,
			GuestGateway: (net + netutils.IPV4Addr(options.Options.DefaultNetworkGatewayAddressEsxi)).String(),
		})
		// Avoid assigning already assigned ip subnet
		existedIpPool.Insert(startIp+1, sSimpleNet{
			Diff: endIp - startIp - 2,
		})
	}
	return ret
}

type sParseAndSuggest struct {
	NInfos        []sNetworkInfo
	AccountName   string
	ZoneIds       []string
	Wires         []SWire
	Networks      [][]SNetwork
	ExistedIpPool *sIPPool
}

type sIPPool struct {
	netranges    []netutils.IPV4Addr
	simpleNetMap map[netutils.IPV4Addr]sSimpleNet
}

func newIPPool(length ...int) *sIPPool {
	initLen := 0
	if len(length) > 0 {
		initLen = length[0]
	}
	return &sIPPool{
		netranges:    make([]netutils.IPV4Addr, 0, initLen),
		simpleNetMap: make(map[netutils.IPV4Addr]sSimpleNet, initLen),
	}
}

type sSimpleNet struct {
	Diff netutils.IPV4Addr
	Id   string
	Vlan int32
}

func (pl *sIPPool) Insert(startIp netutils.IPV4Addr, sNet sSimpleNet) {
	// TODO:check
	index := pl.getIndex(startIp)
	pl.netranges = append(pl.netranges, 0)
	pl.netranges = append(pl.netranges[:index+1], pl.netranges[index:len(pl.netranges)-1]...)
	pl.netranges[index] = startIp
	pl.simpleNetMap[startIp] = sNet
}

func (pl *sIPPool) getIndex(ip netutils.IPV4Addr) int {
	index := sort.Search(len(pl.netranges), func(n int) bool {
		return pl.netranges[n] >= ip
	})
	return index
}

func (pl *sIPPool) Get(ip netutils.IPV4Addr) (sSimpleNet, bool) {
	index := pl.getIndex(ip)
	if index >= len(pl.netranges) || index < 0 {
		return sSimpleNet{}, false
	}
	if pl.netranges[index] == ip {
		return pl.simpleNetMap[ip], true
	}
	if index == 0 {
		return sSimpleNet{}, false
	}
	startIp := pl.netranges[index-1]
	simpleNet := pl.simpleNetMap[startIp]
	if ip-startIp <= simpleNet.Diff {
		return simpleNet, true
	}
	return sSimpleNet{}, false
}

func (scm *SCloudaccountManager) parseAndSuggestSingleWire(params sParseAndSuggest) api.CloudaccountPerformPrepareNetsOutput {
	var (
		output   api.CloudaccountPerformPrepareNetsOutput
		nInfos   = params.NInfos
		wires    = params.Wires
		networks = params.Networks
	)
	output.CAWireNets = make([]api.CAWireNet, 0, len(nInfos))
	for _, ni := range nInfos {
		var (
			wireNet api.CAWireNet
			hostIps = ni.HostIps
		)

		// key of ipHosts is host's ip
		ipHosts := make(map[netutils.IPV4Addr]string, len(hostIps))
		for name, ip := range hostIps {
			ipHosts[ip] = name
		}
		// Find suitable wire and the network containing the Host IP in suitable wire.
		var (
			tmpSocre         int
			maxScore         = len(ipHosts)
			suitableWire     *SWire
			suitableNetworks map[netutils.IPV4Addr]*SNetwork
		)
		for i, nets := range networks {
			score := 0
			tmpSNs := make(map[netutils.IPV4Addr]*SNetwork)
			ipRanges := make([]*netutils.IPV4AddrRange, len(nets))
			for i2 := range ipRanges {
				ipRanges[i2] = nets[i2].GetIPRange()
			}
			for ip := range ipHosts {
				for i := range ipRanges {
					if !ipRanges[i].Contains(ip) {
						continue
					}
					tmpSNs[ip] = &nets[i]
					score += 1
					break
				}
			}
			if score > tmpSocre {
				tmpSocre = score
				suitableWire = &wires[i]
				suitableNetworks = tmpSNs
			}
			if tmpSocre == maxScore {
				break
			}
		}
		if suitableWire != nil {
			wireNet.SuitableWire = suitableWire.GetId()
		} else {
			wireNet.SuggestedWire = api.CAWireConf{
				ZoneIds:     params.ZoneIds,
				Name:        ni.prefix + "-wire",
				Description: fmt.Sprintf("Auto created Wire for VMware account %q", params.AccountName),
			}
		}

		// Give the suggested network configuration for the Host IP that does not have a corresponding suitable network.
		noNetHostIP := make([]netutils.IPV4Addr, 0, len(ipHosts))
		for ip, name := range ipHosts {
			rnet := api.CAHostNet{
				Name: name,
				IP:   ip.String(),
			}
			if net, ok := suitableNetworks[ip]; ok {
				rnet.SuitableNetwork = net.GetId()
			} else {
				noNetHostIP = append(noNetHostIP, ip)
			}
			wireNet.Hosts = append(wireNet.Hosts, rnet)
		}

		if len(noNetHostIP) > 0 {
			sConfs := scm.suggestHostNetworks(noNetHostIP)
			confs := make([]api.CANetConf, len(sConfs))
			for i := range confs {
				confs[i].CASimpleNetConf = sConfs[i]
				confs[i].Name = fmt.Sprintf("%s-host-network-%d", ni.prefix, i+1)
			}
			wireNet.HostSuggestedNetworks = confs
		}

		for i := range wireNet.HostSuggestedNetworks {
			ipStart, _ := netutils.NewIPV4Addr(wireNet.HostSuggestedNetworks[i].GuestIpStart)
			ipEnd, _ := netutils.NewIPV4Addr(wireNet.HostSuggestedNetworks[i].GuestIpEnd)
			params.ExistedIpPool.Insert(ipStart, sSimpleNet{
				Diff: ipEnd - ipStart,
			})
		}

		/* wireNet.Guests = scm.parseSimpleVms(ni.VMs, params.ExistedIpPool)

		for vlan, ips := range ni.VlanIps {
			simNetConfs := scm.expandIPRange(ips, 0, 0, func(proc esxi.SIPProc) bool {
				return proc.VlanId == vlan
			}, ni.IPPool, params.ExistedIpPool)
			for i := range simNetConfs {
				simNetConfs[i].VlanID = vlan
				wireNet.GuestSuggestedNetworks = append(wireNet.GuestSuggestedNetworks, api.CANetConf{
					Name:            fmt.Sprintf("host-network-%d", len(wireNet.GuestSuggestedNetworks)+1),
					CASimpleNetConf: simNetConfs[i],
				})
			}
		}*/
		output.CAWireNets = append(output.CAWireNets, wireNet)
	}
	return output
}

func (manager *SCloudaccountManager) fetchWires(ctx context.Context, userCred mcclient.TokenCredential, domainId string, zoneIds []string) ([]SWire, error) {
	q := WireManager.Query().In("zone_id", zoneIds)
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{}
		ownerId.DomainId = domainId
		q = WireManager.FilterByOwner(ctx, q, WireManager, userCred, ownerId, rbacscope.ScopeDomain)
	} else {
		q = WireManager.FilterByOwner(ctx, q, WireManager, userCred, userCred, rbacscope.ScopeDomain)
	}
	wires := make([]SWire, 0, 1)
	err := db.FetchModelObjects(WireManager, q, &wires)
	return wires, err
}

func (manager *SCloudaccountManager) defaultZoneId(ctx context.Context, userCred mcclient.TokenCredential) (string, error) {
	zone, err := ZoneManager.FetchByName(ctx, userCred, "zone0")
	if err != nil {
		return "", err
	}
	return zone.GetId(), nil
}

func (manager *SCloudaccountManager) FetchEsxiZoneIds() ([]string, error) {
	q := BaremetalagentManager.Query().Equals("agent_type", "esxiagent").Asc("created_at")
	agents := make([]SBaremetalagent, 0, 1)
	err := db.FetchModelObjects(BaremetalagentManager, q, &agents)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(agents))
	for i := range agents {
		if agents[i].Status == api.BAREMETAL_AGENT_ENABLED {
			ids = append(ids, agents[i].ZoneId)
		}
	}
	for i := range agents {
		if agents[i].Status != api.BAREMETAL_AGENT_ENABLED {
			ids = append(ids, agents[i].ZoneId)
		}
	}
	return ids, nil
}

// The suggestHostNetworks give the suggest config of network contains the ip in 'ips'.
// The suggested network mask is 24 and the gateway is x.x.x.1.
// The suggests network is the smallest network segment that meets the above conditions.
func (manager *SCloudaccountManager) suggestHostNetworks(ips []netutils.IPV4Addr) []api.CASimpleNetConf {
	if len(ips) == 0 {
		return nil
	}
	sort.Slice(ips, func(i, j int) bool {
		return ips[i] < ips[j]
	})
	var (
		mask        int8 = 24
		lastnetAddr netutils.IPV4Addr
		consequent  []netutils.IPV4Addr
		ret         []api.CASimpleNetConf
	)
	lastnetAddr = ips[0].NetAddr(mask)
	consequent = []netutils.IPV4Addr{ips[0]}
	for i := 1; i < len(ips); i++ {
		ip := ips[i]
		netAddr := ip.NetAddr(mask)
		if netAddr == lastnetAddr && consequent[len(consequent)-1]+1 == ip {
			consequent = append(consequent, ip)
			continue
		}

		if netAddr != lastnetAddr {
			lastnetAddr = netAddr
		}

		gatewayIP := consequent[0].NetAddr(mask) + netutils.IPV4Addr(options.Options.DefaultNetworkGatewayAddressEsxi)
		ret = append(ret, api.CASimpleNetConf{
			GuestIpStart: consequent[0].String(),
			GuestIpEnd:   consequent[len(consequent)-1].String(),
			GuestIpMask:  mask,
			GuestGateway: gatewayIP.String(),
		})
		consequent = []netutils.IPV4Addr{ip}
	}
	gatewayIp := consequent[0].NetAddr(mask) + netutils.IPV4Addr(options.Options.DefaultNetworkGatewayAddressEsxi)
	ret = append(ret, api.CASimpleNetConf{
		GuestIpStart: consequent[0].String(),
		GuestIpEnd:   consequent[len(consequent)-1].String(),
		GuestIpMask:  mask,
		GuestGateway: gatewayIp.String(),
	})
	return ret
}
