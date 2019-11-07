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

package predicates

import (
	"fmt"
	"math"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type InstanceGroupPredicate struct {
	BasePredicate
}

func (p *InstanceGroupPredicate) Name() string {
	return "instance_group"
}

func (p *InstanceGroupPredicate) Clone() core.FitPredicate {
	return &InstanceGroupPredicate{}
}

func (p *InstanceGroupPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	schedDate := u.SchedData()
	if schedDate.InstanceGroupIds == nil || len(schedDate.InstanceGroupIds) == 0 {
		return false, nil
	}
	return true, nil
}

func (p *InstanceGroupPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return true, nil, nil
}

type SForcedGroupPredicate struct {
	InstanceGroupPredicate
}

func (p *SForcedGroupPredicate) Name() string {
	return "forced_instance_group"
}

func (p *SForcedGroupPredicate) Clone() core.FitPredicate {
	return &SForcedGroupPredicate{}
}

// SForcedGroupPredicate make sure that there is no more guest with same group whose IsForcedSpe is ture in a host
// for all forced groups in u.SchedData.InstanceGroupIds, so that the capacity is the min value of the FreeGroupCounts
func (p *SForcedGroupPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	schedDate := u.SchedData()

	instanceGroups := c.Getter().InstanceGroups()
	minFree := math.MaxInt32
	for _, id := range schedDate.InstanceGroupIds {
		detail := schedDate.InstanceGroupsDetail[id]
		// SForcedGroupPredicate only deal with group whose ForceDispersion is ture
		if detail.ForceDispersion.IsFalse() {
			continue
		}
		var free int
		if _, ok := instanceGroups[id]; ok {
			free, _ = c.Getter().GetFreeGroupCount(id)
			if free < 1 {
				h.AppendPredicateFailMsg(fmt.Sprintf(
					"the number of guests with same instance group '%s' in this host has reached the upper limit",
					instanceGroups[id].GetName()))
				minFree = 0
				break
			}
		} else {
			free = detail.Granularity
		}
		if free < minFree {
			minFree = free
		}
	}
	// chose the min capacity of groups
	h.SetCapacity(int64(minFree))
	return h.GetResult()
}

type SUnForcedGroupPredicate struct {
	InstanceGroupPredicate
}

func (p *SUnForcedGroupPredicate) Name() string {
	return "unforced_instance_group"
}

func (p *SUnForcedGroupPredicate) Clone() core.FitPredicate {
	return &SUnForcedGroupPredicate{}
}

func (p *SUnForcedGroupPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	ret, err := p.InstanceGroupPredicate.PreExecute(u, cs)
	if err != nil || !ret {
		return ret, err
	}
	u.RegisterSelectPriorityUpdater(p.Name(), func(u *core.Unit, origin core.SSelectPriorityValue,
		hostID string) core.SSelectPriorityValue {

		return origin.SubOne()
	})
	return ret, err
}

// SUnForcedGroupPredicate make sure that the guests are assigned to these hosts who has enough FreeGroupCount of
// unforced groups, so that it will improve the priority of these hosts meet the conditions and the priority should
// be the max value of the FreeGroupCounts
func (p *SUnForcedGroupPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason,
	error) {

	h := NewPredicateHelper(p, u, c)
	schedDate := u.SchedData()

	instanceGroups := c.Getter().InstanceGroups()
	maxPriority := 0
	for _, id := range schedDate.InstanceGroupIds {
		detail := schedDate.InstanceGroupsDetail[id]
		// SUnForcedGroupPredicate only deal with group whose ForceDispersion is false
		if detail.ForceDispersion.IsTrue() {
			continue
		}
		var priority int
		if _, ok := instanceGroups[id]; ok {
			free, _ := c.Getter().GetFreeGroupCount(id)
			if free < 1 {
				priority = 0
			}
			priority = free
		} else {
			priority = detail.Granularity
		}
		if priority > maxPriority {
			maxPriority = priority
		}
	}

	// set priority
	h.SetSelectPriority(maxPriority)
	return h.GetResult()
}
