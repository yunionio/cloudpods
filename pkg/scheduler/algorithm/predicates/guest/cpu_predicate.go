package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// CPUPredicate check the current resources of the CPU is available,
// it returns the maximum available capacity.
type CPUPredicate struct {
	predicates.BasePredicate
}

func (f *CPUPredicate) Name() string {
	return "host_cpu"
}

func (f *CPUPredicate) Clone() core.FitPredicate {
	return &CPUPredicate{}
}

func (f *CPUPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.IsPublicCloudProvider() {
		return false, nil
	}

	data := u.SchedData()

	if data.VCPUCount <= 0 {
		return false, nil
	}

	return true, nil
}

func (f *CPUPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)
	d := u.SchedData()
	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	useRsvd := h.UseReserved()
	freeCPUCount := hc.GetFreeCPUCount(useRsvd)
	reqCPUCount := d.VCPUCount
	if freeCPUCount < reqCPUCount {
		totalCPUCount := hc.GetTotalCPUCount(useRsvd)
		h.AppendInsufficientResourceError(reqCPUCount, totalCPUCount, freeCPUCount)
	}

	h.SetCapacity(freeCPUCount / reqCPUCount)
	return h.GetResult()
}
