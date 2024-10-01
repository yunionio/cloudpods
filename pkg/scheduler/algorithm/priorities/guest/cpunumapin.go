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
	"yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type CpuNumaPinPriority struct {
	priorities.BasePriority
}

func (p *CpuNumaPinPriority) Name() string {
	return "cpu_numa_pin"
}

func (p *CpuNumaPinPriority) Clone() core.Priority {
	return &CpuNumaPinPriority{}
}

func (p *CpuNumaPinPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	getter := c.Getter()
	if getter.Host().EnableNumaAllocate && getter.NumaAllocateEnabled() && len(u.SchedData().CpuNumaPin) == 0 {
		cpuNumaFree := getter.GetFreeCpuNuma()

		reqCpuCount := u.SchedInfo.Ncpu
		reqMemSize := u.SchedInfo.Memory
		nodeCount := 1
		for ; nodeCount <= len(cpuNumaFree); nodeCount *= 2 {
			if scheduler.NodesFreeCpuEnough(nodeCount, reqCpuCount, cpuNumaFree) &&
				scheduler.NodesFreeMemSizeEnough(nodeCount, int(reqMemSize), cpuNumaFree) {
				break
			}
		}
		if nodeCount > 0 && nodeCount <= len(cpuNumaFree) {
			score := 100 / nodeCount
			h.SetScore(int(score))
		}
	}
	return h.GetResult()
}
