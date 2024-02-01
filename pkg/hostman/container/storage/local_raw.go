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

	losetup "github.com/zexi/golosetup"
	"yunion.io/x/log"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

func init() {
	RegisterDriver(newLocalRaw())
}

type localRaw struct{}

func newLocalRaw() *localRaw {
	return &localRaw{}
}

func (l localRaw) GetType() StorageType {
	return STORAGE_TYPE_LOCAL_RAW
}

func (l localRaw) CheckConnect(diskPath string) (string, bool, error) {
	devs, err := losetup.ListDevices()
	if err != nil {
		return "", false, errors.Wrap(err, "list loop devices")
	}
	for _, dev := range devs.LoopDevs {
		if dev.BackFile == diskPath {
			return l.checkPartition(dev.Name), true, nil
		}
	}
	return "", false, nil
}

func (l localRaw) ConnectDisk(diskPath string) (string, error) {
	loDev, err := losetup.AttachDevice(diskPath, true)
	if err != nil {
		return "", errors.Wrapf(err, "failed to attach %s as loop device", diskPath)
	}
	return l.checkPartition(loDev.Name), nil
}

func (l localRaw) checkPartition(devName string) string {
	partPath := fmt.Sprintf("%sp1", devName)
	if fileutils2.Exists(partPath) {
		return partPath
	}
	return devName
}

func (l localRaw) DisconnectDisk(diskPath string, mountPoint string) error {
	devs, err := losetup.ListDevices()
	if err != nil {
		return errors.Wrap(err, "list loop devices")
	}
	for _, dev := range devs.LoopDevs {
		if dev.BackFile == diskPath {
			log.Infof("Start detach loop device %s", dev.Name)
			if err := losetup.DetachDevice(dev.Name); err != nil {
				return errors.Wrapf(err, "detach device %s", dev.Name)
			} else {
				log.Infof("detach loop device %s of disk %s", dev.Name, diskPath)
				return nil
			}
		}
	}
	return nil
}
