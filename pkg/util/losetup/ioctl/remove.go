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

package ioctl

/*
#include <linux/loop.h>
*/
import "C"
import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/losetup"
	"yunion.io/x/onecloud/pkg/util/mountutils"
)

const (
	LOOP_CTL_PATH = "/dev/loop-control"

	LOOP_CTL_REMOVE = C.LOOP_CTL_REMOVE
)

func RemoveDevice(devNumber int) error {
	fd, err := os.OpenFile(LOOP_CTL_PATH, os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrapf(err, "Open %s", LOOP_CTL_PATH)
	}
	defer fd.Close()

	retNum, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), LOOP_CTL_REMOVE, uintptr(devNumber))
	if err != nil {
		return errors.Wrapf(err, "IOCT REMOVE %d", devNumber)
	}
	log.Infof("using ioctl remove loop %v", retNum)
	return nil
}

func DetachAndRemoveDevice(devPath string) error {
	getDev := func() (*losetup.Device, error) {
		devs, err := losetup.ListDevices()
		if err != nil {
			return nil, errors.Wrapf(err, "list devices")
		}
		dev := devs.GetDeviceByName(devPath)
		return dev, nil
	}
	dev, err := getDev()
	if err != nil {
		return err
	}
	if dev == nil {
		return nil
	}
	// check mountpoints
	partions, err := losetup.GetDevicePartions(devPath)
	if err != nil {
		return errors.Wrapf(err, "get device %s partions", devPath)
	}
	for _, part := range partions {
		checkMntCmd, _ := losetup.NewCommand("sh", "-c", fmt.Sprintf("mount | grep %s | awk '{print $3}'", part)).Run()
		if out := checkMntCmd.Output(); out != "" {
			mntPoints := strings.Split(out, "\n")
			for _, mntPoint := range mntPoints {
				if mntPoint != "" {
					if err := mountutils.Unmount(mntPoint, false); err != nil {
						return errors.Wrapf(err, "umount %s of dev: %s, part: %s", mntPoint, dev.Name, part)
					}
				}
			}
		}
	}

	_, err = losetup.NewLosetupCommand().AddArgs("-d", dev.Name).Run()
	if err != nil {
		return errors.Wrapf(err, "detach device")
	}

	removeDev := func() error {
		log.Infof("Start removing device %s", dev.Name)
		if fileutils.Exists(dev.Name) {
			devNumStr := strings.TrimPrefix(filepath.Base(dev.Name), "loop")
			devNum, err := strconv.Atoi(devNumStr)
			if err != nil {
				return errors.Wrapf(err, "remove device: invalid device number: %s", devNumStr)
			}
			if err := RemoveDevice(devNum); err != nil {
				return errors.Wrapf(err, "remove device: %s", devNumStr)
			}
		} else {
			log.Infof("Loop device %s removed", dev.Name)
		}
		return nil
	}

	errs := []error{}
	for i := 1; i <= 5; i++ {
		if err := removeDev(); err != nil {
			err = errors.Wrapf(err, "remove loop device, %d times", i)
			log.Warningf("remove device %s: %v", devPath, err)
			errs = append(errs, err)
		} else {
			return nil
		}
		time.Sleep(time.Duration(i) * time.Second)
	}
	return errors.NewAggregate(errs)

	// recheck
	/*dev, err = getDev()
	if err != nil {
		return errors.Wrapf(err, "get device by %s for rechecking", devPath)
	}
	if dev != nil {
		return errors.Errorf("device %s still exists, %s", devPath, jsonutils.Marshal(dev))
	}*/
}
