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

package losetup

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/mountutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	LOSETUP_COMMAND = "losetup"
)

var (
	attachDeviceLock = sync.Mutex{}
)

type Command struct {
	Path   string
	Args   []string
	output string
}

func NewCommand(path string, args ...string) *Command {
	return &Command{
		Path:   path,
		Args:   args,
		output: "",
	}
}

func (cmd *Command) AddArgs(args ...string) *Command {
	cmd.Args = append(cmd.Args, args...)
	return cmd
}

func (cmd *Command) Run() (*Command, error) {
	ecmd := procutils.NewRemoteCommandAsFarAsPossible(cmd.Path, cmd.Args...)
	out, err := ecmd.Output()
	if err != nil {
		err = errors.Wrapf(err, "%s %v: %s", cmd.Path, cmd.Args, out)
	}
	cmd.output = string(out)
	return cmd, err
}

func (cmd *Command) Output() string {
	return cmd.output
}

type LosetupCommand struct {
	*Command
}

func NewLosetupCommand() *LosetupCommand {
	return &LosetupCommand{
		Command: NewCommand(LOSETUP_COMMAND),
	}
}

func parseJsonOutput(content string) (*Devices, error) {
	obj, err := jsonutils.ParseString(content)
	if err != nil {
		return nil, errors.Wrapf(err, "parse json: %s", content)
	}
	devs := new(Devices)
	if err := obj.Unmarshal(devs); err != nil {
		return nil, errors.Wrapf(err, "unmarshal json: %s", content)
	}
	return devs, nil
}

func ListDevices() (*Devices, error) {
	cmd, err := NewLosetupCommand().AddArgs("--json").Run()
	errs := make([]error, 0)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "list by json"))
		devs, err2 := listDevicesOldVersion()
		if err2 != nil {
			errs = append(errs, errors.Wrap(err, "list by using old way"))
		} else {
			return devs, nil
		}
		return nil, errors.NewAggregate(errs)
	}
	output := cmd.Output()
	return parseJsonOutput(output)
}

func listDevicesOldVersion() (*Devices, error) {
	cmd, err := NewLosetupCommand().AddArgs("-l", "-O", "NAME,BACK-FILE").Run()
	if err != nil {
		return nil, err
	}
	output := cmd.Output()
	devs, err := parseDevices(output)
	return devs, err
}

/*func GetUnusedDevice() (string, error) {
	// find first unused device
	cmd, err := NewLosetupCommand().AddArgs("-f").Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(cmd.Output(), "\n"), nil
}*/

func AttachDevice(filePath string, partScan bool) (*Device, error) {
	// See man-page: https://man7.org/linux/man-pages/man8/losetup.8.html
	// The loop device setup is not an atomic operation when used with
	// --find, and losetup does not protect this operation by any lock.
	attachDeviceLock.Lock()
	defer attachDeviceLock.Unlock()

	oldDevs, err := ListDevices()
	if err != nil {
		return nil, err
	}
	oldDev := oldDevs.GetDeviceByFile(filePath)
	if oldDev != nil {
		//return nil, fmt.Errorf("file %q already attached to %q, attached twice???", oldDev.BackFile, oldDev.Name)
		return oldDev, nil
	}

	args := []string{}
	if partScan {
		args = append(args, "-P")
	}
	// args = append(args, []string{"--find", "--nooverlap", filePath}...)
	args = append(args, []string{"--find", filePath}...)
	_, err = NewLosetupCommand().AddArgs(args...).Run()
	if err != nil {
		return nil, err
	}
	devs, err := ListDevices()
	if err != nil {
		return nil, err
	}
	dev := devs.GetDeviceByFile(filePath)
	if dev == nil {
		return nil, fmt.Errorf("Not found loop device by file: %s", filePath)
	}
	return dev, nil
}

