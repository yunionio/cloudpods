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

package pod

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func EnsureDir(dir string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir %s: %s", dir, out)
	}
	return nil
}

func EnsureFile(filename string, content string, mod string) error {
	cmds := []string{
		fmt.Sprintf("echo '%s' > %s", content, filename),
	}
	if mod != "" {
		cmds = append(cmds, fmt.Sprintf("chmod %s %s", mod, filename))
	}
	for _, cmd := range cmds {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
		if err != nil {
			return errors.Wrapf(err, "cmd %s: %s", cmd, out)
		}
	}
	return nil
}

func EnsureContainerSystemCpuDir(cpuDir string, cpuCount int64) error {
	if err := EnsureDir(cpuDir); err != nil {
		return err
	}
	for i := 0; i < int(cpuCount); i++ {
		singleCpuDir := filepath.Join(cpuDir, fmt.Sprintf("cpu%d", i))
		if err := EnsureDir(singleCpuDir); err != nil {
			return errors.Wrapf(err, "create cpu%d", i)
		}
	}
	// create other dirs
	freqDir := filepath.Join(cpuDir, "cpufreq")
	idleDir := filepath.Join(cpuDir, "cpuidle")
	hotplugDir := filepath.Join(cpuDir, "hotplug")
	powerDir := filepath.Join(cpuDir, "power")
	for _, dir := range []string{freqDir, idleDir, hotplugDir, powerDir} {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}
	// create files
	if err := EnsureFile(filepath.Join(cpuDir, "isolated"), "", "755"); err != nil {
		return err
	}
	if err := EnsureFile(filepath.Join(cpuDir, "kernel_max"), fmt.Sprintf("%d", cpuCount), "644"); err != nil {
		return err
	}
	if err := EnsureFile(filepath.Join(cpuDir, "modalias"), "", "755"); err != nil {
		return err
	}
	if err := EnsureFile(filepath.Join(cpuDir, "offline"), "", "755"); err != nil {
		return err
	}
	ensureCpuRangeFile := func(baseName string, mode string) error {
		if mode == "" {
			mode = "644"
		}
		if cpuCount > 1 {
			if err := EnsureFile(filepath.Join(cpuDir, baseName), fmt.Sprintf("0-%d", cpuCount-1), mode); err != nil {
				return err
			}
			return nil
		}
		if err := EnsureFile(filepath.Join(cpuDir, baseName), "0", mode); err != nil {
			return err
		}
		return nil
	}

	if err := ensureCpuRangeFile("online", "755"); err != nil {
		return err
	}
	if err := ensureCpuRangeFile("possible", ""); err != nil {
		return err
	}
	if err := ensureCpuRangeFile("present", ""); err != nil {
		return err
	}
	if err := EnsureFile(filepath.Join(cpuDir, "uevent"), "", "755"); err != nil {
		return err
	}
	return nil
}

type ContainerCPU struct {
	ContainerId string `json:"container_id"`
	Index       int    `json:"index"`
}

func NewContainerCPU(ctrId string, ctrIdx int) *ContainerCPU {
	return &ContainerCPU{
		ContainerId: ctrId,
		Index:       ctrIdx,
	}
}

type HostContainerCPU struct {
	Index      int                        `json:"index"`
	Containers map[string][]*ContainerCPU `json:"containers"`
}

func NewHostContainerCPU(hostIndex int) *HostContainerCPU {
	return &HostContainerCPU{
		Index:      hostIndex,
		Containers: make(map[string][]*ContainerCPU),
	}
}

func (h *HostContainerCPU) HasContainer(ctrId string) bool {
	_, ok := h.Containers[ctrId]
	return ok
}

func (h *HostContainerCPU) DeleteContainer(ctrId string) {
	delete(h.Containers, ctrId)
}

func (h *HostContainerCPU) InsertContainer(ctrId string, ctrIdx int) {
	if h.Containers == nil {
		h.Containers = make(map[string][]*ContainerCPU)
	}
	if h.HasContainer(ctrId) {
		h.Containers[ctrId] = append(h.Containers[ctrId], NewContainerCPU(ctrId, ctrIdx))
	} else {
		h.Containers[ctrId] = []*ContainerCPU{NewContainerCPU(ctrId, ctrIdx)}
	}
}

