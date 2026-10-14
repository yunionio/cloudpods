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
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type MthreadsGpuProc struct {
	Index   int
	Pid     string
	MemMiB  float64
	MemUtil float64 // fraction, 0 to 1
	GpuUtil float64 // percent, 0 to 100
}

func mthreadsGmiBin() string {
	if options.HostOptions.MthreadsGmiPath != "" {
		return options.HostOptions.MthreadsGmiPath
	}
	return "/usr/bin/mthreads-gmi"
}

func GetMthreadsGpuProcessMetrics() ([]MthreadsGpuProc, error) {
	bin := mthreadsGmiBin()
	pmOut, pmErr := procutils.NewRemoteCommandAsFarAsPossible(bin, "-pm").Output()
	memOut, memErr := procutils.NewRemoteCommandAsFarAsPossible(bin).Output()
	if pmErr != nil && memErr != nil {
		return nil, errors.Wrapf(pmErr, "mthreads-gmi -pm (%s); default: %v (%s)", pmOut, memErr, memOut)
	}
	if pmErr != nil {
		log.Errorf("mthreads-gmi -pm: %s %s", pmErr, pmOut)
		pmOut = nil
	}
	if memErr != nil {
		log.Errorf("mthreads-gmi: %s %s", memErr, memOut)
		memOut = nil
	}
	return mergeMthreadsGpuProcs(parseMthreadsProcessMonitor(string(pmOut)), parseMthreadsProcessMemory(string(memOut))), nil
}

func parseMthreadsProcessMonitor(output string) []MthreadsGpuProc {
	var out []MthreadsGpuProc
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 4 {
			continue
		}
		idx, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pid := fields[1]
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		out = append(out, MthreadsGpuProc{
			Index:   idx,
			Pid:     pid,
			MemUtil: parseMthreadsMemPercent(fields[2]),
			GpuUtil: parseMthreadsGpuPercent(fields[3]),
		})
	}
	return out
}

func parseMthreadsProcessMemory(output string) []MthreadsGpuProc {
	var out []MthreadsGpuProc
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		memField := fields[len(fields)-1]
		if !strings.HasSuffix(memField, "MiB") {
			continue
		}
		idx, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pid := fields[1]
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		mem, err := strconv.ParseFloat(strings.TrimSuffix(memField, "MiB"), 64)
		if err != nil {
			continue
		}
		out = append(out, MthreadsGpuProc{
			Index:  idx,
			Pid:    pid,
			MemMiB: mem,
		})
	}
	return out
}

func parseMthreadsMemPercent(raw string) float64 {
	return parseMthreadsPercent(raw) / 100
}

func parseMthreadsGpuPercent(raw string) float64 {
	return parseMthreadsPercent(raw)
}

func parseMthreadsPercent(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "<") {
		return 0
	}
	raw = strings.TrimSuffix(raw, "%")
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func mergeMthreadsGpuProcs(utils, mems []MthreadsGpuProc) []MthreadsGpuProc {
	type key struct {
		index int
		pid   string
	}
	merged := map[key]*MthreadsGpuProc{}
	order := make([]key, 0)
	touch := func(index int, pid string) *MthreadsGpuProc {
		k := key{index, pid}
		if p, ok := merged[k]; ok {
			return p
		}
		p := &MthreadsGpuProc{Index: index, Pid: pid}
		merged[k] = p
		order = append(order, k)
		return p
	}
	for _, p := range utils {
		dst := touch(p.Index, p.Pid)
		dst.MemUtil = p.MemUtil
		dst.GpuUtil = p.GpuUtil
	}
	for _, p := range mems {
		dst := touch(p.Index, p.Pid)
		dst.MemMiB = p.MemMiB
	}
	out := make([]MthreadsGpuProc, 0, len(order))
	for _, k := range order {
		out = append(out, *merged[k])
	}
	return out
}

func selectMthreadsGpuProcs(procs []MthreadsGpuProc, podProcs map[string]struct{}, indexes []int) []MthreadsGpuProc {
	if len(procs) == 0 || len(podProcs) == 0 || len(indexes) == 0 {
		return nil
	}
	want := make(map[int]struct{}, len(indexes))
	for _, idx := range indexes {
		want[idx] = struct{}{}
	}
	out := make([]MthreadsGpuProc, 0)
	for _, p := range procs {
		if _, ok := podProcs[p.Pid]; !ok {
			continue
		}
		if _, ok := want[p.Index]; !ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (m *SGuestMonitor) getPodMthreadsGpuMetrics() []*PodMthreadsGpuMetrics {
	if len(m.mthreadsGpuMetrics) == 0 {
		return nil
	}
	type acc struct {
		mem     float64
		memUtil float64
		gpuUtil float64
	}
	by := map[int]*acc{}
	for _, p := range m.mthreadsGpuMetrics {
		a := by[p.Index]
		if a == nil {
			a = &acc{}
			by[p.Index] = a
		}
		a.mem += p.MemMiB
		a.memUtil += p.MemUtil
		if p.GpuUtil > a.gpuUtil {
			a.gpuUtil = p.GpuUtil
		}
	}
	indexes := make([]int, 0, len(by))
	for idx := range by {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	res := make([]*PodMthreadsGpuMetrics, 0, len(indexes))
	for _, idx := range indexes {
		a := by[idx]
		memUtil := a.memUtil
		if memUtil > 1 {
			memUtil = 1
		}
		gpuUtil := a.gpuUtil
		if gpuUtil > 100 {
			gpuUtil = 100
		}
		res = append(res, &PodMthreadsGpuMetrics{
			Index:   idx,
			Mem:     a.mem,
			MemUtil: memUtil,
			GpuUtil: gpuUtil,
		})
	}
	return res
}

type PodMthreadsGpuMetrics struct {
	PodMetricMeta

	Index   int
	Mem     float64
	MemUtil float64
	GpuUtil float64
}

func (m PodMthreadsGpuMetrics) GetName() string {
	return "pod_mthreads_gpu"
}

func (m PodMthreadsGpuMetrics) GetUniformName() string {
	return "pod_gpu"
}

func (m PodMthreadsGpuMetrics) GetTag() map[string]string {
	return map[string]string{
		"index":        strconv.Itoa(m.Index),
		"dev_type":     compute.GPU_TYPE,
		"sharing_mode": compute.DEVICE_SHARING_MODE_EXCLUSIVE,
	}
}

func (m PodMthreadsGpuMetrics) ToMap() map[string]interface{} {
	return map[string]interface{}{
		MTHREADS_GPU_MEM:      m.Mem,
		MTHREADS_GPU_MEM_UTIL: m.MemUtil,
		MTHREADS_GPU_GPU_UTIL: m.GpuUtil,
	}
}
