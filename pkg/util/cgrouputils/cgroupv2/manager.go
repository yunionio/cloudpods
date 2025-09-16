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
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/cgrouputils/cgroup"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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
	return m.cgroupPath
}

func (m *cgroupManager) GetCgroupVersion() string {
	return cgroup.CGROUP_V2
}

func (m *cgroupManager) GetIoScheduler() string {
	return m.ioScheduler
}

func setParam(name, value string, groups ...string) error {
	groupPath := manager.GetCgroupPath()
	if len(groups) > 0 {
		groups = append([]string{groupPath}, groups...)
		groupPath = path.Join(groups...)
	}
	configPath := path.Join(groupPath, name)
	return ioutil.WriteFile(configPath, []byte(value), 0644)
}

func getParam(name, group string) string {
	configPath := path.Join(manager.GetCgroupPath(), group, name)
	param, err := fileutils2.FileGetContents(configPath)
	if err != nil {
		log.Errorf("failed get cgroup config %s: %s", configPath, err)
		return ""
	}
	return strings.TrimSpace(param)
}

func Init(cgroupPath, ioScheduler string) (*cgroupManager, error) {
	if manager != nil {
		return manager, nil
	}
	manager = &cgroupManager{
		cgroupPath:  cgroupPath,
		ioScheduler: ioScheduler,
	}

	for _, module := range []string{"cpu", "cpuset"} {
		err := initSubGroups(module)
		if err != nil {
			manager = nil
			return nil, err
		}
	}

	return manager, nil
}

func initSubGroups(module string, groups ...string) error {
	err := setParam(CGROUP_SUBTREE_CONTROL, fmt.Sprintf("+%s", module), groups...)
	if err != nil {
		return errors.Wrapf(err, "failed add %s to cgroup.subtree_control", module)
	}
	return nil
}

func (m *cgroupManager) RebalanceProcesses(pids []string) {}

func (m *cgroupManager) CgroupDestroy(pid, name string) bool {
	tasks := []cgroup.ICGroupTask{
		&CGroupCPUTask{CgroupTask: NewCGroupBaseTask(pid, name, nil)},
		&CGroupCPUSetTask{CgroupTask: NewCGroupBaseTask(pid, name, nil)},
	}
	for _, t := range tasks {
		t.SetHand(t)
		if !t.RemoveTask() {
			return false
		}
	}
	return true
}

func (m *cgroupManager) CgroupCleanAll(subName string) {
	cgroupPath := manager.GetCgroupPath()
	if subName != "" {
		cgroupPath = path.Join(cgroupPath, subName)
	}
	files, err := ioutil.ReadDir(cgroupPath)
	if err != nil {
		log.Errorf("%s GetTasks failed: %s", cgroupPath, err)
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
		if fileutils2.IsDir(path.Join(cgroupPath, pid)) {
			if !fileutils2.Exists(path.Join("/proc", spid)) {
				log.Infof("Cgroup cleanup %s", path.Join(cgroupPath, pid))
			}
			subFiles, err := ioutil.ReadDir(path.Join(cgroupPath, pid))
			if err != nil {
				log.Errorf("sub dir %s GetTaskIds failed: %s", path.Join(cgroupPath, pid), err)
			} else {
				for _, fi := range subFiles {
					if !fi.IsDir() {
						continue
					}
					if err := os.Remove(path.Join(cgroupPath, pid, fi.Name())); err != nil {
						log.Errorf("CgroupCleanAll pid=%s tid=%s error: %s", pid, fi.Name(), err)
					}
				}
			}
			if err := os.Remove(path.Join(cgroupPath, pid)); err != nil {
				log.Errorf("CgroupCleanAll pid=%s error: %s", pid, err)
			}
		}
	}
}
