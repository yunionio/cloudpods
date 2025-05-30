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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroup"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	manager *cgroupManager
)

type cgroupManager struct {
	cgroupPath  string
	ioScheduler string
}

func (m *cgroupManager) GetCgroupPath() string {
	return m.cgroupPath
}

func (m *cgroupManager) GetSubModulePath(module string) string {
	return path.Join(m.cgroupPath, module)
}

func (m *cgroupManager) GetCgroupVersion() string {
	return cgroup.CGROUP_V1
}

func (m *cgroupManager) GetIoScheduler() string {
	return m.ioScheduler
}

func (m *cgroupManager) CgroupIsMounted() bool {
	return procutils.NewCommand("mountpoint", m.cgroupPath).Run() == nil
}

func (m *cgroupManager) ModuleIsMounted(module string) bool {
	fullPath := path.Join(m.cgroupPath, module)
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

func Init(cgroupPath, ioScheduler string) (*cgroupManager, error) {
	if manager != nil {
		return manager, nil
	}
	manager = &cgroupManager{
		cgroupPath:  cgroupPath,
		ioScheduler: ioScheduler,
	}
	for _, hand := range []cgroup.ICGroupTask{
		&CGroupTask{},
		&CGroupCPUTask{},
		//&CGroupIOTask{},
	} {
		if !hand.Init() {
			return manager, errors.Errorf("Cannot initialize %s control group subsystem", hand.Module())
		}
	}

	return manager, nil
}

func RootTaskPath(module string) string {
	return path.Join(manager.cgroupPath, module)
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

// cpuset, task, tid, cgname
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

func (m *cgroupManager) CgroupDestroy(pid, name string) bool {
	tasks := []cgroup.ICGroupTask{
		&CGroupCPUTask{&CGroupTask{}},
		&CGroupIOTask{&CGroupTask{}},
		&CGroupMemoryTask{&CGroupTask{}},
		//&CGroupCPUSetTask{&CGroupTask{}, ""},
		&CGroupIOHardlimitTask{CGroupIOTask: &CGroupIOTask{&CGroupTask{}}},
	}
	for _, hand := range tasks {
		hand.InitTask(hand, 0, pid, name)
		if !hand.RemoveTask() {
			return false
		}
	}
	return true
}

func (m *cgroupManager) CgroupCleanAll(subName string) {
	tasks := []cgroup.ICGroupTask{
		&CGroupCPUTask{&CGroupTask{}},
		&CGroupIOTask{&CGroupTask{}},
		&CGroupMemoryTask{&CGroupTask{}},
		&CGroupCPUSetTask{CGroupTask: &CGroupTask{}},
		&CGroupIOHardlimitTask{CGroupIOTask: &CGroupIOTask{&CGroupTask{}}},
	}
	for _, hand := range tasks {
		hand.SetHand(hand)
		cleanupNonexistPids(hand.Module(), subName)
	}
}

// cleanup
func cleanupNonexistPids(module string, subName string) {
	var root = RootTaskPath(module)
	cleanNonexitPidsWithRoot(root)
	if subName != "" {
		root = path.Join(RootTaskPath(module), subName)
		cleanNonexitPidsWithRoot(root)
	}
}

func cleanNonexitPidsWithRoot(root string) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		log.Errorf("GetTaskIds failed: %s", err)
		return
	}
	ids := []string{}
	for _, file := range files {
		ids = append(ids, file.Name())
	}

	re1 := regexp.MustCompile(`^\d+$`)
	re2 := regexp.MustCompile(`^server_[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}_\d+$`)
	for _, pid := range ids {
		spid := ""
		if re1.MatchString(pid) {
			spid = pid
		} else if re2.MatchString(pid) {
			segs := strings.Split(pid, "_")
			spid = segs[len(segs)-1]
		}

		if fileutils2.IsDir(path.Join(root, pid)) {
			if !fileutils2.Exists(path.Join("/proc", spid)) {
				log.Infof("Cgroup cleanup %s", path.Join(root, pid))

				subFiles, err := ioutil.ReadDir(path.Join(root, pid))
				if err != nil {
					log.Errorf("sub dir %s GetTaskIds failed: %s", path.Join(root, pid), err)
				} else {
					for _, fi := range subFiles {
						if !fi.IsDir() {
							continue
						}
						if err := os.Remove(path.Join(root, pid, fi.Name())); err != nil {
							log.Errorf("CleanupNonexistPids pid=%s tid=%s error: %s", pid, fi.Name(), err)
						}
					}
				}
				if err := os.Remove(path.Join(root, pid)); err != nil {
					log.Errorf("CleanupNonexistPids pid=%s error: %s", pid, err)
				}
			}
		}
	}
}
