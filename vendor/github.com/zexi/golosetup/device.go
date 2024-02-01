package golosetup

import (
	"fmt"
	"strings"
)

type Device struct {
	Name      string `json:"name"`
	BackFile  string `json:"back-file"`
	SizeLimit bool   `json:"sizelimit"`
	//Offset    string `json:"offset"`
	//AutoClear string `json:"autoclear"`
	ReadOnly bool `json:"ro"`
}

type Devices struct {
	LoopDevs []Device `json:"loopdevices"`
}

/* $ losetup -l -O NAME,BACK-FILE,SIZELIMIT,RO
 * NAME       BACK-FILE                                   SIZELIMIT RO
 * /dev/loop0 /disks/2b917686-2ace-4a57-a4af-44ece2303dd2         0  0
 * /dev/loop1 /disks/033d6bc0-4ce4-48c4-89d3-125077bcc28e         0  0
 * /dev/loop2 /disks/48bcff6e-4062-439c-bc9e-e601391b059f         0  0
 */
func parseDevices(output string) (*Devices, error) {
	devs := &Devices{}
	if len(output) == 0 {
		return devs, nil
	}
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return devs, nil
	}
	loopDevs := make([]Device, 0)
	for _, line := range lines {
		if len(line) == 0 || strings.HasPrefix(line, "NAME") {
			continue
		}
		dev, err := parseDevice(line)
		if err != nil {
			return nil, err
		}
		loopDevs = append(loopDevs, dev)
	}
	devs.LoopDevs = loopDevs
	return devs, nil
}

func parseDevice(line string) (Device, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Device{}, fmt.Errorf("Invalid line: %q", line)
	}
	sizeLimit := false
	ro := false
	if fields[2] != "0" {
		sizeLimit = true
	}
	if fields[3] != "0" {
		ro = true
	}
	return Device{
		Name:      fields[0],
		BackFile:  fields[1],
		SizeLimit: sizeLimit,
		ReadOnly:  ro,
	}, nil
}

func (devs Devices) GetDeviceByName(name string) *Device {
	for _, dev := range devs.LoopDevs {
		if dev.Name == name {
			return &dev
		}
	}
	return nil
}

func (devs Devices) GetDeviceByFile(filePath string) *Device {
	for _, dev := range devs.LoopDevs {
		if dev.BackFile == filePath {
			return &dev
		}
	}
	return nil
}
