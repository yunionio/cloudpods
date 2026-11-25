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

package container_device

import "testing"

func TestParseNpuSmiChipCounts(t *testing.T) {
	out := `
	Total Count                    : 2

	NPU ID                         : 0
	Chip Count                     : 1

	NPU ID                         : 1
	Chip Count                     : 2
	Chip ID                        : 0
	Chip ID                        : 1
`
	got := parseNpuSmiChipCounts(out)
	if got[0] != 1 || got[1] != 2 || len(got) != 2 {
		t.Fatalf("chip counts = %#v", got)
	}
}

func TestParseNpuSmiNames(t *testing.T) {
	out := `NPU ID                         Chip ID                        Chip Logic ID                  Chip Name
0                              0                              0                              Ascend 910B2
0                              1                              -                              Mcu
1                              0                              1                              Ascend 910B2`
	got := parseNpuSmiNames(out)
	if got[0] != "Ascend 910B2" || got[1] != "Ascend 910B2" || len(got) != 2 {
		t.Fatalf("NPU names = %#v", got)
	}
}

func TestParseNpuSmiHBMCapacity(t *testing.T) {
	out := `NPU ID                         : 0
DDR Capacity(MB)               : 0
HBM Capacity(MB)               : 65536
HBM Clock Speed(MHz)           : 1600`
	if got := parseNpuSmiHBMCapacity(out); got != 65536 {
		t.Fatalf("HBM capacity = %d, want 65536", got)
	}
}
