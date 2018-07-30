package guest

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/priorities"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

type AvoidSameClusterPriority struct {
	priorities.BasePriority
	ClusterTbl map[string]int
}

func (p *AvoidSameClusterPriority) Name() string {
	return "guest_avoid_same_cluster"
}

func (p *AvoidSameClusterPriority) Clone() core.Priority {
	return &AvoidSameClusterPriority{ClusterTbl: make(map[string]int)}
}

func (p *AvoidSameClusterPriority) PreExecute(u *core.Unit, cs []core.Candidater) (bool, []core.PredicateFailureReason, error) {
	d := u.SchedData()
	clusterTbl := make(map[string]int, 0)
	ownerTenantID := d.OwnerTenantID

	for _, c := range cs {
		hc, err := p.HostCandidate(c)
		if err != nil {
			return false, nil, err
		}

		if count, ok := hc.Tenants[ownerTenantID]; ok && count > 0 {
			clusterId := hc.ClusterID

			if count0, ok := clusterTbl[clusterId]; ok {
				clusterTbl[clusterId] = count0 + int(count)
			} else {
				clusterTbl[clusterId] = int(count)
			}
		}
	}

	p.ClusterTbl = clusterTbl
	return true, nil, nil
}

func (p *AvoidSameClusterPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	if count, ok := p.ClusterTbl[c.Get("ClusterID").(string)]; ok {
		h.SetScore(-20 * count)
	}

	return h.GetResult()
}
