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

package tasks

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type CloudAccountSyncVMwareNetworkTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountSyncVMwareNetworkTask{})
}

func (self *CloudAccountSyncVMwareNetworkTask) taskFailed(ctx context.Context, cloudaccount *models.SCloudaccount, desc string, err error) {
	log.Errorf("err: %v", err)
	d := jsonutils.NewDict()
	d.Set("description", jsonutils.NewString(desc))
	if err != nil {
		d.Set("error", jsonutils.NewString(err.Error()))
	}
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_NETWORK_FAILED, d, self.UserCred)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUDACCOUNT_SYNC_NETWORK, d, self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}

func (self *CloudAccountSyncVMwareNetworkTask) taskSuccess(ctx context.Context, cloudaccount *models.SCloudaccount, desc string) {
	d := jsonutils.NewString(desc)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_NETWORK, d, self.UserCred)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUDACCOUNT_SYNC_NETWORK, d, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *CloudAccountSyncVMwareNetworkTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	// make sure zone
	zoneId, err := self.zoneId()
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to make sure zoneid", err)
		return
	}
	self.Params.Set("zoneId", jsonutils.NewString(zoneId))
	cProvider, err := cloudaccount.GetProvider()
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to GetProvider", err)
		return
	}
	// fetch esxiclient
	iregion, err := cProvider.GetOnPremiseIRegion()
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to GetOnPremiseIRegion", err)
		return
	}
	esxiClient := iregion.(*esxi.SESXiClient)
	// check network
	nInfo, err := esxiClient.HostVmIPsPro(ctx)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to fetch HostVmIPs", err)
		return
	}
	log.Infof("nInfo: %s", jsonutils.Marshal(nInfo).String())
	// make sure wire and network existed
	wires, err := self.fetchWires(cloudaccount, zoneId)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to fetch wires", err)
		return
	}
	existedIPPool, err := self.ipPool(cloudaccount, wires)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "uanble to structure existedIPPool", err)
		return
	}
	excludedIPPool, err := self.ipPool(nil, nil)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to structure excludedIPPool", err)
		return
	}
	vss := nInfo.VsMap.List()
	for i := range vss {
		for hostName, ips := range vss[i].HostIps {
			for _, ip := range ips {
				if _, ok := existedIPPool.Get(ip); ok {
					continue
				}
				net, ok := excludedIPPool.Get(ip)
				if !ok {
					continue
				}
				desc := fmt.Sprintf("networks %s shouldn't have ip %s belong to host %s, because the wire %s where in this network don't belong to zone %s or can't be used by cloudaccount %s owner", net.Id, ip, hostName, net.WireId, zoneId, cloudaccount.Id)
				self.taskFailed(ctx, cloudaccount, desc, nil)
				return
			}
		}
	}
	capWires := make([]CAPWire, 0)
	vsList := nInfo.VsMap.List()
	log.Infof("vsList: %s", jsonutils.Marshal(vsList))
	desc := fmt.Sprintf("Auto create for cloudaccount %q", cloudaccount.GetName())
	for i := range vsList {
		vs := vsList[i]
		wireName := fmt.Sprintf("%s/%s", cloudaccount.Name, vs.Name)
		capWire := CAPWire{
			Name:        wireName,
			VsId:        vs.Id,
			Distributed: vs.Distributed,
			Hosts:       vs.Hosts,
		}
		wireScore := make(map[string]int)
		for _, ips := range vs.HostIps {
			for j := 0; j < len(ips); j++ {
				baseNet := ips[j].NetAddr(ipMaskLen)
				ipWithSameNet := []netutils.IPV4Addr{ips[j]}
				for k := j + 1; k < len(ips); k++ {
					net := ips[k].NetAddr(24)
					if net != baseNet {
						break
					}
					ipWithSameNet = append(ipWithSameNet, ips[k])
				}
				ipLimitLow, ipLimitUp := ipWithSameNet[0], ipWithSameNet[len(ips)-1]
				simNetConfs := self.expandIPRnage(ipWithSameNet, ipLimitLow, ipLimitUp,
					func(proc esxi.SIPProc) bool {
						return proc.IsHost
					},
					nInfo.IPPool, excludedIPPool,
					func(net sSimpleNet) {
						wireScore[net.WireId] += 2
					},
				)
				for k := range simNetConfs {
					capWire.HostNetworks = append(capWire.HostNetworks, CANetConf{
						Name:            fmt.Sprintf("%s-host-network-%d", wireName, len(capWire.HostNetworks)+1),
						Description:     desc,
						CASimpleNetConf: simNetConfs[k],
					})
				}
			}
		}
		for vlan, ips := range vs.Vlans {
			simNetConfs := self.expandIPRnage(ips, 0, 0, func(proc esxi.SIPProc) bool {
				return proc.VlanId == vlan && proc.VSId == vs.Id && !proc.IsHost
			}, nInfo.IPPool, excludedIPPool, func(net sSimpleNet) {
				wireScore[net.WireId] += 1
			})
			for j := range simNetConfs {
				simNetConfs[j].VlanID = vlan
				capWire.GuestNetworks = append(capWire.GuestNetworks, CANetConf{
					Name:            fmt.Sprintf("%s-guest-network-%d", wireName, len(capWire.GuestNetworks)+1),
					Description:     desc,
					CASimpleNetConf: simNetConfs[j],
				})
			}
		}
		// make sure wire
		suitableWire := ""
		maxScore := 1
		for wireId, score := range wireScore {
			if score > maxScore {
				suitableWire = wireId
				maxScore = score
			}
		}
		capWire.WireId = suitableWire
		if len(capWire.WireId) == 0 {
			capWire.Description = desc
		}
		capWires = append(capWires, capWire)
	}
	log.Infof("capWires: %v", capWires)
	host2Wire, err := self.createNetworks(ctx, cloudaccount, zoneId, capWires)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "uable to creat networks", err)
		return
	}
	err = cloudaccount.SetHost2Wire(ctx, self.UserCred, host2Wire)
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to store host2wire map", err)
		return
	}
	self.SetStage("OnSyncCloudProviderInfoComplete", nil)
	err = cloudaccount.StartSyncCloudProviderInfoTask(ctx, self.UserCred, nil, self.GetTaskId())
	if err != nil {
		self.taskFailed(ctx, cloudaccount, "unable to StartSyncCloudProviderInfoTask", err)
	}
}

