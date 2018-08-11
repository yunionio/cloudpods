package baremetal

import (
	"fmt"

	//"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/util/baremetal"
)

type StoragePredicate struct {
	BasePredicate
}

func (p *StoragePredicate) Name() string {
	return "baremetal_storage"
}

func (p *StoragePredicate) Clone() core.FitPredicate {
	return &StoragePredicate{}
}

func (p *StoragePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	candidate, err := h.BaremetalCandidate()
	if err != nil {
		return false, nil, err
	}

	layouts, err := baremetal.CalculateLayout(
		schedData.BaremetalDiskConfigs,
		candidate.Storages,
	)

	if err == nil && baremetal.CheckDisksAllocable(layouts, schedData.Disks) {
		h.SetCapacity(int64(1))
	} else {
		h.SetCapacity(int64(0))
		h.AppendPredicateFailMsg(fmt.Sprintf("%s err: %v", predicates.ErrNoEnoughStorage, err))
	}

	return h.GetResult()
}
