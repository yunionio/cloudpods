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

package cgrouputils

import (
	"os"
	"testing"
)

func TestCPUUtils(t *testing.T) {
	cpu, err := GetSystemCpu()
	if err != nil {
		t.Errorf("NewCPU error %s", err)
	}

	t.Logf("die list %v", cpu.DieList)
	t.Logf("physical cpu num %d", cpu.GetPhysicalNum())

	cpusetStr := cpu.GetCpuset(0)
	t.Logf("cpu 0 set str %s", cpusetStr)
	physicalNum := cpu.GetPhysicalId(cpusetStr)
	if physicalNum != 0 {
		t.Errorf("failed get physical id %d from cpuset %s", 0, cpusetStr)
	}

	pid := os.Getpid()
	proc, err := NewProcessCPUinfo(pid)
	if err != nil {
		t.Errorf("New process cpu info error %s", err)
	}
	t.Logf("Shared %v %v %f %f", proc.Share, proc.Cpuset, proc.Util, proc.Weight)
}
