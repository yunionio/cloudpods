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

type CreatingPriority struct {
	priorities.BasePriority
}

func (p *CreatingPriority) Name() string {
	return "creating"
}

func (p *CreatingPriority) Clone() core.Priority {
	return &CreatingPriority{}
}

func (p *CreatingPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	creatingGuestCount := c.Getter().CreatingGuestCount()
	if creatingGuestCount > 0 {
		score := -int(creatingGuestCount)
		h.SetScore(score)
	}

	return h.GetResult()
}
