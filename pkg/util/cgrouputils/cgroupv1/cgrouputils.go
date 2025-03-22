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

package cgroupv1

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroup"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CGROUP_TASKS = "tasks"

	maxWeight     = 16
	normalizeBase = 1024
)

type CGroupTask struct {
	pid       string
	threadIds []string
	name      string
	weight    float64

	hand cgroup.ICGroupTask
}

func NewCGroupTask(pid, name string, cpuShares int, threadIds []string) *CGroupTask {
	return &CGroupTask{
		pid:       pid,
		name:      name,
		weight:    float64(cpuShares) / normalizeBase,
		threadIds: threadIds,
	}
}

func (*CGroupTask) Init() bool {
	if !manager.CgroupIsMounted() {
		if !fileutils2.Exists(manager.GetCgroupPath()) {
			if err := procutils.NewCommand("mkdir", "-p", manager.GetCgroupPath()).Run(); err != nil {
				log.Errorf("mkdir -p %s error: %v", manager.GetCgroupPath(), err)
			}
		}
		if err := procutils.NewCommand("mount", "-t", "tmpfs", "-o", "uid=0,gid=0,mode=0755",
			"cgroup", manager.GetCgroupPath()).Run(); err != nil {
			log.Errorf("mount cgroups path %s, error: %v", manager.GetCgroupPath(), err)
			return false
		}
	}
	file, err := os.Open("/proc/cgroups")
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer file.Close()

	re := regexp.MustCompile(`\s+`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// #subsys_name    hierarchy       num_cgroups     enabled
		line := scanner.Text()
		if line[0] != '#' {
			parts := re.Split(line, -1)
			module := parts[0]
			if parts[3] == "0" { // disabled
				continue
			}
			if !manager.ModuleIsMounted(module) {
				moduleDir := path.Join(manager.GetCgroupPath(), module)
				if !fileutils2.Exists(moduleDir) {
					if output, err := procutils.NewCommand("mkdir", moduleDir).Output(); err != nil {
						log.Errorf("mkdir %s failed: %s, %s", moduleDir, err, output)
						return false
					}
				}
				if err := procutils.NewCommand("mount", "-t", "cgroup", "-o",
					module, module, moduleDir).Run(); err != nil {
					log.Errorf("mount cgroup module %s to %s error: %v", module, moduleDir, err)
					return false
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("scan file %s error: %v", file.Name(), err)
		return false
	}

	return true
}

func (c *CGroupTask) CustomConfig(key, value string) bool {
	configPath := GetTaskParamPath(c.hand.Module(), key, c.GroupName())
	if !fileutils2.Exists(configPath) {
		return true
	}
	return SetRootParam(c.hand.Module(), key, value, c.GroupName())
}

func (c *CGroupTask) SetWeight(coreNum int) {
	c.weight = float64(coreNum) / normalizeBase
}

func (c *CGroupTask) SetHand(hand cgroup.ICGroupTask) {
	c.hand = hand
}

func (c *CGroupTask) SetPid(pid string) {
	c.pid = pid
}

func (c *CGroupTask) SetName(name string) {
	c.name = name
}

func (c *CGroupTask) GroupName() string {
	if len(c.name) > 0 {
		return c.name
	}
	return c.pid
}

func (c *CGroupTask) InitTask(hand cgroup.ICGroupTask, coreNum int, pid, name string) {
	c.SetHand(hand)
	c.SetWeight(coreNum)
	c.SetPid(pid)
	c.SetName(name)
}

func (c *CGroupTask) Module() string {
	return ""
}

func (c *CGroupTask) GetWeight() float64 {
	if c.weight < maxWeight {
		return c.weight
	} else {
		return maxWeight
	}
}

func (c *CGroupTask) GetTaskIds() []string {
	if len(c.pid) == 0 {
		return nil
	}

	if len(c.threadIds) > 0 {
		return c.threadIds
	}

	files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%s/task", c.pid))
	if err != nil {
		log.Errorf("GetTaskIds failed: %s", err)
		return nil
	}
	ids := []string{}
	for _, file := range files {
		ids = append(ids, file.Name())
	}
	return ids
}

func (c *CGroupTask) TaskPath() string {
	return path.Join(RootTaskPath(c.hand.Module()), c.GroupName())
}

func (c *CGroupTask) TaskIsExist() bool {
	return fileutils2.Exists(c.TaskPath())
}

func (c *CGroupTask) createTask() bool {
	if err := os.Mkdir(c.TaskPath(), os.ModePerm); err != nil {
		log.Errorln(err)
		return false
	}
	return true
}

func (c *CGroupTask) GetParam(name string) string {
	return GetRootParam(c.hand.Module(), name, c.GroupName())
}

func (c *CGroupTask) MoveTasksToRoot() {
	procs := c.GetParam(CGROUP_TASKS)
	if len(procs) > 0 {
		for _, proc := range strings.Split(procs, "\n") {
			proc = strings.TrimSpace(proc)
			if len(proc) > 0 {
				c.PushPid(proc, true)
			}
		}
	}
}

func (c *CGroupTask) RemoveTask() bool {
	if c.TaskIsExist() {
		c.MoveTasksToRoot()
		log.Infof("Remove task path %s %s", c.TaskPath(), c.name)
		if err := os.Remove(c.TaskPath()); err != nil {
			log.Errorf("Remove task path failed %s", err)
			return false
		}
	}
	return true
}

func (c *CGroupTask) GetStaticConfig() map[string]string {
	return nil
}

func (c *CGroupTask) GetConfig() map[string]string {
	return nil
}

func (c *CGroupTask) SetParam(name, value string) bool {
	return SetRootParam(c.hand.Module(), name, value, c.GroupName())
}

func (c *CGroupTask) SetParams(conf map[string]string) bool {
	for k, v := range conf {
		if !c.SetParam(k, v) {
			log.Errorf("Fail to set %s/%s=%s for %s", c.hand.Module(), k, v, c.GroupName())
			return false
		}
	}
	return true
}

func (c *CGroupTask) Configure() bool {
	if !c.TaskIsExist() {
		if !c.createTask() {
			return false
		}
	}

	conf := c.hand.GetStaticConfig()
	if !c.SetParams(conf) {
		return false
	}

	conf = c.hand.GetConfig()
	return c.SetParams(conf)
}

func (c *CGroupTask) SetTask() bool {
	if !c.TaskIsExist() {
		if !c.createTask() {
			return false
		}
	}

	conf := c.hand.GetStaticConfig()
	if !c.SetParams(conf) {
		return false
	}

	conf = c.hand.GetConfig()
	if c.SetParams(conf) {
		pids := c.GetTaskIds()
		if len(pids) > 0 {
			for _, pid := range pids {
				c.PushPid(pid, false)
			}
			return true
		}
	}
	return false
}

func (c *CGroupTask) PushPid(tid string, isRoot bool) {
	if c.pid == "" {
		return
	}

	subdir := fmt.Sprintf("/proc/%s/task/%s", c.pid, tid)
	if fi, err := os.Stat(subdir); err != nil {
		log.Errorf("Fail to put pid in task %s", err)
		return
	} else if fi.Mode().IsDir() {
		stat, err := fileutils2.FileGetContents(path.Join(subdir, "stat"))
		if err != nil {
			log.Errorf("Fail to put pid in task %s", err)
			return
		}
		re := regexp.MustCompile(`\s+`)
		data := re.Split(stat, -1)
		if data[2] != "Z" {
			if isRoot {
				SetRootParam(c.hand.Module(), CGROUP_TASKS, tid, "")
			} else {
				c.SetParam(CGROUP_TASKS, tid)
			}
		}
	}
}

/**
 *  CGroupCPUTask
 */

type CGroupCPUTask struct {
	*CGroupTask
}

const (
	CgroupsSharesWeight = 1024
	CPU_SHARES          = "cpu.shares"
)

func (c *CGroupCPUTask) Module() string {
	return "cpu"
}

func (c *CGroupCPUTask) GetConfig() map[string]string {
	wt := int(CgroupsSharesWeight * c.GetWeight())
	return map[string]string{CPU_SHARES: fmt.Sprintf("%d", wt)}
}

func (c *CGroupCPUTask) Init() bool {
	return SetRootParam(c.Module(), CPU_SHARES,
		fmt.Sprintf("%d", CgroupsSharesWeight), "")
}

func (m *cgroupManager) NewCGroupCPUTask(pid, name string, cpuShares int) cgroup.ICGroupTask {
	t := &CGroupCPUTask{NewCGroupTask(pid, name, cpuShares, nil)}
	t.SetHand(t)
	return t
}

/**
 *  CGroupIOTask
 */

type CGroupIOTask struct {
	*CGroupTask
}

const (
	IoWeightBase    = 100
	IoWeightMax     = 1000
	IoWeightMin     = 100
	BLOCK_IO_WEIGHT = "blkio.weight"

	BLOCK_IO_BFQ_WEIGHT = "blkio.bfq.weight"

	IOSCHED_CFQ = "cfq"
	IOSCHED_BFQ = "bfq"
)

func (c *CGroupIOTask) Module() string {
	return "blkio"
}

func (c *CGroupIOTask) GetConfig() map[string]string {
	wt := int(c.GetWeight() * IoWeightBase)
	if wt > IoWeightMax {
		wt = IoWeightMax
	} else if wt < IoWeightMin {
		wt = IoWeightMin
	}
	switch manager.GetIoScheduler() {
	case IOSCHED_CFQ:
		return map[string]string{BLOCK_IO_WEIGHT: fmt.Sprintf("%d", wt)}
	case IOSCHED_BFQ:
		return map[string]string{BLOCK_IO_BFQ_WEIGHT: fmt.Sprintf("%d", wt)}
	default:
		return nil
	}
}

func (c *CGroupIOTask) Init() bool {
	switch manager.GetIoScheduler() {
	case IOSCHED_CFQ:
		return SetRootParam(c.Module(), BLOCK_IO_WEIGHT, fmt.Sprintf("%d", IoWeightMax), "")
	default:
		return true
	}
}

func (m *cgroupManager) NewCGroupIOTask(pid, name string, cpuShares int) cgroup.ICGroupTask {
	task := &CGroupIOTask{NewCGroupTask(pid, name, cpuShares, nil)}
	task.SetHand(task)
	return task
}

/**
 *  CGroupIOHardlimitTask
 */

type CGroupIOHardlimitTask struct {
	*CGroupIOTask

	cpuNum int
	params map[string]int
	devId  string
}

func (c *CGroupIOHardlimitTask) GetConfig() map[string]string {
	config := make(map[string]string, 0)
	for k, v := range c.params {
		if v != 0 {
			config[k] = fmt.Sprintf("%s %d", c.devId, v*c.cpuNum)
		}
	}
	return config
}

func (m *cgroupManager) NewCGroupIOHardlimitTask(pid, name string, coreNum int, params map[string]int, devId string) cgroup.ICGroupTask {
	task := &CGroupIOHardlimitTask{
		CGroupIOTask: m.NewCGroupIOTask(pid, name, 0).(*CGroupIOTask),
		cpuNum:       coreNum,
		params:       params,
		devId:        devId,
	}
	task.SetHand(task)
	return task
}

/**
 *  CGroupMemoryTask
 */

type CGroupMemoryTask struct {
	*CGroupTask
}

const (
	root_swappiness   = 60
	vm_swappiness     = 0
	MEMORY_SWAPPINESS = "memory.swappiness"
)

func (c *CGroupMemoryTask) Module() string {
	return "memory"
}

func (c *CGroupMemoryTask) GetConfig() map[string]string {
	return map[string]string{MEMORY_SWAPPINESS: fmt.Sprintf("%d", vm_swappiness)}
}

func (m *cgroupManager) NewCGroupMemoryTask(pid, name string, coreNum int) cgroup.ICGroupTask {
	task := &CGroupMemoryTask{
		CGroupTask: NewCGroupTask(pid, name, coreNum, nil),
	}
	task.SetHand(task)
	return task
}

/**
 *  CGroupCPUSetTask
 */

type CGroupCPUSetTask struct {
	*CGroupTask

	cpuset string
	mems   string
}

const (
	CPUSET_CPUS = "cpuset.cpus"
	CPUSET_MEMS = "cpuset.mems"
)

func (c *CGroupCPUSetTask) Module() string {
	return "cpuset"
}

func (c *CGroupCPUSetTask) GetConfig() map[string]string {
	if c.cpuset == "" {
		parentPath := filepath.Dir(c.GroupName())
		c.cpuset = GetRootParam(c.Module(), CPUSET_CPUS, parentPath)
	}
	if c.mems == "" {
		parentPath := filepath.Dir(c.GroupName())
		c.mems = GetRootParam(c.Module(), CPUSET_MEMS, parentPath)
	}
	return map[string]string{CPUSET_CPUS: c.cpuset, CPUSET_MEMS: c.mems}
}

func (m *cgroupManager) NewCGroupCPUSetTask(pid, name, cpuset, mems string) cgroup.ICGroupTask {
	task := &CGroupCPUSetTask{
		CGroupTask: NewCGroupTask(pid, name, 0, nil),
		cpuset:     cpuset,
		mems:       mems,
	}
	task.SetHand(task)
	return task
}

func (m *cgroupManager) NewCGroupSubCPUSetTask(pid, name string, cpuset string, threadIds []string) cgroup.ICGroupTask {
	task := &CGroupCPUSetTask{
		CGroupTask: NewCGroupTask(pid, name, 0, threadIds),
		cpuset:     cpuset,
	}
	task.SetHand(task)
	return task
}
