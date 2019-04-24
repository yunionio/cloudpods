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

package raid

import (
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type RaidDriverFactory func(term *ssh.Client) IRaidDriver

type sRaidDrivers map[string]RaidDriverFactory

var RaidDrivers sRaidDrivers

func init() {
	RaidDrivers = make(map[string]RaidDriverFactory)
}

func GetCommand(bin string, args ...string) string {
	cmd := []string{bin}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}

func RegisterDriver(name string, drv RaidDriverFactory) {
	RaidDrivers[name] = drv
}

type RaidBasePhyDev struct {
	Adapter int
	Size    int64
	Model   string
	Rotate  tristate.TriState
	Status  string
	Driver  string
}

func NewRaidBasePhyDev(driver string) *RaidBasePhyDev {
	return &RaidBasePhyDev{
		Size:   -1,
		Rotate: tristate.None,
		Driver: driver,
	}
}

func (dev *RaidBasePhyDev) IsComplete() bool {
	if dev.Model == "" {
		return false
	}
	if dev.Rotate.IsNone() {
		return false
	}
	if dev.Status == "" {
		return false
	}
	return true
}

func (dev *RaidBasePhyDev) ToBaremetalStorage() *baremetal.BaremetalStorage {
	return &baremetal.BaremetalStorage{
		Adapter: dev.Adapter,
		Status:  dev.Status,
		Size:    dev.Size,
		Model:   dev.Model,
		Rotate:  dev.Rotate.Bool(),
		Driver:  dev.Driver,
	}
}

func GetModules(term *ssh.Client) []string {
	ret := []string{}
	lines, err := term.Run("/sbin/lsmod")
	if err != nil {
		log.Errorf("Remote lsmod error: %v", err)
		return ret
	}
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		mod := line[:strings.Index(line, " ")]
		if mod != "Module" {
			ret = append(ret, mod)
		}
	}
	return ret
}
