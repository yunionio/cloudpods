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

package compute

import "testing"

func TestParseSkuSpec(t *testing.T) {
	cases := []struct {
		in      string
		cpu     int
		memMB   int
		wantErr bool
	}{
		{"2c2g", 2, 2048, false},
		{"2C2G", 2, 2048, false},
		{"4c8g", 4, 8192, false},
		{"2核2G", 2, 2048, false},
		{"2核2g", 2, 2048, false},
		{" 4核 16G ", 4, 16384, false},
		{"2c/2g", 2, 2048, false},
		{"2x2g", 2, 2048, false},
		{"2vcpu2gb", 2, 2048, false},
		{"2c2048m", 2, 2048, false},
		{"2g", 0, 0, true},
		{"abc", 0, 0, true},
		{"", 0, 0, true},
	}
	for _, c := range cases {
		cpu, mem, err := ParseSkuSpec(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseSkuSpec(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSkuSpec(%q) unexpected err: %v", c.in, err)
			continue
		}
		if cpu != c.cpu || mem != c.memMB {
			t.Errorf("ParseSkuSpec(%q)=%d,%d want %d,%d", c.in, cpu, mem, c.cpu, c.memMB)
		}
	}
}

func TestServerSkusListOptionsApplySpec(t *testing.T) {
	opts := &ServerSkusListOptions{Spec: "2核2G"}
	params, err := opts.Params()
	if err != nil {
		t.Fatal(err)
	}
	cpu, _ := params.Int("cpu_core_count")
	mem, _ := params.Int("memory_size_mb")
	if cpu != 2 || mem != 2048 {
		t.Fatalf("params cpu=%d mem=%d want 2/2048; raw=%s", cpu, mem, params.String())
	}
}
