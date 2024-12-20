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
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/mountutils"
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
	return mountutils.Mount(devPath, mountPoint, fsType)
}

func Unmount(devPath string) error {
	return mountutils.Unmount(devPath, false)
}

func UnmountWithSubDirs(devPath string) error {
	err := mountutils.Unmount(devPath, false)
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "target is busy") {
		return err
	}
	// found mountpoints starts with devPath
	out, err2 := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("mount | grep ' %s/' | awk '{print $3}'", devPath)).Output()
	if err2 == nil && len(out) != 0 {
		mntPoints := strings.Split(string(out), "\n")
		for _, mntPoint := range mntPoints {
			if mntPoint != "" {
				if err := mountutils.Unmount(mntPoint, false); err != nil {
					return errors.Wrapf(err, "umount subdir %s", mntPoint)
				}
				log.Infof("unmount subdir %q", mntPoint)
			}
		}
	}
	if err := Unmount(devPath); err != nil {
		return errors.Wrapf(err, "unmount %q again", devPath)
	}
	return nil
}
