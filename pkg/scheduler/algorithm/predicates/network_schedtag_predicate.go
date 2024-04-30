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
	"context"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
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

func (p *NetworkSchedtagPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(ctx, p, u, cs)
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

func (n netW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return n.NetworkConfig.JSON(n.NetworkConfig)
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

func (p *NetworkSchedtagPredicate) IsResourceMatchInput(ctx context.Context, input ISchedtagCustomer, res ISchedtagCandidateResource) bool {
	net := input.(*netW)
	network := res.(*api.CandidateNetwork)
	if net.Network != "" {
		if !(network.Id == net.Network || network.Name == net.Network) {
			return false
		}
	}
	return true
}

func (p *NetworkSchedtagPredicate) IsResourceFitInput(ctx context.Context, u *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason {
	network := res.(*api.CandidateNetwork)
	net := input.(*netW)
	return IsNetworkAvailable(ctx, c, u.SchedData(), net.NetworkConfig, network, p.GetNetworkTypes(net.NetType), nil)
}

func (p *NetworkSchedtagPredicate) GetNetworkTypes(specifyType string) []string {
	netTypes := []string{}
	driver := p.GetHypervisorDriver()
	if driver != nil {
		netTypes = driver.GetRandomNetworkTypes()
	}
	if len(specifyType) > 0 {
		netTypes = []string{specifyType}
	}
	return netTypes
}

func (p *NetworkSchedtagPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return p.BaseSchedtagPredicate.Execute(ctx, p, u, c)
}

func (p *NetworkSchedtagPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	p.BaseSchedtagPredicate.OnPriorityEnd(p, u, c)
}

func (p *NetworkSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	p.BaseSchedtagPredicate.OnSelectEnd(p, u, c, count)
}

func (p *NetworkSchedtagPredicate) GetCandidateResourceSortScore(selectRes ISchedtagCandidateResource) int64 {
	return int64(selectRes.(*api.CandidateNetwork).FreePort)
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

func (p *NetworkSchedtagPredicate) AddSelectResult(index int, input ISchedtagCustomer, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource) {
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
