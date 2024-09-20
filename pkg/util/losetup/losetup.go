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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/mountutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	LOSETUP_COMMAND = "losetup"
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

func ListDevices() (*Devices, error) {
	cmd, err := NewLosetupCommand().AddArgs("-l", "-O", "NAME,BACK-FILE,SIZELIMIT,RO").Run()
	if err != nil {
		return nil, err
	}
	output := cmd.Output()
	devs, err := parseDevices(output)
	return devs, err
}

func GetUnusedDevice() (string, error) {
	// find first unused device
	cmd, err := NewLosetupCommand().AddArgs("-f").Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(cmd.Output(), "\n"), nil
}

func AttachDevice(filePath string, partScan bool) (*Device, error) {
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
	args = append(args, []string{"-f", filePath}...)
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
	checkMntCmd, _ := NewCommand("sh", "-c", fmt.Sprintf("mount | grep %s | awk '{print $3}'", dev.Name)).Run()
	if out := checkMntCmd.Output(); out != "" {
		mntPoints := strings.Split(out, "\n")
		for _, mntPoint := range mntPoints {
			if mntPoint != "" {
				if err := mountutils.Unmount(mntPoint); err != nil {
					return errors.Wrapf(err, "umount %s of %s", mntPoint, dev.Name)
				}
			}
		}
	}

	_, err = NewLosetupCommand().AddArgs("-d", dev.Name).Run()
	if err != nil {
		return errors.Wrapf(err, "detach device")
	}
	// recheck
	dev, err = getDev()
	if err != nil {
		return errors.Wrapf(err, "get device by %s for rechecking", devPath)
	}
	if dev != nil {
		return errors.Errorf("device %s still exists, %s", devPath, jsonutils.Marshal(dev))
	}
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
