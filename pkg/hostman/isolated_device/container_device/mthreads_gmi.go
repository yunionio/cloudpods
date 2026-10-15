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
	"regexp"
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

const mthreadsDeviceTypePhysical = "Physical"

type parsedMthreadsGPU struct {
	Index        int
	Name         string
	BusId        string
	DeviceType   string
	Perf         string
	GPUUtil      int
	MemoryUsedMB int
	MemorySizeMB int
	TempC        int
	MPCCapable   string
	ECCMode      string
}

var (
	mthreadsIndexRe = regexp.MustCompile(`^(\d+)\s+(.+)$`)
	mthreadsMemRe   = regexp.MustCompile(`(\d+)%\s+(\d+)MiB\((\d+)MiB\)`)
	mthreadsTempRe  = regexp.MustCompile(`(\d+)C`)
)

func splitMthreadsCols(line string) []string {
	parts := strings.Split(line, "|")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		cols = append(cols, strings.TrimSpace(p))
	}
	return cols
}

func parseMthreadsGmiBlock(lines []string) *parsedMthreadsGPU {
	if len(lines) == 0 {
		return nil
	}
	cols := splitMthreadsCols(lines[0])
	if len(cols) < 3 {
		return nil
	}
	head := mthreadsIndexRe.FindStringSubmatch(cols[0])
	if len(head) != 3 {
		return nil
	}
	idx, err := strconv.Atoi(head[1])
	if err != nil {
		return nil
	}
	gpu := &parsedMthreadsGPU{
		Index: idx,
		Name:  strings.TrimSpace(head[2]),
		BusId: cols[1],
	}
	if mem := mthreadsMemRe.FindStringSubmatch(cols[2]); len(mem) == 4 {
		gpu.GPUUtil, _ = strconv.Atoi(mem[1])
		gpu.MemoryUsedMB, _ = strconv.Atoi(mem[2])
		gpu.MemorySizeMB, _ = strconv.Atoi(mem[3])
	}
	if len(lines) >= 2 {
		cols2 := splitMthreadsCols(lines[1])
		if len(cols2) > 0 {
			fields := strings.Fields(cols2[0])
			if len(fields) > 0 {
				gpu.DeviceType = fields[0]
			}
			if len(fields) > 1 {
				gpu.Perf = fields[1]
			}
		}
		if len(cols2) >= 3 {
			if tm := mthreadsTempRe.FindStringSubmatch(cols2[2]); len(tm) == 2 {
				gpu.TempC, _ = strconv.Atoi(tm[1])
			}
			parts := strings.Fields(cols2[2])
			if len(parts) >= 2 {
				gpu.MPCCapable = parts[len(parts)-1]
			}
		}
	}
	if len(lines) >= 3 {
		cols3 := splitMthreadsCols(lines[2])
		for i := len(cols3) - 1; i >= 0; i-- {
			if cols3[i] != "" {
				gpu.ECCMode = cols3[i]
				break
			}
		}
	}
	return gpu
}

func parseMthreadsGmi(output string) []*parsedMthreadsGPU {
	var out []*parsedMthreadsGPU
	var buf []string
	flush := func() {
		if gpu := parseMthreadsGmiBlock(buf); gpu != nil {
			out = append(out, gpu)
		}
		buf = nil
	}
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			flush()
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

func isMthreadsPhysical(gpu *parsedMthreadsGPU) bool {
	return gpu != nil && strings.EqualFold(gpu.DeviceType, mthreadsDeviceTypePhysical)
}

func mthreadsPCIAddrCandidates(busId string) []string {
	return iluvatarPCIAddrCandidates(busId)
}

func mthreadsDevNode(index int) string {
	return fmt.Sprintf("/dev/mtgpu.%d", index)
}

func mthreadsDeviceSpec(devPath string) *runtimeapi.Device {
	return &runtimeapi.Device{
		ContainerPath: devPath,
		HostPath:      devPath,
		Permissions:   "rwm",
	}
}

func buildMthreadsRuntimeEnvs(indices []string) []*runtimeapi.KeyValue {
	value := strings.Join(indices, ",")
	return []*runtimeapi.KeyValue{
		{Key: "MTHREADS_VISIBLE_DEVICES", Value: value},
		{Key: "MTHREADS_DRIVER_CAPABILITIES", Value: "all"},
		// {Key: "MUSA_VISIBLE_DEVICES", Value: value},
		// {Key: "CUDA_VISIBLE_DEVICES", Value: value},
	}
}

func buildMthreadsExtraConfigures(indices []string) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	if len(indices) == 0 {
		return nil, nil
	}
	return buildMthreadsRuntimeEnvs(indices), nil
}

func buildMthreadsHamiQosEnvs(memoryLimitMiB, smUtilLimit int) []*runtimeapi.KeyValue {
	if memoryLimitMiB <= 0 {
		return nil
	}
	return []*runtimeapi.KeyValue{
		{
			Key:   "MTHREADS_QOS_MEMORY_LIMIT",
			Value: strconv.FormatInt(computeapi.MthreadsSgpuMemoryLimitBytes(memoryLimitMiB), 10),
		},
		{
			Key:   "MTHREADS_QOS_COMPUTING_POWER_WEIGHT",
			Value: strconv.Itoa(computeapi.MthreadsSgpuComputingPowerWeight(smUtilLimit)),
		},
	}
}
