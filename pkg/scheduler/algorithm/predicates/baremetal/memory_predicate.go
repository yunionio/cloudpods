package baremetal

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
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

	freeMemSize := h.GetInt64("FreeMemSize", 0)
	reqMemSize := d.VMEMSize
	if freeMemSize < reqMemSize {
		totalMemSize := h.GetInt64("MemSize", 0)
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
