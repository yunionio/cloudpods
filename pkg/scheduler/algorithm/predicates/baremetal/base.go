package baremetal

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type BasePredicate struct {
	predicates.BasePredicate
}

func (p *BasePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if o.GetOptions().DisableBaremetalPredicates {
		return false, nil
	}
	return true, nil
}
