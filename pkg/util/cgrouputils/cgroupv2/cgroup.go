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

package cgroupv2

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroup"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	CGROUP_SUBTREE_CONTROL = "cgroup.subtree_control"

	CGROUP_PROCS         = "cgroup.procs"
	CGROUP_THREADS       = "cgroup.threads"
	CGROUP_TYPE          = "cgroup.type"
	CGROUP_TYPE_THREADED = "threaded"

	CPU_WEIGHT = "cpu.weight"
)

type CgroupTask struct {
	pid       string
	threadIds []string
	name      string

	hand cgroup.ICGroupTask
}

func NewCGroupBaseTask(pid, name string, threadIds []string) *CgroupTask {
	return &CgroupTask{
		pid:       pid,
		name:      name,
		threadIds: threadIds,
	}
}

func (c *CgroupTask) InitTask(hand cgroup.ICGroupTask, cpuShares int, pid, name string) {}

func (c *CgroupTask) SetPid(string) {}

func (c *CgroupTask) SetName(string) {}

func (c *CgroupTask) SetWeight(coreNum int) {}

func (c *CgroupTask) Init() bool {
	//initSubGroups()
	return true
}

func (c *CgroupTask) Module() string {
	return ""
}

func (c *CgroupTask) SetHand(hand cgroup.ICGroupTask) {
	c.hand = hand
}

func (c *CgroupTask) GetStaticConfig() map[string]string {
	return nil
}

func (c *CgroupTask) GetConfig() map[string]string {
	return nil
}

func (c *CgroupTask) CustomConfig(key, value string) bool {
	configPath := path.Join(manager.GetCgroupPath(), c.GroupName(), key)
	if !fileutils2.Exists(configPath) {
		return true
	}
	return c.SetParam(key, value)
}

func (c *CgroupTask) GetParentGroup() string {
	return filepath.Dir(c.GroupName())
}

func (c *CgroupTask) GetParam(name string) string {
	return getParam(name, c.GroupName())
}

func (c *CgroupTask) RemoveTask() bool {
	if c.TaskIsExist() {
		if c.GetParam(CGROUP_TYPE) == CGROUP_TYPE_THREADED {
			// move threads to parents
			threads := c.GetParam(CGROUP_THREADS)
			if len(threads) > 0 {
				parentGroup := c.GetParentGroup()
				for _, thread := range strings.Split(threads, "\n") {
					thread = strings.TrimSpace(thread)
					if len(thread) > 0 {
						err := setParam(CGROUP_THREADS, thread, parentGroup)
						if err != nil {
							log.Errorf("failed remove thread %s of %s: %s", thread, c.GroupName(), err)
						}
					}
				}
			}
		} else {
			// move procs to root
			procs := c.GetParam(CGROUP_PROCS)
			if len(procs) > 0 {
				for _, proc := range strings.Split(procs, "\n") {
					proc = strings.TrimSpace(proc)
					if len(proc) > 0 {
						err := setParam(CGROUP_PROCS, proc)
						if err != nil {
							log.Errorf("failed remove proc %s of %s: %s", proc, c.GroupName(), err)
						}
					}
				}
			}
		}
		log.Infof("Remove task path %s %s", c.TaskPath(), c.name)
		if err := os.Remove(c.TaskPath()); err != nil {
			log.Errorf("Remove task path failed %s", err)
			return false
		}
	}
	return true
}

func (c *CgroupTask) Configure() bool {
	if !c.ensureTask() {
		return false
	}
	conf := c.hand.GetStaticConfig()
	if !c.SetParams(conf) {
		return false
	}

	conf = c.hand.GetConfig()
	return c.SetParams(conf)
}

func (c *CgroupTask) SetTask() bool {
	if !c.ensureTask() {
		return false
	}
	conf := c.hand.GetStaticConfig()
	if len(conf) > 0 {
		if !c.SetParams(conf) {
			return false
		}
	}
	conf = c.hand.GetConfig()
	if c.SetParams(conf) {
		if len(c.pid) > 0 {
			c.PushPid()
			return true
		}
	}
	return false
}

func (c *CgroupTask) GroupName() string {
	if len(c.name) > 0 {
		return c.name
	}
	return c.pid
}

func (c *CgroupTask) TaskPath() string {
	return path.Join(manager.GetCgroupPath(), c.GroupName())
}

func (c *CgroupTask) TaskIsExist() bool {
	return fileutils2.Exists(c.TaskPath())
}

func (c *CgroupTask) createTask() bool {
	if err := os.Mkdir(c.TaskPath(), os.ModePerm); err != nil {
		log.Errorf("cgroup path create failed %s", err)
		return false
	}
	return true
}

