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

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	defaultTHeadPpuSdkHome = "/usr/local/PPU_SDK"
	defaultTHeadPpuSmiPath = "/usr/local/bin/ppu-smi"
	tHeadPpuDevicePrefix   = "/dev/alixpu_ppu"
)

var tHeadPpuCommonDevicePaths = []string{
	"/dev/alixpu",
	"/dev/alixpu_ctl",
	"/dev/alixpu_sep",
}

type parsedTHeadPPU struct {
	Index        int
	Name         string
	BusId        string
	MemorySizeMB int
	UUID         string
}

var (
	tHeadPpuListRe  = regexp.MustCompile(`(?i)^PPU\s+(\d+):\s+.+\(UUID:\s*([^)]+)\)`)
	tHeadPpuMemRe   = regexp.MustCompile(`(\d+)`)
	tHeadPpuNodeRe  = regexp.MustCompile(`^alixpu_ppu(\d+)$`)
	tHeadPpuQueryRe = regexp.MustCompile(`index,name,uuid,pci\.bus_id,memory\.total`)
)

func tHeadPpuDevNode(index int) string {
	return fmt.Sprintf("%s%d", tHeadPpuDevicePrefix, index)
}

func tHeadPpuDeviceSpec(devPath string) *runtimeapi.Device {
	return &runtimeapi.Device{
		ContainerPath: devPath,
		HostPath:      devPath,
		Permissions:   "rwm",
	}
}

func collectTHeadPpuCommonDevicePaths(pathExists func(string) bool) []string {
	if pathExists == nil {
		return nil
	}
	out := make([]string, 0, len(tHeadPpuCommonDevicePaths))
	for _, p := range tHeadPpuCommonDevicePaths {
		if pathExists(p) {
			out = append(out, p)
		}
	}
	return out
}

func normalizeTHeadPpuSdkHome(sdkHome string) string {
	if sdkHome == "" {
		return defaultTHeadPpuSdkHome
	}
	return sdkHome
}

func tHeadPpuLibDir(sdkHome string, pathExists func(string) bool) string {
	sdkHome = normalizeTHeadPpuSdkHome(sdkHome)
	lib64 := path.Join(sdkHome, "lib64")
	if pathExists == nil || pathExists(lib64) {
		return lib64
	}
	return path.Join(sdkHome, "lib")
}

func parsePpuSmiQueryCSV(output string) []*parsedTHeadPPU {
	var out []*parsedTHeadPPU
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if tHeadPpuQueryRe.MatchString(strings.ToLower(strings.ReplaceAll(line, " ", ""))) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "timestamp") {
			continue
		}
		cols := splitCSVLine(line)
		if len(cols) < 5 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(cols[0]))
		if err != nil {
			continue
		}
		mem := 0
		if m := tHeadPpuMemRe.FindStringSubmatch(cols[4]); len(m) == 2 {
			mem, _ = strconv.Atoi(m[1])
		}
		out = append(out, &parsedTHeadPPU{
			Index:        idx,
			Name:         strings.TrimSpace(cols[1]),
			UUID:         strings.TrimSpace(cols[2]),
			BusId:        strings.TrimSpace(cols[3]),
			MemorySizeMB: mem,
		})
	}
	return out
}

func splitCSVLine(line string) []string {
	parts := strings.Split(line, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		cols = append(cols, strings.TrimSpace(p))
	}
	return cols
}

func parsePpuSmiList(output string) map[int]string {
	ret := map[int]string{}
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := tHeadPpuListRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		ret[idx] = strings.TrimSpace(m[2])
	}
	return ret
}

func mergeTHeadPpuProbe(table []*parsedTHeadPPU, uuids map[int]string) []*parsedTHeadPPU {
	for _, gpu := range table {
		if gpu == nil {
			continue
		}
		if gpu.UUID != "" {
			continue
		}
		if uuid, ok := uuids[gpu.Index]; ok {
			gpu.UUID = uuid
		}
	}
	return table
}

func tHeadPpuPCIAddrCandidates(busId string) []string {
	busId = strings.TrimSpace(busId)
	if busId == "" {
		return nil
	}
	cands := make([]string, 0, 3)
	seen := map[string]bool{}
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		cands = append(cands, s)
	}
	add(busId)
	parts := strings.Split(busId, ":")
	if len(parts) == 3 {
		domain, bus, fn := parts[0], parts[1], parts[2]
		if len(domain) > 4 {
			add(domain[len(domain)-4:] + ":" + bus + ":" + fn)
		}
		add(bus + ":" + fn)
	}
	return cands
}

func parseTHeadPpuNodeIndex(name string) (int, bool) {
	m := tHeadPpuNodeRe.FindStringSubmatch(name)
	if len(m) != 2 {
		return 0, false
	}
	idx, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return idx, true
}

func buildTHeadPpuRuntimeEnvs(indices []string, sdkHome string, pathExists func(string) bool) []*runtimeapi.KeyValue {
	sdkHome = normalizeTHeadPpuSdkHome(sdkHome)
	visible := strings.Join(indices, ",")
	return []*runtimeapi.KeyValue{
		{Key: "CUDA_VISIBLE_DEVICES", Value: visible},
		{Key: "NVIDIA_VISIBLE_DEVICES", Value: visible},
		{Key: "LD_LIBRARY_PATH", Value: tHeadPpuLibDir(sdkHome, pathExists)},
	}
}

func buildTHeadPpuRuntimeMounts(sdkHome string, pathExists func(string) bool) []*runtimeapi.Mount {
	sdkHome = normalizeTHeadPpuSdkHome(sdkHome)
	if pathExists == nil || !pathExists(sdkHome) {
		return nil
	}
	return []*runtimeapi.Mount{
		{
			ContainerPath: sdkHome,
			HostPath:      sdkHome,
			Readonly:      true,
		},
	}
}

func buildTHeadPpuExtraConfigures(indices []string, sdkHome string, pathExists func(string) bool) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	if len(indices) == 0 {
		return nil, nil
	}
	return buildTHeadPpuRuntimeEnvs(indices, sdkHome, pathExists), buildTHeadPpuRuntimeMounts(sdkHome, pathExists)
}