var (
	hostContainerCPUMapLock = sync.Mutex{}
)

type HostContainerCPUMap struct {
	Map       map[string]*HostContainerCPU `json:"map"`
	stateFile string
}

func NewHostContainerCPUMap(topo *hostapi.HostTopology, stateFile string) (*HostContainerCPUMap, error) {
	ret := make(map[string]*HostContainerCPU)
	if fileutils2.Exists(stateFile) {
		content, err := fileutils2.FileGetContents(stateFile)
		if err != nil {
			return nil, errors.Wrapf(err, "get file contents: %s", stateFile)
		}
		obj, err := jsonutils.ParseString(content)
		if err != nil {
			return nil, errors.Wrapf(err, "parse to json: %s", content)
		}
		hm := new(HostContainerCPUMap)
		if err := obj.Unmarshal(hm); err != nil {
			return nil, errors.Wrap(err, "unmarshal to HostContainerCPUMap")
		}
		hm.stateFile = stateFile
		return hm, nil
	}
	nodes := topo.Nodes
	for _, node := range nodes {
		for _, core := range node.Cores {
			for _, processor := range core.LogicalProcessors {
				ret[fmt.Sprintf("%d", processor)] = NewHostContainerCPU(processor)
			}
		}
	}
	return &HostContainerCPUMap{Map: ret, stateFile: stateFile}, nil
}

func (hm *HostContainerCPUMap) dumpToFile() error {
	return fileutils2.FilePutContents(hm.stateFile, jsonutils.Marshal(hm).PrettyString(), false)
}

func (hm *HostContainerCPUMap) Delete(ctrId string) error {
	hostContainerCPUMapLock.Lock()
	defer hostContainerCPUMapLock.Unlock()

	for _, cm := range hm.Map {
		if cm.HasContainer(ctrId) {
			cm.DeleteContainer(ctrId)
		}
	}
	return hm.dumpToFile()
}

func (hm *HostContainerCPUMap) Get(ctrId string, ctrCpuIndex int) (int, error) {
	hostContainerCPUMapLock.Lock()
	defer hostContainerCPUMapLock.Unlock()

	hostIndex := hm.findLeastUsedIndex(ctrId, ctrCpuIndex)
	if err := hm.markUsed(hostIndex, ctrId, ctrCpuIndex); err != nil {
		return 0, errors.Wrapf(err, "mark container %s %d used", ctrId, ctrCpuIndex)
	}
	return hostIndex, nil
}

func (hm *HostContainerCPUMap) findLeastUsedIndex(ctrId string, ctrCpuIndex int) int {
	unusedMap := make(map[string]*HostContainerCPU)
	usedMap := make(map[string]*HostContainerCPU)
	for idx, hc := range hm.Map {
		tmpHc := hc
		if tmpHc.HasContainer(ctrId) {
			usedMap[idx] = tmpHc
		} else {
			unusedMap[idx] = tmpHc
		}
	}
	if len(unusedMap) != 0 {
		for idx, _ := range unusedMap {
			idxNum, _ := strconv.Atoi(idx)
			return idxNum
		}
	}
	// find the least used index from usedMap
	isStart := true
	leastIndex := 0
	leastUsed := 0
	for idx, hc := range usedMap {
		cm := hc.Containers[ctrId]
		idxNum, _ := strconv.Atoi(idx)
		if isStart {
			leastIndex = idxNum
			leastUsed = len(cm)
			isStart = false
			continue
		}
		if len(cm) < leastUsed {
			leastUsed = len(cm)
			leastIndex = idxNum
		}
	}
	return leastIndex
}

func (hm *HostContainerCPUMap) markUsed(hostIdx int, ctrId string, ctrIdx int) error {
	hc := hm.Map[fmt.Sprintf("%d", hostIdx)]
	hc.InsertContainer(ctrId, ctrIdx)
	return hm.dumpToFile()
}
