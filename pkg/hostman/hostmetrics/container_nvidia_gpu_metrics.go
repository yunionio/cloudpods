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
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type NvidiaGpuProcessMetrics struct {
	Index   int     // Gpu Index
	Pid     string  // Process ID
	Type    string  // Process Type C/G, Compute or Graphics
	FB      int     // Framebuffer Memory Usage
	Ccpm    int     // Current CUDA Contexts Per Measurement
	Sm      float64 // Streaming Multiprocessor Utilization
	SmValid bool    // whether Sm was reported by pmon
	Mem     float64 // Memory Utilization
	Enc     float64 // Encoder Utilization
	Dec     float64 // Decoder Utilization
	Jpg     float64 // JPEG Decoder Utilization
	Ofa     float64 // Other Feature Utilization
	Command string  // Process Command Name
}

func GetNvidiaGpuProcessMetrics() ([]NvidiaGpuProcessMetrics, error) {
	cmd := "nvidia-smi pmon -s mu -c 1"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}
	return parseNvidiaGpuProcessMetrics(string(output)), nil
}

/*
# gpu         pid   type     fb   ccpm     sm    mem    enc    dec    jpg    ofa    command
# Idx           #    C/G     MB     MB      %      %      %      %      %      %    name
*/
func parseNvidiaGpuProcessMetrics(gpuMetricsStr string) []NvidiaGpuProcessMetrics {
	gpuProcessMetrics := make([]NvidiaGpuProcessMetrics, 0)

	lines := strings.Split(gpuMetricsStr, "\n")
	for _, line := range lines {

		// Skip comments and blank lines
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		var processMetrics NvidiaGpuProcessMetrics
		var fb, ccpm, sm, mem, enc, dec, jpg, ofa string
		_, err := fmt.Sscanf(line, "%d %s %s %s %s %s %s %s %s %s %s %s",
			&processMetrics.Index, &processMetrics.Pid, &processMetrics.Type, &fb, &ccpm,
			&sm, &mem, &enc, &dec, &jpg, &ofa, &processMetrics.Command)
		if err != nil {
			log.Debugf("failed parse nvidia gpu metrics %s: %s", line, err)
			continue
		}
		if processMetrics.Command == "nvidia-cuda-mps" || processMetrics.Command == "-" {
			continue
		}
		if fb != "-" {
			val, err := strconv.Atoi(fb)
			if err != nil {
				log.Errorf("failed parse sm %s: %s", sm, err)
			}
			processMetrics.FB = val
		}
		if ccpm != "-" {
			val, err := strconv.Atoi(ccpm)
			if err != nil {
				log.Errorf("failed parse sm %s: %s", sm, err)
			}
			processMetrics.Ccpm = val
		}
		if sm != "-" {
			val, err := strconv.ParseFloat(sm, 64)
			if err != nil {
				log.Errorf("failed parse sm %s: %s", sm, err)
			}
			processMetrics.Sm = val
			processMetrics.SmValid = true
		}
		if mem != "-" {
			val, err := strconv.ParseFloat(mem, 64)
			if err != nil {
				log.Errorf("failed parse mem %s: %s", mem, err)
			}
			processMetrics.Mem = val
		}
		if enc != "-" {
			val, err := strconv.ParseFloat(enc, 64)
			if err != nil {
				log.Errorf("failed parse enc %s: %s", enc, err)
			}
			processMetrics.Enc = val
		}
		if dec != "-" {
			val, err := strconv.ParseFloat(dec, 64)
			if err != nil {
				log.Errorf("failed parse dec %s: %s", dec, err)
			}
			processMetrics.Dec = val
		}
		if jpg != "-" {
			val, err := strconv.ParseFloat(jpg, 64)
			if err != nil {
				log.Errorf("failed parse jpg %s: %s", jpg, err)
			}
			processMetrics.Jpg = val
		}
		if ofa != "-" {
			val, err := strconv.ParseFloat(ofa, 64)
			if err != nil {
				log.Errorf("failed parse ofa %s: %s", ofa, err)
			}
			processMetrics.Ofa = val
		}

		gpuProcessMetrics = append(gpuProcessMetrics, processMetrics)
	}
	return gpuProcessMetrics
}

// GetNvidiaGpuDeviceUtilizations returns the device-level GPU core utilization
// in percent, keyed by GPU index. nvidia-smi pmon only reports per-process
// utilization on a limited set of GPUs, so this is used as a fallback.
func GetNvidiaGpuDeviceUtilizations() (map[int]float64, error) {
	cmd := "nvidia-smi --query-gpu=index,utilization.gpu --format=csv,noheader,nounits"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}
	return parseNvidiaGpuDeviceUtilizations(string(output)), nil
}

func parseNvidiaGpuDeviceUtilizations(output string) map[int]float64 {
	res := make(map[int]float64)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, ",")
		if len(fields) != 2 {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}
		utilStr := strings.TrimSuffix(strings.TrimSpace(fields[1]), "%")
		util, err := strconv.ParseFloat(strings.TrimSpace(utilStr), 64)
		if err != nil {
			continue
		}
		res[index] = util
	}
	return res
}

func (s *SGuestMonitorCollector) collectGpuPodsProcesses() map[string]map[string]struct{} {
	podProcIds := map[string]map[string]struct{}{}
	guestmanager := guestman.GetGuestManager()
	cgroupRoot := path.Join(cgrouputils.GetSubModulePath("cpuset"), guestman.PodCgroupParent())
	guestmanager.Servers.Range(func(k, v interface{}) bool {
		pod, ok := v.(guestman.PodInstance)
		if !ok {
			return true
		}
		if !pod.IsRunning() {
			return true
		}
		podDesc := pod.GetDesc()
		hasGpu := false
		for i := range podDesc.IsolatedDevices {
			if podDesc.IsolatedDevices[i].DevType != compute.GPU_TYPE && podDesc.IsolatedDevices[i].DevType != compute.NPU_TYPE {
				continue
			}
			hasGpu = true
			break
		}
		if !hasGpu {
			return true
		}

		criIds := pod.GetPodContainerCriIds()
		procs := map[string]struct{}{}
		for i := range criIds {
			cgroupPath := path.Join(cgroupRoot, criIds[i], "cgroup.procs")
			pids, err := ReadProccessFromCgroupProcs(cgroupPath)
			if err != nil {
				log.Errorf("collectGpuPodsProcesses: %s", err)
				continue
			}
			for _, pid := range pids {
				procs[pid] = struct{}{}
			}
		}
		if len(procs) > 0 {
			podProcIds[pod.GetId()] = procs
		}
		return true
	})
	return podProcIds
}

func ReadProccessFromCgroupProcs(procFilePath string) ([]string, error) {
	out, err := os.ReadFile(procFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "os.ReadFile")
	}

	pids := strings.Split(string(out), "\n")
	return pids, nil
}
