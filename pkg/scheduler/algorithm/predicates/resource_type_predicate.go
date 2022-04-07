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
	"context"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
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

func (p *ResourceTypePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.SchedData().ResourceType == "" {
		return false, nil
	}
	return true, nil
}

func (p *ResourceTypePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	d := u.SchedData()

	hostResType := c.GetResourceType()
	reqResType := d.ResourceType
	if hostResType != reqResType {
		h.Exclude2("resource_type", hostResType, reqResType)
	}

	if hostResType == computeapi.HostResourceTypePrepaidRecycle {
		if c.GetGuestCount() == 0 {
			h.SetCapacity(1)
		} else {
			h.Exclude(ErrPrepaidHostOccupied)
		}
	}
	// TODO: support HostResourceTypeDedicated

	return h.GetResult()
}