// converts a raw key value pair string into a map of key value pairs
// example raw string of `foo="0" bar="1" baz="biz"` is returned as:
// map[string]string{"foo":"0", "bar":"1", "baz":"biz"}
func parseKeyValuePairString(propsRaw string) map[string]string {
	// first split the single raw string on spaces and initialize a map of
	// a length equal to the number of pairs
	props := strings.Split(propsRaw, " ")
	propMap := make(map[string]string, len(props))

	for _, kvpRaw := range props {
		// split each individual key value pair on the equals sign
		kvp := strings.Split(kvpRaw, "=")
		if len(kvp) == 2 {
			// first element is the final key, second element is the final value
			// (don't forget to remove surrounding quotes from the value)
			propMap[kvp[0]] = strings.Replace(kvp[1], `"`, "", -1)
		}
	}

	return propMap
}

const (
	// DiskType is a disk type
	DiskType = "disk"
	// SSDType is an sdd type
	SSDType = "ssd"
	// PartType is a partition type
	PartType = "part"
	// CryptType is an encrypted type
	CryptType = "crypt"
	// LVMType is an LVM type
	LVMType = "lvm"
	// MultiPath is for multipath devices
	MultiPath = "mpath"
	// LinearType is a linear type
	LinearType = "linear"
	// LoopType is a loop device type
	LoopType  = "loop"
	sgdiskCmd = "sgdisk"
	// CephLVPrefix is the prefix of a LV owned by ceph-volume
	CephLVPrefix = "ceph--"
	// DeviceMapperPrefix is the prefix of a LV from the device mapper interface
	DeviceMapperPrefix = "dm-"
)

// Partition represents a partition metadata
type Partition struct {
	Name       string
	Size       uint64
	Label      string
	Filesystem string
}

// ref: https://github.com/rook/rook/blob/master/pkg/util/sys/device.go#L135
func GetDevicePartions(devicePath string) ([]string, error) {
	cmd, err := NewCommand("lsblk", devicePath, "--bytes", "--pairs", "--output", "NAME,SIZE,TYPE,PKNAME").Run()
	if err != nil {
		return nil, errors.Wrapf(err, "get device %s partions", devicePath)
	}
	device := filepath.Base(devicePath)
	output := cmd.Output()
	partInfo := strings.Split(output, "\n")
	var partitions = make([]string, 0)
	var totalPartitionSize uint64
	for _, info := range partInfo {
		props := parseKeyValuePairString(info)
		name := props["NAME"]
		if name == device {
			// found the main device
			log.Infof("Device found - %s", name)
			_, err = strconv.ParseUint(props["SIZE"], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to get device %s size. %+v", device, err)
			}
		} else if props["PKNAME"] == device && props["TYPE"] == PartType {
			// found a partition
			p := Partition{Name: name}
			p.Size, err = strconv.ParseUint(props["SIZE"], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to get partition %s size. %+v", name, err)
			}
			totalPartitionSize += p.Size

			partitions = append(partitions, name)
		} else if strings.HasPrefix(name, CephLVPrefix) && props["TYPE"] == LVMType {
			partitions = append(partitions, name)
		}
	}

	return partitions, nil
}

func DetachDevice(devPath string) error {
	getDev := func() (*Device, error) {
		devs, err := ListDevices()
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
	partions, err := GetDevicePartions(devPath)
	if err != nil {
		return errors.Wrapf(err, "get device %s partions", devPath)
	}
	for _, part := range partions {
		checkMntCmd, _ := NewCommand("sh", "-c", fmt.Sprintf("mount | grep %s | awk '{print $3}'", part)).Run()
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

	_, err = NewLosetupCommand().AddArgs("-d", dev.Name).Run()
	if err != nil {
		return errors.Wrapf(err, "detach device")
	}

	// recheck
	/*dev, err = getDev()
	if err != nil {
		return errors.Wrapf(err, "get device by %s for rechecking", devPath)
	}
	if dev != nil {
		return errors.Errorf("device %s still exists, %s", devPath, jsonutils.Marshal(dev))
	}*/
	return nil
}

func DetachDeviceByFile(filePath string) error {
	devs, err := ListDevices()
	if err != nil {
		return err
	}
	dev := devs.GetDeviceByFile(filePath)
	if dev == nil {
		return nil
	}
	_, err = NewLosetupCommand().AddArgs("-d", dev.Name).Run()
	return err
}

func ResizeLoopDevice(loopDev string) error {
	_, err := NewLosetupCommand().AddArgs("-c", loopDev).Run()
	return err
}
