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
	defaultKunlunxinXreHome    = "/usr/local/xpu"
	defaultKunlunxinXpuSmiPath = "/usr/local/bin/xpu-smi"
	kunlunxinXpuDevicePrefix   = "/dev/xpu"
)

var kunlunxinXpuCommonDevicePaths = []string{
	"/dev/xpuctrl",
}

type parsedKunlunxinXPU struct {
	Index        int
	Name         string
	BusId        string
	MemorySizeMB int
	UUID         string
}

var (
	// |   0  P800 OAM           N/A   | 00000000:03:00.0 N/A |                    0 |
	kunlunxinXpuBusLineRe = regexp.MustCompile(`(?i)^\|\s*(\d+)\s+(.+?)\s+N/A\s+\|\s*([0-9a-f]+:[0-9a-f]+:[0-9a-f]+\.[0-9a-f]+)`)
	// | N/A   35C  N/A     87W / 400W |      0MiB / 98304MiB |      0%      Default |
	kunlunxinXpuMemUsedTotalRe = regexp.MustCompile(`(?i)(\d+)\s*MiB\s*/\s*(\d+)\s*MiB`)
	kunlunxinXpuNodeRe         = regexp.MustCompile(`^xpu(\d+)$`)
)

func kunlunxinXpuDevNode(index int) string {
	return fmt.Sprintf("%s%d", kunlunxinXpuDevicePrefix, index)
}

func kunlunxinXpuDeviceSpec(devPath string) *runtimeapi.Device {
	return &runtimeapi.Device{
		ContainerPath: devPath,
		HostPath:      devPath,
		Permissions:   "rwm",
	}
}

func collectKunlunxinXpuCommonDevicePaths(pathExists func(string) bool) []string {
	if pathExists == nil {
		return nil
	}
	out := make([]string, 0, len(kunlunxinXpuCommonDevicePaths))
	for _, p := range kunlunxinXpuCommonDevicePaths {
		if pathExists(p) {
			out = append(out, p)
		}
	}
	return out
}

func normalizeKunlunxinXreHome(xreHome string) string {
	if xreHome == "" {
		return defaultKunlunxinXreHome
	}
	return xreHome
}

func kunlunxinXpuLibDir(xreHome string, pathExists func(string) bool) string {
	xreHome = normalizeKunlunxinXreHome(xreHome)
	candidates := []string{
		path.Join(xreHome, "so"),
		path.Join(xreHome, "lib64"),
		path.Join(xreHome, "lib"),
	}
	if pathExists == nil {
		return candidates[0]
	}
	for _, p := range candidates {
		if pathExists(p) {
			return p
		}
	}
	return candidates[0]
}

func parseXpuSmiTable(output string) []*parsedKunlunxinXPU {
	var out []*parsedKunlunxinXPU
	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		m := kunlunxinXpuBusLineRe.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		name := strings.TrimSpace(m[2])
		busId := strings.TrimSpace(m[3])
		mem := 0
		// memory total is on the next data row
		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			memLine := strings.TrimSpace(lines[j])
			mm := kunlunxinXpuMemUsedTotalRe.FindStringSubmatch(memLine)
			if len(mm) == 3 {
				mem, _ = strconv.Atoi(mm[2])
				break
			}
		}
		out = append(out, &parsedKunlunxinXPU{
			Index:        idx,
			Name:         name,
			BusId:        busId,
			MemorySizeMB: mem,
			UUID:         busId,
		})
	}
	return out
}

func kunlunxinXpuPCIAddrCandidates(busId string) []string {
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

func parseKunlunxinXpuNodeIndex(name string) (int, bool) {
	m := kunlunxinXpuNodeRe.FindStringSubmatch(name)
	if len(m) != 2 {
		return 0, false
	}
	idx, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return idx, true
}

func buildKunlunxinXpuRuntimeEnvs(indices []string, xreHome string, pathExists func(string) bool) []*runtimeapi.KeyValue {
	xreHome = normalizeKunlunxinXreHome(xreHome)
	visible := strings.Join(indices, ",")
	return []*runtimeapi.KeyValue{
		{Key: "XPU_VISIBLE_DEVICES", Value: visible},
		{Key: "LD_LIBRARY_PATH", Value: kunlunxinXpuLibDir(xreHome, pathExists)},
	}
}

func buildKunlunxinXpuRuntimeMounts(xreHome string, pathExists func(string) bool) []*runtimeapi.Mount {
	xreHome = normalizeKunlunxinXreHome(xreHome)
	if pathExists == nil || !pathExists(xreHome) {
		return nil
	}
	return []*runtimeapi.Mount{
		{
			ContainerPath: xreHome,
			HostPath:      xreHome,
			Readonly:      true,
		},
	}
}

func buildKunlunxinXpuExtraConfigures(indices []string, xreHome string, pathExists func(string) bool) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	if len(indices) == 0 {
		return nil, nil
	}
	return buildKunlunxinXpuRuntimeEnvs(indices, xreHome, pathExists), buildKunlunxinXpuRuntimeMounts(xreHome, pathExists)
}
