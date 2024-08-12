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

package storage

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	drivers = make(map[StorageType]IContainerStorage)
)

type StorageType string

const (
	STORAGE_TYPE_LOCAL_RAW   StorageType = "local_raw"
	STORAGE_TYPE_LOCAL_QCOW2 StorageType = "local_qcow2"
)

type IContainerStorage interface {
	GetType() StorageType
	CheckConnect(diskPath string) (string, bool, error)
	ConnectDisk(diskPath string) (string, error)
	DisconnectDisk(diskPath string, mountPoint string) error
}

func GetDriver(t StorageType) IContainerStorage {
	return drivers[t]
}

func RegisterDriver(drv IContainerStorage) {
	_, ok := drivers[drv.GetType()]
	if ok {
		panic(fmt.Sprintf("driver %s already registered", drv.GetType()))
	}
	drivers[drv.GetType()] = drv
}

func Mount(devPath string, mountPoint string, fsType string) error {
	if !fileutils2.Exists(mountPoint) {
		output, err := procutils.NewCommand("mkdir", "-p", mountPoint).Output()
		if err != nil {
			return errors.Wrapf(err, "mkdir %s failed: %s", mountPoint, output)
		}
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", mountPoint).Run(); err == nil {
		log.Warningf("mountpoint %s is already mounted", mountPoint)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if out, err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx, "mount", "-t", fsType, devPath, mountPoint).Output(); err != nil {
		return errors.Wrapf(err, "mount %s to %s with fs %s: %s", devPath, mountPoint, fsType, string(out))
	}
	return nil
}

func Unmount(mountPoint string) error {
	err := unmount(mountPoint)
	if err != nil {
		if strings.Contains(err.Error(), "target is busy") {
			// use lsof to find process using this mountpoint and kill it
			if err := cleanProcessUseMountPoint(mountPoint); err != nil {
				return errors.Wrapf(err, "clean process use mountpoint: %s", mountPoint)
			}
			// umount again
			if err := unmount(mountPoint); err != nil {
				return errors.Wrapf(err, "unmount %s after clean process using it", mountPoint)
			}
			return nil
		} else {
			return err
		}
	}
	return nil
}

func getMountPointDevices(mountPoint string) ([]string, error) {
	mountFile := "/proc/mounts"
	data, err := os.ReadFile(mountFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", mountFile)
	}
	lines := strings.Split(string(data), "\n")
	devs := sets.NewString()
	for _, line := range lines {
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}
		point := parts[1]
		if point != mountPoint {
			continue
		}
		devs.Insert(parts[0])
	}
	return devs.List(), nil
}

func cleanProcessUseMountPoint(mountPoint string) error {
	devs, err := getMountPointDevices(mountPoint)
	if err != nil {
		return errors.Wrapf(err, "get mount point devices: %s", mountPoint)
	}
	for _, dev := range devs {
		pids, err := useLsofFindDevProcess(dev)
		if err != nil {
			return errors.Wrapf(err, "use lsof find device %q process", dev)
		}
		if err := killProcess(pids); err != nil {
			return errors.Wrapf(err, "kill process: %v", pids)
		}
	}
	return nil
}

func killProcess(pids []int) error {
	for _, pid := range pids {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("kill", "-9", fmt.Sprintf("%d", pid)).Output()
		if err != nil {
			if strings.Contains(err.Error(), "No such process") {
				continue
			}
			return errors.Wrapf(err, "kill -9 %d: %s", pid, out)
		}
	}
	return nil
}

func useLsofFindDevProcess(dev string) ([]int, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("lsof", "+f", "--", dev).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "'lsof +f -- %s' failed", dev)
	}
	pids := sets.NewInt()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "COMMAND") {
			continue
		}
		parts := strings.Split(line, " ")
		newParts := []string{}
		for _, part := range parts {
			if part != "" {
				newParts = append(newParts, part)
			}
		}
		if len(newParts) < 2 {
			continue
		}
		pid, err := strconv.Atoi(newParts[1])
		if err != nil {
			return nil, errors.Wrapf(err, "parse pid: %s", newParts[1])
		}
		log.Infof("find process %q use device %q", line, dev)
		pids.Insert(pid)
	}
	return pids.List(), nil
}

func unmount(mountPoint string) error {
	mountOut, err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", mountPoint).Output()
	if err == nil {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", mountPoint).Output()
		if err != nil {
			return errors.Wrapf(err, "umount %s failed %s", mountPoint, out)
		}
	}
	if strings.Contains(string(mountOut), "No such file or directory") {
		return nil
	}
	if strings.Contains(string(mountOut), "not a mountpoint") {
		return nil
	}
	return errors.Wrapf(err, "check mountpoint %s: %s", mountPoint, string(mountOut))
}
