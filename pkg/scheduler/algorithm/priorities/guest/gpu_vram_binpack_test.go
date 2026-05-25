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
	"testing"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func TestScoreVramBinpack(t *testing.T) {
	mk := func(memMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{MemorySize: memMb}
	}

	cases := []struct {
		name      string
		devs      []*core.IsolatedDeviceDesc
		perDevMin int
		// score is approximate; assert it falls in [minScore, maxScore].
		minScore int
		maxScore int
	}{
		{
			name:      "tight fit (24GiB request on 24GiB GPUs) → near 100",
			devs:      []*core.IsolatedDeviceDesc{mk(24576), mk(24576)},
			perDevMin: 24576,
			minScore:  100, maxScore: 100,
		},
		{
			name:      "loose fit (24GiB request on 80GiB GPUs) → low",
			devs:      []*core.IsolatedDeviceDesc{mk(81920), mk(81920)},
			perDevMin: 24576,
			minScore:  25, maxScore: 35, // 24576/81920 ≈ 30
		},
		{
			name:      "mixed: ignore non-fitting; score from fitting only",
			devs:      []*core.IsolatedDeviceDesc{mk(10000), mk(24576)}, // 10G excluded (< 20G req)
			perDevMin: 20000,
			minScore:  80, maxScore: 85, // 20000/24576 ≈ 81
		},
		{
			name:      "unknown MemorySize=0 → ignored, no influence",
			devs:      []*core.IsolatedDeviceDesc{mk(0), mk(0), mk(24576)},
			perDevMin: 24576,
			minScore:  100, maxScore: 100, // single 24G GPU → tight
		},
		{
			name:      "no fitting devices → 0",
			devs:      []*core.IsolatedDeviceDesc{mk(8000)},
			perDevMin: 40000,
			minScore:  0, maxScore: 0,
		},
		{
			name:      "request 0 (unconstrained) → 0 (don't bias)",
			devs:      []*core.IsolatedDeviceDesc{mk(24576)},
			perDevMin: 0,
			minScore:  0, maxScore: 0,
		},
		{
			name:      "host A (24G average) vs host B (80G average) — A scores higher",
			devs:      []*core.IsolatedDeviceDesc{mk(24576)},
			perDevMin: 20480,
			minScore:  80, maxScore: 100,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := scoreVramBinpack(c.devs, c.perDevMin)
			if got < c.minScore || got > c.maxScore {
				t.Errorf("got score %d, want in [%d, %d]", got, c.minScore, c.maxScore)
			}
		})
	}
}

// Verify the A vs B ordering claim from the plan: same request lands
// higher-scoring on the tight-fit host than on the loose-fit host.
func TestScoreVramBinpack_HostOrdering(t *testing.T) {
	mk := func(memMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{MemorySize: memMb}
	}
	req := 20480                                              // 20 GiB
	hostA := []*core.IsolatedDeviceDesc{mk(24576), mk(24576)} // tight
	hostB := []*core.IsolatedDeviceDesc{mk(81920), mk(81920)} // loose
	sA := scoreVramBinpack(hostA, req)
	sB := scoreVramBinpack(hostB, req)
	if sA <= sB {
		t.Errorf("expected tight-fit host A (%d) > loose-fit host B (%d)", sA, sB)
	}
}
