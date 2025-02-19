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
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type AvoidSameHostPriority struct {
	priorities.BasePriority
}

func (p *AvoidSameHostPriority) Name() string {
	return "guest_avoid_same_host"
}

func (p *AvoidSameHostPriority) Clone() core.Priority {
	return &AvoidSameHostPriority{}
}

func (p *AvoidSameHostPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	ownerTenantID := u.SchedData().Project
	if count, ok := c.Getter().ProjectGuests()[ownerTenantID]; ok && count > 0 {
		h.SetScore(-1 * int(float64(count)*0.1))
	}

	return h.GetResult()
}