func (self *CloudAccountSyncVMwareNetworkTask) OnSyncCloudProviderInfoComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	self.taskSuccess(ctx, cloudaccount, "")
}

func (self *CloudAccountSyncVMwareNetworkTask) OnSyncCloudProviderInfoCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_NETWORK_FAILED, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUDACCOUNT_SYNC_NETWORK, err, self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}

func (self *CloudAccountSyncVMwareNetworkTask) createNetworks(ctx context.Context, cloudaccount *models.SCloudaccount, zoneId string, capWires []CAPWire) (map[string][]models.SVs2Wire, error) {
	var err error
	ret := make(map[string][]models.SVs2Wire)
	for i := range capWires {
		var wireId = capWires[i].WireId
		if len(wireId) == 0 {
			wireId, err = self.createWire(ctx, cloudaccount, api.DEFAULT_VPC_ID, zoneId, capWires[i].Name, capWires[i].Description)
			if err != nil {
				return nil, errors.Wrapf(err, "can't create wire %s", capWires[i].Name)
			}
			capWires[i].WireId = wireId
		}
		for _, net := range capWires[i].HostNetworks {
			err := self.createNetwork(ctx, cloudaccount, wireId, api.NETWORK_TYPE_BAREMETAL, net)
			if err != nil {
				return nil, errors.Wrapf(err, "can't create network %v", net)
			}
		}
		for _, net := range capWires[i].GuestNetworks {
			err := self.createNetwork(ctx, cloudaccount, wireId, api.NETWORK_TYPE_GUEST, net)
			if err != nil {
				return nil, errors.Wrapf(err, "can't create network %v", net)
			}
		}
		for _, host := range capWires[i].Hosts {
			ret[host.Id] = append(ret[host.Id], models.SVs2Wire{
				VsId:        capWires[i].VsId,
				WireId:      capWires[i].WireId,
				Distributed: capWires[i].Distributed,
				Mac:         host.Mac,
			})
		}
	}
	return ret, nil
}

