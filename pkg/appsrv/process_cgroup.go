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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// processCPUSample keeps the last /process_stats CPU times for interval usage.
var processCPUSample struct {
	mu        sync.Mutex
	lastTotal float64
	lastAt    time.Time
}

// getProcessMemPercent returns memory usage percent relative to the container
// (cgroup) memory limit when one is set; otherwise falls back to host MemTotal
// (gopsutil MemoryPercent).
func getProcessMemPercent(p *process.Process) float32 {
	if used, limit, ok := cgroupMemoryUsageLimit(); ok && limit > 0 {
		return float32(100 * float64(used) / float64(limit))
	}
	pct, _ := p.MemoryPercent()
	return pct
}

// getProcessCPUPercent returns recent CPU usage percent relative to the
// container CPU quota when set; otherwise relative to host GOMAXPROCS/NumCPU.
// First sample after process start returns 0 (no prior interval).
func getProcessCPUPercent(p *process.Process) float64 {
	times, err := p.Times()
	if err != nil {
		return 0
	}
	now := time.Now()
	total := times.Total()

	processCPUSample.mu.Lock()
	defer processCPUSample.mu.Unlock()

	lastTotal := processCPUSample.lastTotal
	lastAt := processCPUSample.lastAt
	processCPUSample.lastTotal = total
	processCPUSample.lastAt = now

	if lastAt.IsZero() {
		return 0
	}
	elapsed := now.Sub(lastAt).Seconds()
	if elapsed <= 0 {
		return 0
	}
	delta := total - lastTotal
	if delta < 0 {
		// process restart / counter reset
		return 0
	}
	coresUsed := delta / elapsed

	limitCores := cgroupCPULimitCores()
	if limitCores <= 0 {
		limitCores = float64(runtime.NumCPU())
		if limitCores <= 0 {
			return 0
		}
	}
	return 100 * coresUsed / limitCores
}

// parseCgroupV2MemoryMax parses memory.max ("max" or byte count).
// ok=false means unlimited / unusable.
func parseCgroupV2MemoryMax(s string) (limit uint64, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "max" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return v, true
}

// parseCgroupV1MemoryLimit treats near-max uint64 values as unlimited.
func parseCgroupV1MemoryLimit(limit uint64, hostTotal uint64) (uint64, bool) {
	if limit == 0 {
		return 0, false
	}
	// cgroup v1 often reports ~2^63-1 / page-aligned max when unlimited
	if limit > 1<<62 {
		return 0, false
	}
	if hostTotal > 0 && limit >= hostTotal {
		return 0, false
	}
	return limit, true
}

// parseCgroupV2CPUMax parses cpu.max ("quota period" or "max period").
// Returns limit in cores; ok=false when unlimited.
func parseCgroupV2CPUMax(s string) (cores float64, ok bool) {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) < 1 {
		return 0, false
	}
	if fields[0] == "max" {
		return 0, false
	}
	quota, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || quota <= 0 {
		return 0, false
	}
	period := int64(100000)
	if len(fields) >= 2 {
		period, err = strconv.ParseInt(fields[1], 10, 64)
		if err != nil || period <= 0 {
			return 0, false
		}
	}
	return float64(quota) / float64(period), true
}

// parseCgroupV1CPUQuota converts cfs_quota_us / cfs_period_us to cores.
// quota < 0 means unlimited.
func parseCgroupV1CPUQuota(quota, period int64) (cores float64, ok bool) {
	if quota <= 0 || period <= 0 {
		return 0, false
	}
	return float64(quota) / float64(period), true
}
