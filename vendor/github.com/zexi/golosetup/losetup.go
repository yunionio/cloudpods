package golosetup

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const (
	LOSETUP_COMMAND = "losetup"
)

type Command struct {
	Path   string
	Args   []string
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewCommand(path string, args ...string) *Command {
	return &Command{
		Path:   path,
		Args:   args,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
}

func (cmd *Command) AddArgs(args ...string) *Command {
	cmd.Args = append(cmd.Args, args...)
	return cmd
}

func (cmd *Command) Run() (*Command, error) {
	ecmd := exec.Command(cmd.Path, cmd.Args...)
	ecmd.Stdout = cmd.stdout
	ecmd.Stderr = cmd.stderr
	err := ecmd.Run()
	if err != nil {
		err = fmt.Errorf("%v: %s", err, cmd.ErrOutput())
	}
	return cmd, err
}

func (cmd *Command) Output() string {
	return cmd.stdout.String()
}

func (cmd *Command) ErrOutput() string {
	return cmd.stderr.String()
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
	devs, err := ListDevices()
	if err != nil {
		return err
	}
	dev := devs.GetDeviceByName(devPath)
	if dev == nil {
		return nil
	}
	_, err = NewLosetupCommand().AddArgs("-d", dev.Name).Run()
	return err
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
