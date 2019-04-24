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
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// CPUPredicate check the current resources of the CPU is available,
// it returns the maximum available capacity.
type CPUPredicate struct {
	predicates.BasePredicate
}

func (f *CPUPredicate) Name() string {
	return "host_cpu"
}

func (f *CPUPredicate) Clone() core.FitPredicate {
	return &CPUPredicate{}
}

func (f *CPUPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.IsPublicCloudProvider() {
		return false, nil
	}

	data := u.SchedData()

	if data.Ncpu <= 0 {
		return false, nil
	}

	return true, nil
}

func (f *CPUPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)
	d := u.SchedData()
	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	useRsvd := h.UseReserved()
	freeCPUCount := hc.GetFreeCPUCount(useRsvd)
	reqCPUCount := int64(d.Ncpu)
	if freeCPUCount < reqCPUCount {
		totalCPUCount := hc.GetTotalCPUCount(useRsvd)
		h.AppendInsufficientResourceError(reqCPUCount, totalCPUCount, freeCPUCount)
	}

	h.SetCapacity(freeCPUCount / reqCPUCount)
	return h.GetResult()
}
