package baremetal

import (
	o "yunion.io/x/onecloud/cmd/scheduler/options"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
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
