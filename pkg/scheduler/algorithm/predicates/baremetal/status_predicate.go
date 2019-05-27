package baremetal

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

var (
	ExpectedStatus = sets.NewString("running", "start_convert")
)

type StatusPredicate struct {
	BasePredicate
}

func (p *StatusPredicate) Name() string {
	return "baremetal_status"
}

func (p *StatusPredicate) Clone() core.FitPredicate {
	return &StatusPredicate{}
}

func (p *StatusPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	getter := c.Getter()

	status := getter.Status()
	enabled := getter.Enabled()
	if !ExpectedStatus.Has(status) {
		h.Exclude2("status", status, ExpectedStatus)
		return h.GetResult()
	}

	if !enabled {
		h.Exclude2("enable_status", "disable", "enable")
		return h.GetResult()
	}

	if getter.IsEmpty() {
		h.SetCapacity(1)
	} else {
		h.AppendPredicateFailMsg(predicates.ErrBaremetalHasAlreadyBeenOccupied)
		h.SetCapacity(0)
	}

	return h.GetResult()
}
