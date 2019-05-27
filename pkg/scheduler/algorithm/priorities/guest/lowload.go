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
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

type LowLoadPriority struct {
	priorities.BasePriority
}

func (p *LowLoadPriority) Name() string {
	return "host_lowload"
}

func (p *LowLoadPriority) Clone() core.Priority {
	return &LowLoadPriority{}
}

func (p *LowLoadPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	getter := c.Getter()
	cpuCommitRate := float64(getter.RunningCPUCount()) / float64(getter.TotalCPUCount(false))
	memCommitRate := float64(getter.RunningMemorySize()) / float64(getter.TotalMemorySize(false))
	if cpuCommitRate < 0.5 && memCommitRate < 0.5 {
		score := 10 * (1 - cpuCommitRate - memCommitRate)
		h.SetScore(int(score))
	}
	return h.GetResult()
}

func (p *LowLoadPriority) ScoreIntervals() score.Intervals {
	return score.NewIntervals(0, 1, 5)
}
