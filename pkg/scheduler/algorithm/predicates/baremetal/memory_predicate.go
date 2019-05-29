package baremetal

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type MemoryPredicate struct {
	BasePredicate
}

func (p *MemoryPredicate) Name() string {
	return "baremetal_memory"
}

func (p *MemoryPredicate) Clone() core.FitPredicate {
	return &MemoryPredicate{}
}

func (p *MemoryPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	d := u.SchedData()

	useRsvd := h.UseReserved()
	getter := c.Getter()
	freeMemSize := getter.FreeMemorySize(useRsvd)
	reqMemSize := int64(d.Memory)
	if freeMemSize < reqMemSize {
		totalMemSize := getter.TotalMemorySize(useRsvd)
		h.AppendInsufficientResourceError(reqMemSize, totalMemSize, freeMemSize)
		h.SetCapacity(0)
	} else {
		if reqMemSize/freeMemSize != 1 {
			h.Exclude2("memory", freeMemSize, reqMemSize)
		} else {
			h.SetCapacity(1)
		}
	}

	return h.GetResult()
}