func (c *CgroupTask) ensureTask() bool {
	if !c.TaskIsExist() {
		if !c.createTask() {
			return false
		}
	}
	if len(c.threadIds) > 0 {
		// switch to threaded mode
		if !c.SetParam(CGROUP_TYPE, CGROUP_TYPE_THREADED) {
			return false
		}
	}

	if !c.SetParam(CGROUP_SUBTREE_CONTROL, fmt.Sprintf("+%s", c.hand.Module())) {
		return false
	}
	return true
}

func (c *CgroupTask) SetParam(name, value string) bool {
	err := setParam(name, value, c.GroupName())
	if err != nil {
		log.Errorf("Fail to set %s=%s for %s: %s", name, value, c.GroupName(), err)
		return false
	}
	return true
}

func (c *CgroupTask) SetParams(conf map[string]string) bool {
	for k, v := range conf {
		if !c.SetParam(k, v) {
			return false
		}
	}
	return true
}

func (c CgroupTask) PushPid() {
	if c.pid == "" {
		return
	}
	if len(c.threadIds) > 0 {
		for i := range c.threadIds {
			subdir := fmt.Sprintf("/proc/%s/task/%s", c.pid, c.threadIds[i])
			if fi, err := os.Stat(subdir); err != nil {
				log.Errorf("Fail to stat %s in task %s", subdir, err)
				continue
			} else if fi.Mode().IsDir() {
				stat, err := fileutils2.FileGetContents(path.Join(subdir, "stat"))
				if err != nil {
					log.Errorf("Fail to stat %s in task %s", stat, err)
					continue
				}
				re := regexp.MustCompile(`\s+`)
				data := re.Split(stat, -1)
				if data[2] != "Z" {
					c.SetParam(CGROUP_THREADS, c.threadIds[i])
				}
			}
		}
	} else {
		c.SetParam(CGROUP_PROCS, c.pid)
	}
}

// cgroup cpu.weight

// Convert cgroup v1 cpu.shares value to cgroup v2 cpu.weight
// https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2#phase-1-convert-from-cgroups-v1-settings-to-v2
func CpuSharesToCpuWeight(cpuShares uint64) uint64 {
	return uint64((((cpuShares - 2) * 9999) / 262142) + 1)
}

type CGroupCPUTask struct {
	*CgroupTask

	weight uint64
}

func (c *CGroupCPUTask) Module() string {
	return "cpu"
}

func (c *CGroupCPUTask) GetConfig() map[string]string {
	return map[string]string{CPU_WEIGHT: fmt.Sprintf("%d", c.weight)}
}

func (m *cgroupManager) NewCGroupCPUTask(pid, name string, cpuShares int) cgroup.ICGroupTask {
	task := &CGroupCPUTask{
		CgroupTask: NewCGroupBaseTask(pid, name, nil),
		weight:     CpuSharesToCpuWeight(uint64(cpuShares)),
	}
	task.SetHand(task)
	return task
}

// cgroup cpuset.cpus
const (
	CPUSET_CPUS = "cpuset.cpus"
	CPUSET_MEMS = "cpuset.mems"
)

type CGroupCPUSetTask struct {
	*CgroupTask
	cpuset string
	mems   string
}

func (c *CGroupCPUSetTask) Module() string {
	return "cpuset"
}

func (c *CGroupCPUSetTask) GetConfig() map[string]string {
	if c.cpuset == "" {
		c.cpuset = getParam(CPUSET_CPUS, c.GetParentGroup())
	}
	if c.mems == "" {
		c.mems = getParam(CPUSET_MEMS, c.GetParentGroup())
	}
	config := map[string]string{}
	if c.cpuset != "" {
		config[CPUSET_CPUS] = c.cpuset
	}
	if c.mems != "" {
		config[CPUSET_MEMS] = c.mems
	}
	return config
}

func (m *cgroupManager) NewCGroupCPUSetTask(pid, name, cpuset, mems string) cgroup.ICGroupTask {
	task := &CGroupCPUSetTask{
		CgroupTask: NewCGroupBaseTask(pid, name, nil),
		cpuset:     cpuset,
		mems:       mems,
	}
	task.SetHand(task)
	return task
}

func (m *cgroupManager) NewCGroupSubCPUSetTask(pid, name string, cpuset string, threadIds []string) cgroup.ICGroupTask {
	task := &CGroupCPUSetTask{
		CgroupTask: NewCGroupBaseTask(pid, name, threadIds),
		cpuset:     cpuset,
	}
	task.SetHand(task)
	return task
}
