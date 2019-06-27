package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
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
	return len(u.SchedData().HostId) > 0, nil
}

func (p *MigratePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	if schedData.HostId == c.IndexKey() {
		h.Exclude(predicates.ErrHostIsSpecifiedForMigration)
		return h.GetResult()
	}

	if schedData.LiveMigrate {
		host := c.Getter().Host()
		if schedData.CpuDesc != host.CpuDesc {
			h.Exclude(predicates.ErrHostCpuModelIsNotMatchForLiveMigrate)
			return h.GetResult()
		}
		if len(schedData.CpuMicrocode) > 0 && schedData.CpuMicrocode != host.CpuMicrocode {
			h.Exclude(predicates.ErrHostCpuMicrocodeNotMatchForLiveMigrate)
			return h.GetResult()
		}
	}

	return h.GetResult()
}
