package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type AvoidSameHostPriority struct {
	priorities.BasePriority
}

func (p *AvoidSameHostPriority) Name() string {
	return "guest_avoid_same_host"
}

func (p *AvoidSameHostPriority) Clone() core.Priority {
	return &AvoidSameHostPriority{}
}

func (p *AvoidSameHostPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	hc, err := p.HostCandidate(c)
	if err != nil {
		return core.HostPriority{}, err
	}

	ownerTenantID := u.SchedData().OwnerTenantID
	if count, ok := hc.Tenants[ownerTenantID]; ok && count > 0 {
		h.SetScore(-50 * int(count))
	}

	return h.GetResult()
}
