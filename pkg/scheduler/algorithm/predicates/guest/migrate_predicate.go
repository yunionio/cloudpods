package guest

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

// MigratePredicate filters whether the current candidate can be migrated.
type MigratePredicate struct {
	predicates.BasePredicate
}

func (p *MigratePredicate) Name() string {
	return "host_migrate"
}

func (p *MigratePredicate) Clone() core.FitPredicate {
	return &MigratePredicate{}
}

func (p *MigratePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	return len(u.SchedData().HostID) > 0, nil
}

func (p *MigratePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	if u.SchedData().HostID == c.IndexKey() {
		h.Exclude(predicates.ErrHostIsSpecifiedForMigration)
	}

	return h.GetResult()
}
