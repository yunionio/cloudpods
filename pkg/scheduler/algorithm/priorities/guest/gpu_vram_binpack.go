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

// GPUVramBinpackPriority prefers hosts whose free GPUs have memory close to
// the request's MemoryMb (bin-packing): avoids landing a small model on a
// large GPU when a tighter-fitting candidate exists. Hosts without a memory
// constraint in the request score 0 — they don't participate.
type GPUVramBinpackPriority struct {
	priorities.BasePriority
}

func (p *GPUVramBinpackPriority) Name() string {
	return "guest-gpu-vram-binpack"
}

func (p *GPUVramBinpackPriority) Clone() core.Priority {
	return &GPUVramBinpackPriority{}
}

func (p *GPUVramBinpackPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	// Take the maximum MemoryMb across the request — for an LLM SKU this
	// is the per-device share of vram_claim_mb (all entries equal); for mixed
	// requests we treat the largest as the bin-pack target.
	perDevMin := 0
	for _, d := range u.SchedData().IsolatedDevices {
		if d.MemoryMb > perDevMin {
			perDevMin = d.MemoryMb
		}
	}
	if perDevMin == 0 {
		// No VRAM constraint — don't influence ranking.
		return h.GetResult()
	}

	score := scoreVramBinpack(c.Getter().UnusedIsolatedDevices(), perDevMin)
	h.SetScore(score)
	return h.GetResult()
}

// scoreVramBinpack is the pure scoring core, factored out so it can be unit
// tested without constructing a CandidatePropertyGetter mock. Average the
// VRAM of free GPUs that satisfy the request, then score 100 * perDevMin /
// avg — tighter fit ranks higher. Devices with MemorySize == 0 are neutral
// (don't influence the average) so unreported rows don't skew rankings.
func scoreVramBinpack(devs []*core.IsolatedDeviceDesc, perDevMin int) int {
	if perDevMin <= 0 {
		return 0
	}
	var totalFitVram, fitCount int
	for _, d := range devs {
		if d.MemorySize <= 0 {
			continue
		}
		if d.MemorySize < perDevMin {
			continue
		}
		totalFitVram += d.MemorySize
		fitCount++
	}
	if fitCount == 0 {
		return 0
	}
	avgVram := totalFitVram / fitCount
	score := 100 * perDevMin / avgVram
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}
