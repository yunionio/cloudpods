package predicates

import (
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type ResourceTypePredicate struct {
	BasePredicate
}

func (p *ResourceTypePredicate) Name() string {
	return "resource_type"
}

func (p *ResourceTypePredicate) Clone() core.FitPredicate {
	return &ResourceTypePredicate{}
}

func (p *ResourceTypePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.SchedData().ResourceType == "" {
		return false, nil
	}
	return true, nil
}

func (p *ResourceTypePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	d := u.SchedData()

	hostResType := c.GetResourceType()
	reqResType := d.ResourceType
	if hostResType != reqResType {
		h.Exclude2("resource_type", hostResType, reqResType)
	}

	if reqResType == models.HostResourceTypePrepaidRecycle {
		if c.GetGuestCount() == 0 {
			h.SetCapacity(1)
		} else {
			h.Exclude(ErrPrepaidHostOccupied)
		}
	}
	// TODO: support HostResourceTypeDedicated

	return h.GetResult()
}
