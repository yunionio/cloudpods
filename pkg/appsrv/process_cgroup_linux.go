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

//go:build linux
// +build linux

package appsrv

import (
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
)

func cgroupMemoryUsageLimit() (used, limit uint64, ok bool) {
	// cgroup v2 (k3s / modern k8s)
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		if lim, limOK := parseCgroupV2MemoryMax(string(data)); limOK {
			curData, err := os.ReadFile("/sys/fs/cgroup/memory.current")
			if err != nil {
				return 0, 0, false
			}
			cur, err := strconv.ParseUint(strings.TrimSpace(string(curData)), 10, 64)
			if err != nil {
				return 0, 0, false
			}
			return cur, lim, true
		}
	}

	// cgroup v1
	limData, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		return 0, 0, false
	}
	lim, err := strconv.ParseUint(strings.TrimSpace(string(limData)), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	var hostTotal uint64
	if vm, err := mem.VirtualMemory(); err == nil {
		hostTotal = vm.Total
	}
	lim, limOK := parseCgroupV1MemoryLimit(lim, hostTotal)
	if !limOK {
		return 0, 0, false
	}
	usageData, err := os.ReadFile("/sys/fs/cgroup/memory/memory.usage_in_bytes")
	if err != nil {
		return 0, 0, false
	}
	usage, err := strconv.ParseUint(strings.TrimSpace(string(usageData)), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return usage, lim, true
}

func cgroupCPULimitCores() float64 {
	// cgroup v2
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		if cores, ok := parseCgroupV2CPUMax(string(data)); ok {
			return cores
		}
		return 0
	}

	// cgroup v1
	quotaData, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	if err != nil {
		return 0
	}
	periodData, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if err != nil {
		return 0
	}
	quota, err1 := strconv.ParseInt(strings.TrimSpace(string(quotaData)), 10, 64)
	period, err2 := strconv.ParseInt(strings.TrimSpace(string(periodData)), 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}
	cores, ok := parseCgroupV1CPUQuota(quota, period)
	if !ok {
		return 0
	}
	return cores
}
