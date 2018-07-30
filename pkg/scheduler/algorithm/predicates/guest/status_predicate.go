package guest

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

const (
	ExpectedStatus       = "running"
	ExpectedHostStatus   = "online"
	ExpectedEnableStatus = "enable"
)

// StatusPredicate is to filter the current state of host is available,
// not available host's capacity will be set to 0 and filtered out.
type StatusPredicate struct {
	predicates.BasePredicate
}

func (p *StatusPredicate) Name() string {
	return "host_status"
}

func (p *StatusPredicate) Clone() core.FitPredicate {
	return &StatusPredicate{}
}

func (p *StatusPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	curStatus := h.Get("Status").(string)
	curHostStatus := h.Get("HostStatus").(string)
	curEnableStatus := h.Get("EnableStatus").(string)

	if curStatus != ExpectedStatus {
		h.Exclude2("status", curStatus, ExpectedStatus)
	}

	if curHostStatus != ExpectedHostStatus {
		h.Exclude2("host_status", curHostStatus, ExpectedHostStatus)
	}

	if curEnableStatus != ExpectedEnableStatus {
		h.Exclude2("enable_status", curEnableStatus, ExpectedEnableStatus)
	}

	return h.GetResult()
}
