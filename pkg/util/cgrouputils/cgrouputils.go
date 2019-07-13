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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CGROUP_PATH_UBUNTU = "/sys/fs/cgroup"
	CGROUP_PATH_CENTOS = "/cgroup"
	CGROUP_TASKS       = "tasks"

	maxWeight     = 16
	normalizeBase = 1024
)

var (
	cgroupsPath = getGroupPath()
)

type ICGroupTask interface {
	InitTask(hand ICGroupTask, coreNum int, pid string)
	SetPid(string)
	SetWeight(coreNum int)
	SetHand(hand ICGroupTask)

	GetStaticConfig() map[string]string
	GetConfig() map[string]string
	Module() string
	RemoveTask() bool
	SetTask() bool

	init() bool
}

type CGroupTask struct {
	pid    string
	weight float64

	hand ICGroupTask
}

func NewCGroupTask(pid string, coreNum int) *CGroupTask {
	return &CGroupTask{
		pid:    pid,
		weight: float64(coreNum) / normalizeBase,
	}
}

func getGroupPath() string {
	if fileutils2.Exists(CGROUP_PATH_UBUNTU) {
		return CGROUP_PATH_UBUNTU
	} else {
		return CGROUP_PATH_CENTOS
	}
}

func CgroupIsMounted() bool {
	return procutils.NewCommand("mountpoint", cgroupsPath).Run() == nil
}

func ModuleIsMounted(module string) bool {
	fullPath := path.Join(cgroupsPath, module)
	if fi, err := os.Lstat(fullPath); err != nil {
		log.Errorln(err)
		return false
	} else if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		// is link
		fullPath, err = filepath.EvalSymlinks(fullPath)
		if err != nil {
			log.Errorln(err)
		}
	}
	return procutils.NewCommand("mountpoint", fullPath).Run() == nil
}

func RootTaskPath(module string) string {
	return path.Join(cgroupsPath, module)
}

func GetTaskParamPath(module, name, pid string) string {
	spath := RootTaskPath(module)
	if len(pid) > 0 {
		spath = path.Join(spath, pid)
	}
	return path.Join(spath, name)
}

func GetRootParam(module, name, pid string) string {
	param, err := fileutils2.FileGetContents(GetTaskParamPath(module, name, pid))
	if err != nil {
		log.Errorln(err)
		return ""
	}
	return strings.TrimSpace(param)
}

func SetRootParam(module, name, value, pid string) bool {
	param := GetRootParam(module, name, pid)
	if param != value {
		err := ioutil.WriteFile(GetTaskParamPath(module, name, pid), []byte(value), 0644)
		if err != nil {
			if len(pid) == 0 {
				pid = "root"
			}
			log.Errorf("fail to set %s to %s(%s): %s", name, value, pid, err)
			return false
		}
	}
	return true
}

func CleanupNonexistPids(module string) {
	var root = RootTaskPath(module)
	files, err := ioutil.ReadDir(root)
	if err != nil {
		log.Errorf("GetTaskIds failed: %s", err)
		return
	}
	ids := []string{}
	for _, file := range files {
		ids = append(ids, file.Name())
	}
	re := regexp.MustCompile(`^\d+$`)
	for _, pid := range ids {
		if re.MatchString(pid) && fileutils2.IsDir(path.Join(root, pid)) {
			if !fileutils2.Exists(path.Join("/proc", pid)) {
				log.Infof("Cgroup clenup %s", pid)
				if err := os.Remove(path.Join(root, pid)); err != nil {
					log.Errorf("CleanupNonexistPids pid=%s error: %s", pid, err)
				}
			}
		}
	}
}

func (c *CGroupTask) SetWeight(coreNum int) {
	c.weight = float64(coreNum) / normalizeBase
}

func (c *CGroupTask) SetHand(hand ICGroupTask) {
	c.hand = hand
}

func (c *CGroupTask) SetPid(pid string) {
	c.pid = pid
}

func (c *CGroupTask) InitTask(hand ICGroupTask, coreNum int, pid string) {
	c.SetHand(hand)
	c.SetWeight(coreNum)
	c.SetPid(pid)
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
	return path.Join(RootTaskPath(c.hand.Module()), c.pid)
}

func (c *CGroupTask) taskIsExist() bool {
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
	return GetRootParam(c.hand.Module(), name, c.pid)
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
	if c.taskIsExist() {
		c.MoveTasksToRoot()
		log.Infof("Remove task path %s", c.TaskPath())
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
	return SetRootParam(c.hand.Module(), name, value, c.pid)
}

func (c *CGroupTask) SetParams(conf map[string]string) bool {
	for k, v := range conf {
		if !c.SetParam(k, v) {
			log.Errorf("Fail to set %s/%s=%s for %s", c.hand.Module(), k, v, c.pid)
			return false
		}
	}
	return true
}

func (c *CGroupTask) SetTask() bool {
	if !c.taskIsExist() {
		if !c.createTask() {
			return false
		}
	}
	conf := c.hand.GetStaticConfig()
	if c.SetParams(conf) {
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
	}
	return false
}

func (c *CGroupTask) PushPid(pid string, isRoot bool) {
	subdir := fmt.Sprintf("/proc/%s/task/%s", c.pid, pid)
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
				SetRootParam(c.hand.Module(), CGROUP_TASKS, pid, "")
			} else {
				c.SetParam(CGROUP_TASKS, pid)
			}
		}
	}
}

