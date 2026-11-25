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

package appsrv

import (
	"math"
	"testing"
)

func TestParseCgroupV2MemoryMax(t *testing.T) {
	cases := []struct {
		in     string
		want   uint64
		wantOK bool
	}{
		{"max", 0, false},
		{"max\n", 0, false},
		{"2183135232", 2183135232, true},
		{"2183135232\n", 2183135232, true},
		{"0", 0, false},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, c := range cases {
		got, ok := parseCgroupV2MemoryMax(c.in)
		if ok != c.wantOK || got != c.want {
			t.Fatalf("parseCgroupV2MemoryMax(%q)=(%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestParseCgroupV1MemoryLimit(t *testing.T) {
	host := uint64(32 * 1024 * 1024 * 1024)
	lim, ok := parseCgroupV1MemoryLimit(2183135232, host)
	if !ok || lim != 2183135232 {
		t.Fatalf("expected limited cgroup, got %d ok=%v", lim, ok)
	}
	if _, ok := parseCgroupV1MemoryLimit(1<<63-4096, host); ok {
		t.Fatal("near-max v1 limit should be treated as unlimited")
	}
	if _, ok := parseCgroupV1MemoryLimit(host, host); ok {
		t.Fatal("limit >= host total should be unlimited")
	}
}

func TestParseCgroupV2CPUMax(t *testing.T) {
	cores, ok := parseCgroupV2CPUMax("266700 100000")
	if !ok {
		t.Fatal("expected ok")
	}
	if math.Abs(cores-2.667) > 1e-9 {
		t.Fatalf("cores=%v, want 2.667", cores)
	}
	if _, ok := parseCgroupV2CPUMax("max 100000"); ok {
		t.Fatal("max should be unlimited")
	}
	cores, ok = parseCgroupV2CPUMax("100000")
	if !ok || math.Abs(cores-1.0) > 1e-9 {
		t.Fatalf("default period cores=%v ok=%v", cores, ok)
	}
}

func TestParseCgroupV1CPUQuota(t *testing.T) {
	cores, ok := parseCgroupV1CPUQuota(266700, 100000)
	if !ok || math.Abs(cores-2.667) > 1e-9 {
		t.Fatalf("cores=%v ok=%v", cores, ok)
	}
	if _, ok := parseCgroupV1CPUQuota(-1, 100000); ok {
		t.Fatal("negative quota is unlimited")
	}
}
