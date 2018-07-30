package baremetal

import (
	o "github.com/yunionio/onecloud/cmd/scheduler/options"
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
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
