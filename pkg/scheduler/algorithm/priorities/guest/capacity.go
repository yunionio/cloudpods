package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type CapacityPriority struct {
	priorities.BasePriority
}

func (p *CapacityPriority) Name() string {
	return "host_capacity"
}

func (p *CapacityPriority) Clone() core.Priority {
	return &CapacityPriority{}
}

func (p *CapacityPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	capacity := u.GetCapacity(c.IndexKey())
	h.SetRawScore(int(capacity))

	return h.GetResult()
}
