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

	ownerTenantID := u.SchedData().Project
	if count, ok := c.Getter().ProjectGuests()[ownerTenantID]; ok && count > 0 {
		h.SetFrontRawScore(-1 * int(count))
	}

	return h.GetResult()
}
