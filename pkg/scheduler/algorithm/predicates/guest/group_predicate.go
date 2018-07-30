package guest

import (
	"fmt"

	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

// GroupPredicate filter the packet based on the label information,
// the same group of guests should avoid schedule on same host.
type GroupPredicate struct {
	predicates.BasePredicate

	ExcludeGroups []string
	RequireGroups []string
	AvoidGroups   []string
	PreferGroups  []string
}

func (p *GroupPredicate) Name() string {
	return "host_group"
}

func (p *GroupPredicate) Clone() core.FitPredicate {
	return &GroupPredicate{}
}

func (p *GroupPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	d := u.SchedData()
	if len(d.GroupRelations) == 0 {
		return false, nil
	}

	for _, r := range d.GroupRelations {
		if r.Strategy == "exclude" {
			p.ExcludeGroups = append(p.ExcludeGroups, r.GroupID)
		} else if r.Strategy == "require" {
			p.RequireGroups = append(p.RequireGroups, r.GroupID)
		} else if r.Strategy == "avoid" {
			p.AvoidGroups = append(p.AvoidGroups, r.GroupID)
		} else if r.Strategy == "prefer" {
			p.PreferGroups = append(p.PreferGroups, r.GroupID)
		}
	}
	u.AppendSelectPlugin(p)
	return true, nil
}

func (p *GroupPredicate) OnSelect(u *core.Unit, c core.Candidater) bool {
	if len(p.ExcludeGroups) > 0 {
		return false
	}

	if len(p.RequireGroups) > 0 {
		// TODO: what?
	}

	if len(p.AvoidGroups) > 0 {
		u.IncreaseScore(c.IndexKey(),
			p.Name()+":avoid", -core.PriorityStep*len(p.AvoidGroups),
		)
	}

	if len(p.PreferGroups) > 0 {
		u.IncreaseScore(c.IndexKey(),
			p.Name()+":prefer", core.PriorityStep*len(p.PreferGroups),
		)
	}

	return true
}

func (p *GroupPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
}

func (p *GroupPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {

	h := predicates.NewPredicateHelper(p, u, c)

	g, err := h.GetGroupCounts()
	if err != nil {
		return false, nil, err
	}

	if len(p.ExcludeGroups) > 0 {
		for _, groupId := range p.ExcludeGroups {
			if g.ExistsGroup(groupId) {
				h.Exclude(fmt.Sprintf("exclude by %v:exclude", groupId))
				break
			}
		}
	} else if len(p.RequireGroups) > 0 {
		for _, groupId := range p.RequireGroups {
			if !g.ExistsGroup(groupId) {
				h.Exclude(fmt.Sprintf("exclude by %v:require", groupId))
				break
			}
		}
	}

	return h.GetResult()
}
