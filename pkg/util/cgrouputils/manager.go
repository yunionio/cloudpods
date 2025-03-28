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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroup"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroupv1"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroupv2"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CGROUP_PATH_SYSFS = "/sys/fs/cgroup"
	CGROUP_PATH_ROOT  = "/cgroup"

	CPUSET_SCHED_LOAD_BALANCE = "cpuset.sched_load_balance"
	CPUSET_CLONE_CHILDREN     = "cgroup.clone_children"
)

type ICgroupManager interface {
	RebalanceProcesses(pids []string)
	CgroupCleanAll(subName string)
	CgroupDestroy(pid, name string) bool
	GetCgroupPath() string
	GetSubModulePath(module string) string

	NewCGroupCPUSetTask(pid, name, cpuset, mems string) cgroup.ICGroupTask
	NewCGroupCPUTask(pid, name string, cpuShares int) cgroup.ICGroupTask
	NewCGroupSubCPUSetTask(pid, name string, cpuset string, threadIds []string) cgroup.ICGroupTask
}

func GetCgroupVersion() string {
	return cgroupManager.GetCgroupPath()
}

func RebalanceProcesses(pids []string) {
	cgroupManager.RebalanceProcesses(pids)
}

func CgroupCleanAll(subName string) {
	cgroupManager.CgroupCleanAll(subName)
}

func CgroupDestroy(pid, name string) bool {
	return cgroupManager.CgroupDestroy(pid, name)
}

func GetCgroupPath() string {
	return cgroupManager.GetCgroupPath()
}

func GetSubModulePath(module string) string {
	return cgroupManager.GetSubModulePath(module)
}

func NewCGroupCPUTask(pid, name string, cpuShares int) cgroup.ICGroupTask {
	return cgroupManager.NewCGroupCPUTask(pid, name, cpuShares)
}

func NewCGroupCPUSetTask(pid, name, cpuset, mems string) cgroup.ICGroupTask {
	return cgroupManager.NewCGroupCPUSetTask(pid, name, cpuset, mems)
}

func NewCGroupSubCPUSetTask(pid, name string, cpuset string, threadIds []string) cgroup.ICGroupTask {
	return cgroupManager.NewCGroupSubCPUSetTask(pid, name, cpuset, threadIds)
}

var cgroupManager ICgroupManager

func Init(ioScheduler string) error {
	if cgroupManager != nil {
		return nil
	}

	cgroupPath := ""
	if fileutils2.Exists(CGROUP_PATH_SYSFS) {
		cgroupPath = CGROUP_PATH_SYSFS
	} else if fileutils2.Exists("CGROUP_PATH_ROOT") {
		cgroupPath = CGROUP_PATH_ROOT
	}
	if cgroupPath == "" {
		return errors.Errorf("Can't detect cgroup path")
	}
	output, err := procutils.NewCommand("stat", "-fc", "%T", cgroupPath).Output()
	if err != nil {
		return errors.Wrapf(err, "stat cgroup path %s", cgroupPath)
	}
	cgroupfs := strings.TrimSpace(string(output))
	if cgroupfs == "cgroup2fs" {
		// cgroup v2
		cgroupManager, err = cgroupv2.Init(cgroupPath, ioScheduler)
	} else {
		// cgroup v1
		cgroupManager, err = cgroupv1.Init(cgroupPath, ioScheduler)
	}
	if err != nil {
		return errors.Wrap(err, "init cgroup")
	}
	return nil
}
