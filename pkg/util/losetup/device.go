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
	if len(fields) < 2 {
		return Device{}, fmt.Errorf("Invalid line: %q", line)
	}
	return Device{
		Name:     fields[0],
		BackFile: strings.Join(fields[1:], " "),
	}, nil
}

func (devs Devices) GetDeviceByName(name string) *Device {
	for i := range devs.LoopDevs {
		dev := &devs.LoopDevs[i]
		if dev.Name == name {
			return dev
		}
	}
	return nil
}

func (devs Devices) GetDeviceByFile(filePath string) *Device {
	for i := range devs.LoopDevs {
		dev := &devs.LoopDevs[i]
		if dev.BackFile == filePath {
			return dev
		}
	}
	return nil
}
