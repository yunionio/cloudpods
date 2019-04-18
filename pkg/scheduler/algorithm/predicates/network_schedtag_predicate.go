package predicates

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

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

func (p *NetworkSchedtagPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(p, u, cs)
}

type netW struct {
	*computeapi.NetworkConfig
}

func (n netW) Keyword() string {
	return "net"
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

func (p *NetworkSchedtagPredicate) IsResourceFitInput(u *core.Unit, res ISchedtagCandidateResource, input ISchedtagCustomer) error {
	network := res.(*api.CandidateNetwork)
	net := input.(*netW)
	if net.Network != "" {
		if network.Id != net.Network && network.Name != net.Network {
			return fmt.Errorf("Network name %s != (%s:%s)", net.Network, network.Name, network.Id)
		}
	}
	if net.Wire != "" {
		if network.WireId != net.Wire {
			return fmt.Errorf("Wire %s != %s", net.Wire, network.WireId)
		}
	}
	netTypes := p.GetNetworkTypes(net.NetType)
	if net.Network == "" && !utils.IsInStringArray(network.ServerType, netTypes) {
		return fmt.Errorf("Network %s type %s not in %v", network.Name, network.ServerType, netTypes)
	}
	schedData := u.SchedData()
	if net.Private {
		if network.IsPublic {
			return fmt.Errorf("Network %s is public", network.Name)
		}
		if network.ProjectId != schedData.Project {
			return fmt.Errorf("Network project %s not owner by %s", network.ProjectId, schedData.Project)
		}
	} else {
		if !network.IsPublic {
			return fmt.Errorf("Network %s is private", network.Name)
		}
	}
	if len(net.Address) > 0 {
		ipAddr, err := netutils.NewIPV4Addr(net.Address)
		if err != nil {
			return fmt.Errorf("Invalid ip address %s: %v", net.Address, err)
		}
		if !network.GetIPRange().Contains(ipAddr) {
			return fmt.Errorf("Address %s not in range", net.Address)
		}
	}
	free := network.GetFreeAddressCount()
	req := u.SchedData().Count
	if free < req {
		return fmt.Errorf("Network %s no free IPs, free %d, require %d", network.Name, free, req)
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

func (p *NetworkSchedtagPredicate) GetCandidateResourceSortScore(selectRes ISchedtagCandidateResource) int {
	return selectRes.(*api.CandidateNetwork).GetFreeAddressCount()
}

func (p *NetworkSchedtagPredicate) DoSelect(
	c core.Candidater,
	input ISchedtagCustomer,
	res []ISchedtagCandidateResource,
) []ISchedtagCandidateResource {
	return res
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
