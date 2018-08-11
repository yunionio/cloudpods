package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// NestPredicate will filter whether the current host is turned on KVM,
//if the scheduling specified settings are inconsistent, then the host
// will be filtered out.
type NestPredicate struct {
	predicates.BasePredicate
}

func (p *NestPredicate) Name() string {
	return "host_nest"
}

func (p *NestPredicate) Clone() core.FitPredicate {
	return &NestPredicate{}
}

func (p *NestPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	d := u.SchedData()

	if d.Meta["kvm"] == "enabled" {
		if hc.Metadata["nest"] != "enabled" {
			h.Exclude(predicates.ErrNotSupportNest)
		}
	}

	return h.GetResult()
}
