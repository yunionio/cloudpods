package guest

import (
	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/api"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

const (
	CONTAINER_ALLOWED_TAG = "container"
)

// HypervisorPredicate is to select candidates match guest hyperviosr
// runtime
type HypervisorPredicate struct {
	predicates.BasePredicate
}

func (f *HypervisorPredicate) Name() string {
	return "host_hypervisor_runtime"
}

func (f *HypervisorPredicate) Clone() core.FitPredicate {
	return &HypervisorPredicate{}
}

func hostHasContainerTag(c core.Candidater) bool {
	aggs := c.GetHostAggregates()
	for _, agg := range aggs {
		if agg.Name == CONTAINER_ALLOWED_TAG {
			return true
		}
	}
	return false
}

func hostAllowRunContainer(c core.Candidater) bool {
	hostType := c.Get("HostType")
	if hostType == api.HostTypeKubelet {
		return true
	}
	if hostHasContainerTag(c) {
		log.Debugf("Host %q has %q tag, allow it run container", c.IndexKey(), CONTAINER_ALLOWED_TAG)
		return true
	}
	return false
}

func (f *HypervisorPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)

	hostType := c.Get("HostType")
	guestNeedType := u.SchedData().Hypervisor

	if guestNeedType != hostType {
		if guestNeedType == api.SchedTypeContainer && hostAllowRunContainer(c) {
			return h.GetResult()
		}
		h.Exclude2(f.Name(), hostType, guestNeedType)
	}
	return h.GetResult()
}
