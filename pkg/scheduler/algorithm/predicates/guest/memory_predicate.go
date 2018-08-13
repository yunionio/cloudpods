package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// MemoryPredicate filter current resources free memory capacity is meet,
// if it is satisfied to return the size of the memory that
// can carry the scheduling request.
type MemoryPredicate struct {
	predicates.BasePredicate
}

func (p *MemoryPredicate) Name() string {
	return "host_memory"
}

func (p *MemoryPredicate) Clone() core.FitPredicate {
	return &MemoryPredicate{}
}

func (p *MemoryPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.IsPublicCloudProvider() {
		return false, nil
	}

	data := u.SchedData()

	if data.VMEMSize <= 0 {
		return false, nil
	}

	return true, nil
}

func (p *MemoryPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	d := u.SchedData()
	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	useRsvd := h.UseReserved()
	freeMemSize := hc.GetFreeMemSize(useRsvd)
	reqMemSize := d.VMEMSize
	if freeMemSize < reqMemSize {
		totalMemSize := hc.GetTotalMemSize(useRsvd)
		h.AppendInsufficientResourceError(reqMemSize, totalMemSize, freeMemSize)
	}

	h.SetCapacity(freeMemSize / reqMemSize)
	return h.GetResult()
}
