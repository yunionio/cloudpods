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

package mountutils

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

func mountWrap(mountPoint string, action func() error) error {
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
	return action()
}

func Mount(devPath string, mountPoint string, fsType string) error {
	return mountWrap(mountPoint, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if out, err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx, "mount", "-t", fsType, devPath, mountPoint).Output(); err != nil {
			return errors.Wrapf(err, "mount %s to %s with fs %s: %s", devPath, mountPoint, fsType, string(out))
		}
		return nil
	})
}

func MountOverlay(lowerDir []string, upperDir string, workDir string, mergedDir string) error {
	return mountOverlay(lowerDir, upperDir, workDir, mergedDir, nil)
}

type MountOverlayFeatures struct {
	MetaCopy bool
}

func MountOverlayWithFeatures(lowerDir []string, upperDir string, workDir string, mergedDir string, features *MountOverlayFeatures) error {
	return mountOverlay(lowerDir, upperDir, workDir, mergedDir, features)
}

func mountOverlay(lowerDir []string, upperDir string, workDir string, mergedDir string, features *MountOverlayFeatures) error {
	return mountWrap(mergedDir, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		optStr := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", strings.Join(lowerDir, ":"), upperDir, workDir)
		if features != nil {
			if features.MetaCopy {
				optStr += ",metacopy=on"
			}
		}
		overlayArgs := []string{"-t", "overlay", "overlay", "-o", optStr, mergedDir}
		if out, err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx, "mount", overlayArgs...).Output(); err != nil {
			return errors.Wrapf(err, "mount %v: %s", overlayArgs, out)
		}
		return nil
	})
}

func MountBind(src, target string) error {
	return mountWrap(target, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		bindArgs := []string{"--bind", src, target}
		if out, err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx, "mount", bindArgs...).Output(); err != nil {
			return errors.Wrapf(err, "mount %v: %s", bindArgs, out)
		}
		return nil
	})
}

func Unmount(mountPoint string, useLazy bool) error {
	err := unmount(mountPoint, useLazy)
	errs := []error{}
	if err != nil {
		errs = append(errs, errors.Wrap(err, "umount firstly"))
		if strings.Contains(err.Error(), "target is busy") {
			// use lsof to find process using this mountpoint and kill it
			if err := cleanProcessUseMountPoint(mountPoint); err != nil {
				errs = append(errs, errors.Wrapf(err, "clean process use mountpoint: %s", mountPoint))
			}
			// umount again
			if err := unmount(mountPoint, useLazy); err != nil {
				errs = append(errs, errors.Wrapf(err, "unmount %s after clean process using it", mountPoint))
				return errors.NewAggregate(errs)
			}
			return nil
		} else {
			return errors.NewAggregate(errs)
		}
	}
	return nil
}

func unmount(mountPoint string, useLazy bool) error {
	mountOut, err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", mountPoint).Output()
	if err == nil {
		args := []string{mountPoint}
		if useLazy {
			args = append([]string{"-l"}, args...)
		}
		out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", args...).Output()
		if err != nil {
			//if strings.Contains(err.Error(), fmt.Sprintf("umount: %s", mountPoint)) && strings.Contains(err.Error(), "not mounted") {
			if strings.Contains(string(out), "not mounted") {
				// handle error like: 'umount: /opt/cloud/workspace/servers/2bc8dabf-88c7-448a-8f9e-cbf557acfde2/volumes/6a7ccfcc-a591-4cdd-8d3f-40fde1110870/data/data/com.douban.frodo/: not mounted.'
				return nil
			}
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

func cleanProcessUseMountPoint(mountPoint string) error {
	devs, err := getMountPointDevices(mountPoint)
	if err != nil {
		return errors.Wrapf(err, "get mount point devices: %s", mountPoint)
	}
	errs := []error{}
	for _, dev := range devs {
		pids, err := useLsofFindDevProcess(dev)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "use lsof find device %q process", dev))
		}
		if len(pids) > 0 {
			if err := killProcess(pids); err != nil {
				errs = append(errs, errors.Wrapf(err, "kill process %q", pids))
				return errors.NewAggregate(errs)
			}
		} else {
			if err != nil {
				return errors.NewAggregate(errs)
			}
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
		err = errors.Wrapf(err, "'lsof +f -- %s' failed: %s", dev, out)
		if len(out) == 0 {
			return nil, err
		}
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
	return pids.List(), err
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
		seg1 := parts[0]
		var dev string
		switch seg1 {
		case "sysfs", "proc", "tmpfs", "overlay":
			dev = point
		default:
			dev = seg1
		}

		devs.Insert(dev)
	}
	return devs.List(), nil
}
