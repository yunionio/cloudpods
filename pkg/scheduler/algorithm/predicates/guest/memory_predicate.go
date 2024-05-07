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
	"context"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// MemoryPredicate filter current resources free memory capacity is meet,
// if it is satisfied to return the size of the memory that
// can carry the scheduling request.
type MemoryPredicate struct {
	predicates.BasePredicate
}

func (p *MemoryPredicate) Name() string {
	return "host_memory"
}

func (p *MemoryPredicate) Clone() core.FitPredicate {
	return &MemoryPredicate{}
}

func (p *MemoryPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	driver := u.GetHypervisorDriver()
	if driver != nil && !driver.DoScheduleMemoryFilter() {
		return false, nil
	}

	data := u.SchedData()

	if data.Memory <= 0 {
		return false, nil
	}

	return true, nil
}

func (p *MemoryPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	d := u.SchedData()

	useRsvd := h.UseReserved()
	getter := c.Getter()
	freeMemSize := getter.FreeMemorySize(useRsvd)
	reqMemSize := int64(d.Memory)
	if freeMemSize < reqMemSize {
		totalMemSize := getter.TotalMemorySize(useRsvd)
		h.AppendInsufficientResourceError(reqMemSize, totalMemSize, freeMemSize)
	}

	h.SetCapacity(freeMemSize / reqMemSize)
	return h.GetResult()
}
