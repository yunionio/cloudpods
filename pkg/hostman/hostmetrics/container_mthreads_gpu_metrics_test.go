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

package hostmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMthreadsProcessMonitor(t *testing.T) {
	buf, err := os.ReadFile(filepath.Join("testdata", "mthreads-pm.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := parseMthreadsProcessMonitor(string(buf))
	if len(got) != 7 {
		t.Fatalf("rows=%d, want 7", len(got))
	}
	worker := got[2]
	if worker.Index != 0 || worker.Pid != "1306725" || worker.MemUtil != 0.10 || worker.GpuUtil != 100 {
		t.Fatalf("worker=%+v", worker)
	}
	if got[0].MemUtil != 0 || got[0].GpuUtil != 0 {
		t.Fatalf("<1%% parsed as %+v", got[0])
	}
}

func TestParseMthreadsProcessMemory(t *testing.T) {
	buf, err := os.ReadFile(filepath.Join("testdata", "mthreads-processes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := parseMthreadsProcessMemory(string(buf))
	if len(got) != 6 {
		t.Fatalf("rows=%d, want 6", len(got))
	}
	if got[1].Pid != "1305830" || got[1].MemMiB != 36 {
		t.Fatalf("engine=%+v", got[1])
	}
	if got[2].Pid != "1306725" || got[2].MemMiB != 8758 {
		t.Fatalf("worker=%+v", got[2])
	}
}

func TestAggregateMthreadsPodGpu(t *testing.T) {
	pm, err := os.ReadFile(filepath.Join("testdata", "mthreads-pm.txt"))
	if err != nil {
		t.Fatal(err)
	}
	mem, err := os.ReadFile(filepath.Join("testdata", "mthreads-processes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	procs := mergeMthreadsGpuProcs(parseMthreadsProcessMonitor(string(pm)), parseMthreadsProcessMemory(string(mem)))
	podProcs := map[string]struct{}{
		"1305267": {},
		"1305830": {},
		"1306725": {},
	}
	selected := selectMthreadsGpuProcs(procs, podProcs, []int{0})
	m := &SGuestMonitor{mthreadsGpuMetrics: selected}
	got := m.getPodMthreadsGpuMetrics()
	if len(got) != 1 {
		t.Fatalf("gpus=%d, want 1", len(got))
	}
	if got[0].Index != 0 || got[0].Mem != 8830 || got[0].MemUtil != 0.10 || got[0].GpuUtil != 100 {
		t.Fatalf("metric=%+v", got[0])
	}
	fields := got[0].ToMap()
	if fields[MTHREADS_GPU_MEM] != 8830.0 || fields[MTHREADS_GPU_MEM_UTIL] != 0.10 || fields[MTHREADS_GPU_GPU_UTIL] != 100.0 {
		t.Fatalf("fields=%v", fields)
	}
	if got[0].GetUniformName() != "pod_gpu" {
		t.Fatalf("uniform=%s", got[0].GetUniformName())
	}
}

func TestAggregateMthreadsMemUtilCapAndGpuMax(t *testing.T) {
	m := &SGuestMonitor{mthreadsGpuMetrics: []MthreadsGpuProc{
		{Index: 0, Pid: "1", MemMiB: 100, MemUtil: 0.6, GpuUtil: 20},
		{Index: 0, Pid: "2", MemMiB: 50, MemUtil: 0.6, GpuUtil: 90},
	}}
	got := m.getPodMthreadsGpuMetrics()
	if len(got) != 1 || got[0].Mem != 150 || got[0].MemUtil != 1 || got[0].GpuUtil != 90 {
		t.Fatalf("metric=%+v", got)
	}
}
