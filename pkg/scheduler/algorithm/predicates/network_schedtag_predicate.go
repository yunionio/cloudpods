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

package predicates

import (
	"fmt"
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type NetworkSchedtagPredicate struct {
	*BaseSchedtagPredicate
}

func (p *NetworkSchedtagPredicate) Name() string {
	return "network_schedtag"
}

func (p *NetworkSchedtagPredicate) Clone() core.FitPredicate {
	return &NetworkSchedtagPredicate{
		BaseSchedtagPredicate: NewBaseSchedtagPredicate(),
	}
}

func (p *NetworkSchedtagPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(p, u, cs)
}

type netW struct {
	*computeapi.NetworkConfig
}

func (n netW) Keyword() string {
	return "net"
}

func (n netW) ResourceKeyword() string {
	return "network"
}

func (n netW) IsSpecifyResource() bool {
	return n.Network != ""
}

func (n netW) GetSchedtags() []*computeapi.SchedtagConfig {
	return n.NetworkConfig.Schedtags
}

func (p *NetworkSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	ret := make([]ISchedtagCustomer, 0)
	for _, net := range u.SchedData().Networks {
		ret = append(ret, &netW{net})
	}
	return ret
}

func (p *NetworkSchedtagPredicate) GetResources(c core.Candidater) []ISchedtagCandidateResource {
	ret := make([]ISchedtagCandidateResource, 0)
	for _, network := range c.Getter().Networks() {
		ret = append(ret, network)
	}
	return ret
}

func (p *NetworkSchedtagPredicate) IsResourceMatchInput(input ISchedtagCustomer, res ISchedtagCandidateResource) bool {
	net := input.(*netW)
	network := res.(*api.CandidateNetwork)
	if net.Network != "" {
		if !(network.Id == net.Network || network.Name == net.Network) {
			return false
		}
	}
	return true
}

func (p *NetworkSchedtagPredicate) IsResourceFitInput(u *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason {
	network := res.(*api.CandidateNetwork)
	net := input.(*netW)
	if net.Network != "" {
		if network.Id != net.Network && network.Name != net.Network {
			return &FailReason{
				Reason: fmt.Sprintf("Network name %s != (%s:%s)", net.Network, network.Name, network.Id),
				Type:   NetworkMatch,
			}
		}
	}
	if net.Wire != "" {
		if network.WireId != net.Wire {
			return &FailReason{
				Reason: fmt.Sprintf("Wire %s != %s", net.Wire, network.WireId),
				Type:   NetworkWire,
			}
		}
	}

	if net.Network == "" {
		netTypes := p.GetNetworkTypes(net.NetType)
		if !utils.IsInStringArray(network.ServerType, netTypes) {
			return &FailReason{
				Reason: fmt.Sprintf("Network %s type %s not in %v", network.Name, network.ServerType, netTypes),
				Type:   NetworkTypeMatch,
			}
		}
		schedData := u.SchedData()
		if net.Private {
			if network.IsPublic {
				return &FailReason{
					Reason: fmt.Sprintf("Network %s is public", network.Name),
					Type:   NetworkPublic,
				}
			}
			if network.ProjectId != schedData.Project && !utils.IsInStringArray(schedData.Project, network.GetSharedProjects()) {
				return &FailReason{
					Reason: fmt.Sprintf("Network project %s + %v not owner by %s", network.ProjectId, network.GetSharedProjects(), schedData.Project),
					Type:   NetworkOwner,
				}
			}
		} else {
			if !network.IsPublic {
				return &FailReason{
					fmt.Sprintf("Network %s is private", network.Name),
					NetworkPrivate,
				}
			}
			if rbacutils.TRbacScope(network.PublicScope) == rbacutils.ScopeDomain {
				netDomain := network.DomainId
				reqDomain := net.Domain
				if netDomain != reqDomain {
					return &FailReason{
						fmt.Sprintf("Network domain scope %s not owner by %s", netDomain, reqDomain),
						NetworkDomain,
					}
				}
			}
		}
	}

	if len(net.Address) > 0 {
		ipAddr, err := netutils.NewIPV4Addr(net.Address)
		if err != nil {
			return &FailReason{
				fmt.Sprintf("Invalid ip address %s: %v", net.Address, err),
				NetworkRange,
			}
		}
		if !network.GetIPRange().Contains(ipAddr) {
			return &FailReason{
				fmt.Sprintf("Address %s not in range", net.Address),
				NetworkRange,
			}
		}
	}
	free, err := network.GetFreeAddressCount()
	if err != nil {
		return &FailReason{
			Reason: fmt.Sprintf("get free address count: %v", err),
			Type:   NetworkFreeCount,
		}
	}
	req := u.SchedData().Count
	if free < req {
		return &FailReason{
			Reason: fmt.Sprintf("Network %s no free IPs, free %d, require %d", network.Name, free, req),
			Type:   NetworkFreeCount,
		}
	}
	return nil
}

func (p *NetworkSchedtagPredicate) GetNetworkTypes(specifyType string) []string {
	netTypes := p.GetHypervisorDriver().GetRandomNetworkTypes()
	if len(specifyType) > 0 {
		netTypes = []string{specifyType}
	}
	return netTypes
}

func (p *NetworkSchedtagPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return p.BaseSchedtagPredicate.Execute(p, u, c)
}

func (p *NetworkSchedtagPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	p.BaseSchedtagPredicate.OnPriorityEnd(p, u, c)
}

func (p *NetworkSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	p.BaseSchedtagPredicate.OnSelectEnd(p, u, c, count)
}

func (p *NetworkSchedtagPredicate) GetCandidateResourceSortScore(selectRes ISchedtagCandidateResource) int64 {
	cnt, err := selectRes.(*api.CandidateNetwork).GetFreeAddressCount()
	if err != nil {
		return -1
	}
	return int64(cnt)
}

type SortNetworks struct {
	requireNet *netW
	nets       []*api.CandidateNetwork
}

func newSortNetworks(req *netW, nets []*api.CandidateNetwork) *SortNetworks {
	return &SortNetworks{
		requireNet: req,
		nets:       nets,
	}
}

func (ns *SortNetworks) Len() int {
	return len(ns.nets)
}

func (ns *SortNetworks) Swap(i, j int) {
	ns.nets[i], ns.nets[j] = ns.nets[j], ns.nets[i]
}

func (ns *SortNetworks) Less(i, j int) bool {
	// match order by project_id, domain_id
	n1 := ns.nets[i]
	n2 := ns.nets[j]
	reqProject := ns.requireNet.Project
	reqDomain := ns.requireNet.Domain
	if n1.ProjectId == reqProject && n2.ProjectId != reqProject {
		return true
	}
	if n1.DomainId == reqDomain && n2.DomainId != reqDomain {
		return true
	}
	return false
}

func (ns *SortNetworks) Results() []ISchedtagCandidateResource {
	res := make([]ISchedtagCandidateResource, 0)
	for _, n := range ns.nets {
		res = append(res, n)
	}
	return res
}

func (p *NetworkSchedtagPredicate) DoSelect(
	c core.Candidater,
	input ISchedtagCustomer,
	res []ISchedtagCandidateResource,
) []ISchedtagCandidateResource {
	networks := make([]*api.CandidateNetwork, 0)
	reqNet := input.(*netW)
	for _, netObj := range res {
		networks = append(networks, netObj.(*api.CandidateNetwork))
	}
	sNets := newSortNetworks(reqNet, networks)
	sort.Sort(sNets)
	return sNets.Results()
}

func (p *NetworkSchedtagPredicate) AddSelectResult(index int, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource) {
	networkIds := []string{}
	for _, res := range selectRes {
		networkIds = append(networkIds, res.GetId())
	}
	ret := &schedapi.CandidateNet{
		Index:      index,
		NetworkIds: networkIds,
	}
	log.Debugf("Suggestion networks %v for net%d", networkIds, index)
	output.Nets = append(output.Nets, ret)
}
