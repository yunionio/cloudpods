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

package fsutils

import (
	"path"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"
)

func (d *SFsutilDriver) GetResizeDevBySerial(diskId string) (string, error) {
	out, err := d.Exec("sh", "-c", "lsblk -d -o NAME,SERIAL  | awk 'NR>1'")
	if err != nil {
		return "", errors.Wrapf(err, "ResizePartition lsblk %s", err)
	}
	lines := strings.Split(string(out), "\n")
	diskSerial := strings.ReplaceAll(diskId, "-", "")
	if len(diskSerial) >= 20 {
		diskSerial = diskSerial[:20]
	}
	resizeDev := ""
	for i := range lines {
		segs := strings.Fields(lines[i])
		if len(segs) == 0 {
			continue
		}
		log.Errorf("segs %v", segs)
		if len(segs) == 1 {
			// fetch vpd 80 serial id
			ret, err := d.Exec("sg_inq", "-u", "-p", "0x80", path.Join("/dev/", segs[0]))
			if err != nil {
				log.Infof("failed exec sg_inq: %s %s", ret, err)
				continue
			}
			serialStr := strings.TrimSpace(string(ret))
			serialSegs := strings.Split(serialStr, "=")
			log.Errorf("serial segs %v", serialSegs)
			if len(serialSegs) == 2 && serialSegs[1] == diskSerial {
				resizeDev = path.Join("/dev/", segs[0])
				break
			}
		}
		devName, serial := segs[0], segs[1]
		log.Infof("lsblk segs: %s %s |", devName, serial)
		if strings.HasPrefix(diskSerial, serial) {
			resizeDev = path.Join("/dev/", devName)
			break
		}
	}
	return resizeDev, nil
}

func (d *SFsutilDriver) IsLvmPvDevice(device string) bool {
	return d.Run("pvs", device) == nil
}

func (d *SFsutilDriver) Pvresize(device string) error {
	out, err := d.Exec("partprobe")
	if err != nil {
		return errors.Wrapf(err, "failed resize pv partprobe %s", out)
	}
	out, err = d.Exec("pvscan")
	if err != nil {
		return errors.Wrapf(err, "failed resize pv pvscan %s", out)
	}
	out, err = d.Exec("pvresize", device)
	if err != nil {
		return errors.Wrapf(err, "failed resize pv %s", out)
	}
	return nil
}

func (d *SFsutilDriver) GetVgOfPvDevice(device string) string {
	out, err := d.Exec("pvs", "--noheadings", "-o", "vg_name", device)
	if err != nil {
		log.Errorf("get vg from pv %s device failed: %s %s", device, out, err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GetFsFormat(diskPath string) string {
	fsutilDriver := NewFsutilDriver(driver.NewProcDriver())
	return fsutilDriver.GetFsFormat(diskPath)
}

func (d *SFsutilDriver) GetFsFormat(diskPath string) string {
	ret, err := d.Exec("blkid", "-o", "value", "-s", "TYPE", diskPath)
	if err != nil {
		log.Errorf("failed exec blkid of dev %s: %s, %s", diskPath, err, ret)
		return ""
	}
	var res string
	for _, line := range strings.Split(string(ret), "\n") {
		res += line
	}
	return res
}

func (d *SFsutilDriver) GetDevUuid(dev string) (map[string]string, error) {
	lines, err := d.Exec("blkid", dev)
	if err != nil {
		log.Errorf("GetDevUuid %s error: %v", dev, err)
		return map[string]string{}, errors.Wrapf(err, "blkid")
	}
	for _, l := range strings.Split(string(lines), "\n") {
		if strings.HasPrefix(l, dev) {
			var ret = map[string]string{}
			for _, part := range strings.Split(l, " ") {
				data := strings.Split(part, "=")
				if len(data) == 2 && strings.HasSuffix(data[0], "UUID") {
					if data[1][0] == '"' || data[1][0] == '\'' {
						ret[data[0]] = data[1][1 : len(data[1])-1]
					} else {
						ret[data[0]] = data[1]
					}
				}
			}
			return ret, nil
		}
	}
	return map[string]string{}, nil
}
