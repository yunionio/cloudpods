package cgrouputils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	CGROUP_PATH_UBUNTU = "/sys/fs/cgroup"
	CGROUP_PATH_CENTOS = "/cgroup"
	CGROUP_TASKS       = "tasks"

	maxWeight     = 16
	normalizeBase = 1024
)

var (
	module      string
	cgroupsPath = getGroupPath()
)

type CGroupTask struct {
	pid    string
	weight float64
}

type ICGroupTask interface {
	GetStaticConfig() map[string]string
	GetConfig() map[string]string
}

func getGroupPath() string {
	if fileutils2.Exists(CGROUP_PATH_UBUNTU) {
		return CGROUP_PATH_UBUNTU
	} else {
		return CGROUP_PATH_CENTOS
	}
}

func CgroupIsMounted() bool {
	return exec.Command("mountpoint", cgroupsPath).Run() == nil
}

func ModuleIsMounted(module string) bool {
	fullPath := path.Join(cgroupsPath, module)
	if fi, err := os.Lstat(fullPath); err != nil {
		log.Errorln(err)
		return false
	} else if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		// is link
		fullPath, err = os.Readlink(fullPath)
		if err != nil {
			log.Errorln(err)
		}
	}
	return exec.Command("mountpoint", fullPath).Run() == nil
}

func RootTaskPath() string {
	return path.Join(cgroupsPath, module)
}

func GetTaskParamPath(name, pid string) string {
	spath := RootTaskPath()
	if len(pid) > 0 {
		spath = path.Join(spath, pid)
	}
	return path.Join(spath, name)
}

func GetRootParam(name, pid string) (string, bool) {
	param, err := fileutils2.FileGetContents(GetTaskParamPath(name, pid))
	if err != nil {
		log.Errorln(err)
		return param, false
	}
	return strings.TrimSpace(param), true
}

func SetRootParam(name, value, pid string) bool {
	if param, succ := GetRootParam(name, pid); !succ {
		return false
	} else if param != value {
		fi, err := os.Open(GetTaskParamPath(name, pid))
		if err == nil {
			fi.Write([]byte(value))
			err = fi.Sync()
		}
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
	return path.Join(RootTaskPath(), c.pid)
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

func (c *CGroupTask) GetStaticConfig() map[string]string {
	return nil
}

func (c *CGroupTask) GetConfig() map[string]string {
	return nil
}

func (c *CGroupTask) SetParam(name, value string) bool {
	return SetRootParam(name, value, c.pid)
}

func (c *CGroupTask) SetParams(conf map[string]string) bool {
	for k, v := range conf {
		if !c.SetParam(k, v) {
			log.Errorf("Fail to set %s/%s=%s for %s", module, k, v, c.pid)
			return false
		}
	}
	return true
}

func (c *CGroupTask) SetTask() bool {
	return c.setTask(c)
}

func (c *CGroupTask) setTask(t ICGroupTask) bool {
	if !c.taskIsExist() {
		if !c.createTask() {
			return false
		}
	}
	conf := t.GetStaticConfig()
	if c.SetParams(conf) {
		conf = t.GetConfig()
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
	// if os.Stat(subdir)
}

func (c *CGroupTask) init() bool {
	if !CgroupIsMounted() {
		if !fileutils2.Exists(cgroupsPath) {
			if err := exec.Command("mkdir", "-p", cgroupsPath).Run(); err != nil {
				log.Errorln(err)
			}
		}
		if err := exec.Command("mount", "-t", "tmpfs", "-o", "uid=0,gid=0,mode=0755",
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
					if err := exec.Command("mkdir", moduleDir).Run(); err != nil {
						log.Errorln(err)
						return false
					}
				}
				if err := exec.Command("mount", "-t", "cgroup", "-o",
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

func Init() {

}
