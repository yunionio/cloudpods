// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain the copy of the License at
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
	defaultIluvatarCorexHome = "/usr/local/corex-4.4.0"
	iluvatarCorexAlias       = "/usr/local/corex"
	iluvatarCtlDevicePath    = "/dev/itrctl"
	iluvatarComputeModeOK    = "Default"
)

type parsedIluvatarGPU struct {
	Index        int
	Name         string
	BusId        string
	MemorySizeMB int
	ComputeMode  string
	UUID         string
}

var (
	ixsmiMemRe  = regexp.MustCompile(`(\d+)\s*MiB\s*/\s*(\d+)\s*MiB`)
	ixsmiListRe = regexp.MustCompile(`(?i)^GPU\s+(\d+):\s+.+\(UUID:\s*([^)]+)\)`)
)

func splitIxsmiCols(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		cols = append(cols, strings.TrimSpace(p))
	}
	return cols
}

func parseIxsmiTable(output string) []*parsedIluvatarGPU {
	var out []*parsedIluvatarGPU
	var cur *parsedIluvatarGPU
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "+") || !strings.HasPrefix(line, "|") {
			continue
		}
		cols := splitIxsmiCols(line)
		if len(cols) < 3 {
			continue
		}
		left := cols[0]
		if strings.HasPrefix(left, "GPU") || strings.HasPrefix(left, "Fan") ||
			strings.HasPrefix(left, "IX-ML") || strings.HasPrefix(left, "Processes") ||
			strings.HasPrefix(left, "No running") {
			continue
		}
		fields := strings.Fields(left)
		if len(fields) >= 2 {
			if idx, err := strconv.Atoi(fields[0]); err == nil {
				name := strings.TrimSpace(strings.TrimPrefix(left, fields[0]))
				cur = &parsedIluvatarGPU{
					Index: idx,
					Name:  name,
					BusId: cols[1],
				}
				continue
			}
		}
		if cur == nil {
			continue
		}
		if m := ixsmiMemRe.FindStringSubmatch(cols[1]); len(m) == 3 {
			cur.MemorySizeMB, _ = strconv.Atoi(m[2])
		}
		modeFields := strings.Fields(cols[len(cols)-1])
		if len(modeFields) > 0 {
			cur.ComputeMode = modeFields[len(modeFields)-1]
		}
		out = append(out, cur)
		cur = nil
	}
	if cur != nil {
		out = append(out, cur)
	}
	return out
}

func parseIxsmiList(output string) map[int]string {
	ret := map[int]string{}
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := ixsmiListRe.FindStringSubmatch(line)
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

func mergeIluvatarProbe(table []*parsedIluvatarGPU, uuids map[int]string) []*parsedIluvatarGPU {
	for _, gpu := range table {
		if gpu == nil {
			continue
		}
		if uuid, ok := uuids[gpu.Index]; ok {
			gpu.UUID = uuid
		}
	}
	return table
}

func iluvatarPCIAddrCandidates(busId string) []string {
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

func iluvatarDevNode(minor int) string {
	return fmt.Sprintf("/dev/iluvatar%d", minor)
}

func resolveIluvatarDeviceMinor(index int, pathExists func(string) bool) int {
	if pathExists != nil && pathExists(iluvatarDevNode(index)) {
		return index
	}
	return -1
}

func iluvatarDeviceSpec(devPath string) *runtimeapi.Device {
	return &runtimeapi.Device{
		ContainerPath: devPath,
		HostPath:      devPath,
		Permissions:   "rwm",
	}
}

func collectIluvatarCommonDevicePaths(pathExists func(string) bool) []string {
	if pathExists != nil && pathExists(iluvatarCtlDevicePath) {
		return []string{iluvatarCtlDevicePath}
	}
	return nil
}

func normalizeIluvatarCorexHome(corexHome string) string {
	if corexHome == "" {
		return defaultIluvatarCorexHome
	}
	return corexHome
}

func buildIluvatarRuntimeEnvs(indices []string, corexHome string) []*runtimeapi.KeyValue {
	corexHome = normalizeIluvatarCorexHome(corexHome)
	return []*runtimeapi.KeyValue{
		{
			Key:   "IX_VISIBLE_DEVICES",
			Value: strings.Join(indices, ","),
		},
		{
			Key:   "COREX_HOME",
			Value: corexHome,
		},
		{
			Key:   "LD_LIBRARY_PATH",
			Value: path.Join(corexHome, "lib64"),
		},
	}
}

func buildIluvatarRuntimeMounts(corexHome string, pathExists func(string) bool) []*runtimeapi.Mount {
	corexHome = normalizeIluvatarCorexHome(corexHome)
	if pathExists == nil || !pathExists(corexHome) {
		return nil
	}
	mounts := []*runtimeapi.Mount{
		{
			ContainerPath: corexHome,
			HostPath:      corexHome,
			Readonly:      true,
		},
	}
	if path.Clean(corexHome) != path.Clean(iluvatarCorexAlias) {
		mounts = append(mounts, &runtimeapi.Mount{
			ContainerPath: iluvatarCorexAlias,
			HostPath:      corexHome,
			Readonly:      true,
		})
	}
	return mounts
}

func buildIluvatarExtraConfigures(indices []string, corexHome string, pathExists func(string) bool) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	if len(indices) == 0 {
		return nil, nil
	}
	return buildIluvatarRuntimeEnvs(indices, corexHome), buildIluvatarRuntimeMounts(corexHome, pathExists)
}
