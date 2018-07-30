package baremetal

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
	"github.com/yunionio/pkg/util/sets"
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

	bm, err := h.BaremetalCandidate()
	if err != nil {
		return false, nil, err
	}

	if !ExpectedStatus.Has(bm.Status) {
		h.Exclude2("status", bm.Status, ExpectedStatus)
		return h.GetResult()
	}

	if !bm.Enabled {
		h.Exclude2("enable_status", "disable", "enable")
		return h.GetResult()
	}

	if bm.ServerID == "" {
		h.SetCapacity(1)
	} else {
		h.AppendPredicateFailMsg(predicates.ErrBaremetalHasAlreadyBeenOccupied)
		h.SetCapacity(0)
	}

	return h.GetResult()
}
