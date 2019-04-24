// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package guest

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

// GroupPredicate filter the packet based on the label information,
// the same group of guests should avoid schedule on same host.
type GroupPredicate struct {
	predicates.BasePredicate
	plugin.BasePlugin

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
			p.ExcludeGroups = append(p.ExcludeGroups, r.GroupId)
		} else if r.Strategy == "require" {
			p.RequireGroups = append(p.RequireGroups, r.GroupId)
		} else if r.Strategy == "avoid" {
			p.AvoidGroups = append(p.AvoidGroups, r.GroupId)
		} else if r.Strategy == "prefer" {
			p.PreferGroups = append(p.PreferGroups, r.GroupId)
		}
	}
	u.AppendSelectPlugin(p)
	return true, nil
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

func (p *GroupPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	if len(p.AvoidGroups) > 0 {
		u.SetFrontScore(
			c.IndexKey(),
			score.NewScore(
				score.TScore(-core.PriorityStep*len(p.AvoidGroups)),
				p.Name()+":avoid",
			))
	}

	if len(p.PreferGroups) > 0 {
		u.SetFrontScore(
			c.IndexKey(),
			score.NewScore(
				score.TScore(core.PriorityStep*len(p.PreferGroups)),
				p.Name()+":prefer",
			))
	}
}
