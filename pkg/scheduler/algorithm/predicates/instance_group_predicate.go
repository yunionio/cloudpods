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
	h := NewPredicateHelper(p, u, c)
	schedDate := u.SchedData()

	instanceGroups := c.Getter().InstanceGroups()
	minFree := math.MaxInt16
	for _, id := range schedDate.InstanceGroupIds {
		var free int
		if _, ok := instanceGroups[id]; ok {
			free, _ = c.Getter().GetFreeGroupCount(id)
			if free < 1 {
				h.AppendPredicateFailMsg(fmt.Sprintf(
					"the number of guests with same group %s in this host has reached the upper limit", id))
				break
			}
		} else {
			detail := schedDate.InstanceGroupsDetail[id]
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