// NETWORK_TYPE_GUEST     = "guest"
// NETWORK_TYPE_BAREMETAL = "baremetal"
func (self *CloudAccountSyncVMwareNetworkTask) createNetwork(ctx context.Context, cloudaccount *models.SCloudaccount, wireId, networkType string, net CANetConf) error {
	network := &models.SNetwork{}
	network.Name = net.Name
	if hint, err := models.NetworkManager.NewIfnameHint(net.Name); err != nil {
		log.Errorf("can't NewIfnameHint form hint %s", net.Name)
	} else {
		network.IfnameHint = hint
	}
	network.GuestIpStart = net.IpStart
	network.GuestIpEnd = net.IpEnd
	network.GuestIpMask = net.IpMask
	network.GuestGateway = net.Gateway
	network.VlanId = int(net.VlanID)
	network.WireId = wireId
	network.ServerType = networkType
	network.IsPublic = true
	network.Status = api.NETWORK_STATUS_AVAILABLE
	network.PublicScope = string(rbacutils.ScopeDomain)
	network.ProjectId = cloudaccount.ProjectId
	network.DomainId = cloudaccount.DomainId
	network.Description = net.Description

	network.SetModelManager(models.NetworkManager, network)
	// TODO: Prevent IP conflict
	log.Infof("create network %s succussfully", network.Id)
	err := models.NetworkManager.TableSpec().Insert(ctx, network)
	return err
}

func (self *CloudAccountSyncVMwareNetworkTask) createWire(ctx context.Context, cloudaccount *models.SCloudaccount, vpcId, zoneId, wireName, desc string) (string, error) {
	wire := &models.SWire{
		Bandwidth: 10000,
		Mtu:       1500,
	}
	wire.VpcId = vpcId
	wire.ZoneId = zoneId
	wire.IsEmulated = false
	wire.Name = wireName
	wire.DomainId = cloudaccount.GetOwnerId().GetDomainId()
	wire.Description = desc
	wire.Status = api.WIRE_STATUS_AVAILABLE
	wire.SetModelManager(models.WireManager, wire)
	err := models.WireManager.TableSpec().Insert(ctx, wire)
	if err != nil {
		return "", err
	}
	log.Infof("create wire %s succussfully", wire.GetId())
	return wire.GetId(), nil
}

var ipMaskLen int8 = 24

func (self *CloudAccountSyncVMwareNetworkTask) ipPool(cloudaccount *models.SCloudaccount, wires map[string]*models.SWire) (*sIPPool, error) {
	networks := make([]models.SNetwork, 0, len(wires))
	if wires == nil {
		obj, err := models.VpcManager.FetchById(api.DEFAULT_VPC_ID)
		if err != nil {
			return nil, errors.Wrap(err, "unable fetch defaut vpc")
		}
		networks, err = obj.(*models.SVpc).GetNetworks()
		if err != nil {
			return nil, errors.Wrap(err, "unable to get networks of vpc default")
		}
	} else {
		for _, wire := range wires {
			nets, err := wire.GetNetworks(cloudaccount.GetOwnerId(), rbacutils.ScopeDomain)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to fetch networks of wire %s", wire.GetId())
			}
			networks = append(networks, nets...)
		}
	}
	pool := newIPPool(len(networks))
	for i := range networks {
		startIp, _ := netutils.NewIPV4Addr(networks[i].GuestIpStart)
		endIp, _ := netutils.NewIPV4Addr(networks[i].GuestIpEnd)
		pool.Insert(startIp, sSimpleNet{
			Diff:   endIp - startIp,
			Vlan:   int32(networks[i].VlanId),
			Id:     networks[i].Id,
			WireId: networks[i].WireId,
		})
	}
	return pool, nil
}

func (self *CloudAccountSyncVMwareNetworkTask) zoneId() (string, error) {
	if !self.Params.Contains("zone") {
		zoneids, err := models.CloudaccountManager.FetchEsxiZoneIds()
		if err != nil {
			return "", errors.Wrap(err, "unable to fetch esxi zoneids")
		}
		return zoneids[0], nil
	}
	zone, _ := self.Params.GetString("zone")
	obj, err := models.ZoneManager.FetchByIdOrName(self.UserCred, zone)
	if err != nil {
		return "", errors.Wrapf(err, "unable to fetch zone %q", zone)
	}
	return obj.GetId(), nil
}