func (c *CGroupTask) init() bool {
	if !CgroupIsMounted() {
		if !fileutils2.Exists(cgroupsPath) {
			if err := procutils.NewCommand("mkdir", "-p", cgroupsPath).Run(); err != nil {
				log.Errorln(err)
			}
		}
		if err := procutils.NewCommand("mount", "-t", "tmpfs", "-o", "uid=0,gid=0,mode=0755",
			"cgroup", cgroupsPath).Run(); err != nil {
			log.Errorln(err)
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
		line := scanner.Text()
		if line[0] != '#' {
			parts := re.Split(line, -1)
			module := parts[0]
			if !ModuleIsMounted(module) {
				moduleDir := path.Join(cgroupsPath, module)
				if !fileutils2.Exists(moduleDir) {
					if _, err := procutils.NewCommand("mkdir", moduleDir).Output(); err != nil {
						log.Errorln(err)
						return false
					}
				}
				if err := procutils.NewCommand("mount", "-t", "cgroup", "-o",
					module, module, moduleDir).Run(); err != nil {
					log.Errorln(err)
					return false
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorln(err)
		return false
	}

	return true
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

func (c *CGroupCPUTask) init() bool {
	return SetRootParam(c.Module(), CPU_SHARES,
		fmt.Sprintf("%d", CgroupsSharesWeight), "")
}

func NewCGroupCPUTask(pid string, coreNum int) CGroupCPUTask {
	cgroup := CGroupCPUTask{NewCGroupTask(pid, coreNum)}
	cgroup.hand = &cgroup
	return cgroup
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
	return map[string]string{BLOCK_IO_WEIGHT: fmt.Sprintf("%d", wt)}
}

func (c *CGroupIOTask) init() bool {
	return SetRootParam(c.Module(), BLOCK_IO_WEIGHT, fmt.Sprintf("%d", IoWeightMax), "")
}

func NewCGroupIOTask(pid string, coreNum int) *CGroupIOTask {
	task := &CGroupIOTask{NewCGroupTask(pid, coreNum)}
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

func NewCGroupIOHardlimitTask(pid string, mem int, params map[string]int, devId string) *CGroupIOHardlimitTask {
	task := &CGroupIOHardlimitTask{
		CGroupIOTask: NewCGroupIOTask(pid, 0),
		cpuNum:       mem,
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

func NewCGroupMemoryTask(pid string, coreNum int) *CGroupMemoryTask {
	task := &CGroupMemoryTask{
		CGroupTask: NewCGroupTask(pid, coreNum),
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
}

const (
	CPUSET_CPUS = "cpuset.cpus"
	CPUSET_MEMS = "cpuset.mems"
)

func (c *CGroupCPUSetTask) Module() string {
	return "cpuset"
}

func (c *CGroupCPUSetTask) GetStaticConfig() map[string]string {
	return map[string]string{CPUSET_MEMS: GetRootParam(c.Module(), CPUSET_MEMS, "")}
}

func (c *CGroupCPUSetTask) GetConfig() map[string]string {
	return map[string]string{CPUSET_CPUS: c.cpuset}
}

func NewCGroupCPUSetTask(pid string, coreNum int, cpuset string) CGroupCPUSetTask {
	task := CGroupCPUSetTask{
		CGroupTask: NewCGroupTask(pid, coreNum),
		cpuset:     cpuset,
	}
	task.SetHand(&task)
	return task
}

func Init() bool {
	for _, hand := range []ICGroupTask{&CGroupTask{}, &CGroupCPUTask{}, &CGroupIOTask{}} {
		if !hand.init() {
			return false
		}
	}
	return true
}

func CgroupSet(pid string, coreNum int) bool {
	tasks := []ICGroupTask{
		&CGroupCPUTask{&CGroupTask{}},
		&CGroupIOTask{&CGroupTask{}},
		&CGroupMemoryTask{&CGroupTask{}},
	}
	for _, hand := range tasks {
		hand.InitTask(hand, coreNum, pid)
		if !hand.SetTask() {
			return false
		}
	}
	return true
}

func CgroupIoHardlimitSet(
	pid string, coreNum int,
	params map[string]int, devId string,
) bool {
	cg := NewCGroupIOHardlimitTask(pid, coreNum, params, devId)
	return cg.SetTask()
}

func CgroupDestroy(pid string) bool {
	tasks := []ICGroupTask{
		&CGroupCPUTask{&CGroupTask{}},
		&CGroupIOTask{&CGroupTask{}},
		&CGroupMemoryTask{&CGroupTask{}},
		&CGroupCPUSetTask{&CGroupTask{}, ""},
		&CGroupIOHardlimitTask{CGroupIOTask: &CGroupIOTask{&CGroupTask{}}},
	}
	for _, hand := range tasks {
		hand.InitTask(hand, 0, pid)
		if !hand.RemoveTask() {
			return false
		}
	}
	return true
}

func CgroupCleanAll() {
	tasks := []ICGroupTask{
		&CGroupCPUTask{&CGroupTask{}},
		&CGroupIOTask{&CGroupTask{}},
		&CGroupMemoryTask{&CGroupTask{}},
		&CGroupCPUSetTask{CGroupTask: &CGroupTask{}},
		&CGroupIOHardlimitTask{CGroupIOTask: &CGroupIOTask{&CGroupTask{}}},
	}
	for _, hand := range tasks {
		hand.SetHand(hand)
		CleanupNonexistPids(hand.Module())
	}
}
