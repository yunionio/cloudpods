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
	"testing"
)

func TestBuildNvidiaGpuAssignedQuotasHamiUsesMemoryLimit(t *testing.T) {
	got := buildNvidiaGpuAssignedQuotas(
		[]nvidiaGpuAssignInput{{Id: "dev-hami", MemoryLimit: 8192}},
		map[string]int{"dev-hami": 0},
		map[string]int{"0": 81920},
	)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d, want 1", len(got))
	}
	if got[0].PhysicalIndex != 0 {
		t.Errorf("PhysicalIndex=%d, want 0", got[0].PhysicalIndex)
	}
	if got[0].MemTotal != 8192 {
		t.Errorf("MemTotal=%d, want 8192 (HAMI slice)", got[0].MemTotal)
	}
}

func TestBuildNvidiaGpuAssignedQuotasMpsUsesHostMap(t *testing.T) {
	got := buildNvidiaGpuAssignedQuotas(
		[]nvidiaGpuAssignInput{{Id: "dev-mps", MemoryLimit: 0}},
		map[string]int{"dev-mps": 1},
		map[string]int{"1": 10240},
	)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d, want 1", len(got))
	}
	if got[0].PhysicalIndex != 1 {
		t.Errorf("PhysicalIndex=%d, want 1", got[0].PhysicalIndex)
	}
	if got[0].MemTotal != 10240 {
		t.Errorf("MemTotal=%d, want 10240 (MPS slice from host map)", got[0].MemTotal)
	}
}

func TestBuildNvidiaGpuAssignedQuotasSameGpuSumsMemoryLimit(t *testing.T) {
	got := buildNvidiaGpuAssignedQuotas(
		[]nvidiaGpuAssignInput{
			{Id: "dev-a", MemoryLimit: 4096},
			{Id: "dev-b", MemoryLimit: 4096},
		},
		map[string]int{"dev-a": 0, "dev-b": 0},
		map[string]int{"0": 81920},
	)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d, want 1", len(got))
	}
	if got[0].MemTotal != 8192 {
		t.Errorf("MemTotal=%d, want 8192 (sum of MemoryLimit)", got[0].MemTotal)
	}
}

func TestGetPodNvidiaGpuMetricsIdleAssignedZeroFill(t *testing.T) {
	m := &SGuestMonitor{
		nvidiaGpuAssigned: []nvidiaGpuAssignedQuota{
			{PhysicalIndex: 2, MemTotal: 8192},
		},
		nvidiaGpuIndexMemoryMap: map[string]int{"2": 81920},
	}
	got := m.getPodNvidiaGpuMetrics()
	if len(got) != 1 {
		t.Fatalf("len(got)=%d, want 1", len(got))
	}
	if got[0].PhysicalIndex != 2 {
		t.Errorf("PhysicalIndex=%d, want 2", got[0].PhysicalIndex)
	}
	if got[0].Index != 0 {
		t.Errorf("Index=%d, want 0 (pod-local)", got[0].Index)
	}
	if got[0].MemTotal != 8192 {
		t.Errorf("MemTotal=%d, want 8192", got[0].MemTotal)
	}
	if got[0].Mem != 0 || got[0].MemUtil != 0 {
		t.Errorf("Mem=%d MemUtil=%v, want 0", got[0].Mem, got[0].MemUtil)
	}
}

func TestGetPodNvidiaGpuMetricsProcessPlusIdleAssigned(t *testing.T) {
	m := &SGuestMonitor{
		nvidiaGpuMetrics: []NvidiaGpuProcessMetrics{
			{Index: 0, FB: 1024},
		},
		nvidiaGpuAssigned: []nvidiaGpuAssignedQuota{
			{PhysicalIndex: 0, MemTotal: 8192},
			{PhysicalIndex: 1, MemTotal: 8192},
		},
	}
	got := m.getPodNvidiaGpuMetrics()
	if len(got) != 2 {
		t.Fatalf("len(got)=%d, want 2", len(got))
	}
	if got[0].PhysicalIndex != 0 || got[0].Index != 0 {
		t.Errorf("gpu0 PhysicalIndex=%d Index=%d, want 0, 0", got[0].PhysicalIndex, got[0].Index)
	}
	if got[0].Mem != 1024 {
		t.Errorf("gpu0 Mem=%d, want 1024", got[0].Mem)
	}
	if got[0].MemUtil != 1024.0/8192.0 {
		t.Errorf("gpu0 MemUtil=%v, want %v", got[0].MemUtil, 1024.0/8192.0)
	}
	if got[1].PhysicalIndex != 1 || got[1].Index != 1 {
		t.Errorf("gpu1 PhysicalIndex=%d Index=%d, want 1, 1", got[1].PhysicalIndex, got[1].Index)
	}
	if got[1].Mem != 0 || got[1].MemUtil != 0 {
		t.Errorf("gpu1 Mem=%d MemUtil=%v, want 0", got[1].Mem, got[1].MemUtil)
	}
}

func TestGetPodNvidiaGpuMetricsHamiSliceUtil(t *testing.T) {
	m := &SGuestMonitor{
		nvidiaGpuMetrics: []NvidiaGpuProcessMetrics{
			{Index: 0, FB: 8192},
		},
		nvidiaGpuAssigned: []nvidiaGpuAssignedQuota{
			{PhysicalIndex: 0, MemTotal: 8192},
		},
		nvidiaGpuIndexMemoryMap: map[string]int{"0": 81920},
	}
	got := m.getPodNvidiaGpuMetrics()
	if len(got) != 1 {
		t.Fatalf("len(got)=%d, want 1", len(got))
	}
	if got[0].MemTotal != 8192 {
		t.Errorf("MemTotal=%d, want 8192 (HAMI slice, not full card)", got[0].MemTotal)
	}
	if got[0].Mem != 8192 {
		t.Errorf("Mem=%d, want 8192", got[0].Mem)
	}
	if got[0].MemUtil != 1 {
		t.Errorf("MemUtil=%v, want 1", got[0].MemUtil)
	}
	if got[0].PhysicalIndex != 0 {
		t.Errorf("PhysicalIndex=%d, want 0", got[0].PhysicalIndex)
	}
}

func TestGetPodNvidiaGpuMetricsNilWithoutAssignedOrProcess(t *testing.T) {
	m := &SGuestMonitor{}
	if got := m.getPodNvidiaGpuMetrics(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestPodNvidiaGpuMetricsToMapIncludesMem(t *testing.T) {
	m := PodNvidiaGpuMetrics{Mem: 512, MemUtil: 0.25, MemTotal: 2048}
	got := m.ToMap()
	if got[NVIDIA_GPU_MEM] != 512 {
		t.Errorf("mem=%v, want 512", got[NVIDIA_GPU_MEM])
	}
	if got[NVIDIA_GPU_MEM_UTIL] != 0.25 {
		t.Errorf("mem_util=%v, want 0.25", got[NVIDIA_GPU_MEM_UTIL])
	}
}
