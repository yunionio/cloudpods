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
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

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
	return errors.Wrapf(err, "check mountpoint %s: %s", mountPoint, string(mountOut))
}