func (self *CloudAccountSyncVMwareNetworkTask) fetchWires(cloudaccount *models.SCloudaccount, zoneId string) (map[string]*models.SWire, error) {
	q := models.WireManager.Query().Equals("zone_id", zoneId)
	q = models.WireManager.FilterByOwner(q, cloudaccount.GetOwnerId(), rbacutils.ScopeDomain)
	wires := make([]models.SWire, 0, 1)
	err := db.FetchModelObjects(models.WireManager, q, &wires)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]*models.SWire, len(wires))
	for i := range wires {
		ret[wires[i].GetId()] = &wires[i]
	}
	return ret, nil
}

func (self *CloudAccountSyncVMwareNetworkTask) fetchEsxiZoneIds() ([]string, error) {
	q := models.BaremetalagentManager.Query().Equals("agent_type", "esxiagent").Asc("created_at")
	agents := make([]models.SBaremetalagent, 0, 1)
	err := db.FetchModelObjects(models.BaremetalagentManager, q, &agents)
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

func (self *CloudAccountSyncVMwareNetworkTask) expandIPRnage(ips []netutils.IPV4Addr, limitLow, limitUp netutils.IPV4Addr, expand func(esxi.SIPProc) bool, ipPool esxi.SIPPool, existedIpPool *sIPPool, existed func(sSimpleNet)) []CASimpleNetConf {
	ret := make([]CASimpleNetConf, 0)
	for i := 0; i < len(ips); i++ {
		log.Infof("ipPool: %v", ipPool)
		log.Infof("existedIpPool: %v", existedIpPool)
		ip := ips[i]
		if net, ok := existedIpPool.Get(ip); ok {
			if existed != nil {
				existed(net)
			}
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
		log.Infof("ip: %s", ip.String())
		// find startip
		startIp := ip - 1
		for ; startIp >= netLimitLow; startIp-- {
			log.Infof("startIp: %s", startIp)
			if _, ok := existedIpPool.Get(startIp); ok {
				log.Infof("existedIpPool get startIp")
				break
			}
			if _, ok := ipPool.Get(startIp); ok {
				log.Infof("ipPool get startIp")
				break
			}
		}
		endIp := ip + 1
		for ; endIp <= netLimitUp; endIp++ {
			log.Infof("endIp: %s", endIp)
			if _, ok := existedIpPool.Get(endIp); ok {
				log.Infof("existedIpPool get endIp")
				break
			}
			if proc, ok := ipPool.Get(endIp); ok {
				log.Infof("existedIpPool get endIp")
				if expand(proc) {
					i++
					continue
				}
				break
			}
		}
		ret = append(ret, CASimpleNetConf{
			IpStart: (startIp + 1).String(),
			IpEnd:   (endIp - 1).String(),
			IpMask:  ipMaskLen,
			Gateway: (net + netutils.IPV4Addr(options.Options.DefaultNetworkGatewayAddressEsxi)).String(),
		})
		// Avoid assigning already assigned ip subnet
		existedIpPool.Insert(startIp+1, sSimpleNet{
			Diff: endIp - startIp - 2,
		})
	}
	return ret
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
	Diff   netutils.IPV4Addr
	Id     string
	Vlan   int32
	WireId string
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
	if index > len(pl.netranges) || index < 0 {
		return sSimpleNet{}, false
	}
	if index < len(pl.netranges) && pl.netranges[index] == ip {
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

type CAPWire struct {
	VsId          string
	WireId        string
	Name          string
	Distributed   bool
	Description   string
	Hosts         []esxi.SSimpleHostDev
	HostNetworks  []CANetConf
	GuestNetworks []CANetConf
}

type CASimpleNetConf struct {
	IpStart string `json:"guest_ip_start"`
	IpEnd   string `json:"guest_ip_end"`
	IpMask  int8   `json:"guest_ip_mask"`
	Gateway string `json:"guest_gateway"`
	VlanID  int32  `json:"vlan_id"`
}

type CANetConf struct {
	CASimpleNetConf

	Name        string `json:"name"`
	Description string `json:"description"`
}
